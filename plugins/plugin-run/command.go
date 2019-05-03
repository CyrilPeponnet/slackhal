package runplugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/fsnotify/fsnotify"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// run struct define your plugin
type run struct {
	plugin.Metadata
	bot           *plugin.Bot
	Logger        *zap.Logger
	sink          chan<- *plugin.SlackResponse
	commands      []command
	configuration *viper.Viper
}

// Repository struct
type command struct {
	Name         string
	Description  string
	Command      []string
	AllowedUsers []string
}

func (h *run) ReloadConfiguration() {

	err := h.configuration.ReadInConfig()
	if err != nil {
		h.Logger.Error("Not able to read configuration for run plugin.", zap.Error(err))
	} else {
		if err := h.configuration.UnmarshalKey("Commands", &h.commands); err != nil {
			h.Logger.Error("Error while reading configuration", zap.Error(err))
		}
	}

	// Repopulate our triggers
	h.ActiveTriggers = []plugin.Command{}

	for _, command := range h.commands {

		h.ActiveTriggers = append(h.ActiveTriggers, plugin.Command{
			Name: command.Name,
			ShortDescription: command.Description + func() string {
				if len(command.AllowedUsers) > 0 {
					return fmt.Sprintf(" (restricted to %d users)", len(command.AllowedUsers))
				}
				return ""
			}(),
		})
	}
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *run) Init(Logger *zap.Logger, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {

	h.Logger = Logger
	h.bot = bot
	h.sink = output
	h.configuration = viper.New()

	// Read the configuration
	h.configuration.AddConfigPath("/etc/slackhal/")
	h.configuration.AddConfigPath("$HOME/.slackhal")
	h.configuration.AddConfigPath(".")
	h.configuration.SetConfigName("plugin-run")
	h.configuration.SetConfigType("yaml")
	h.ReloadConfiguration()

	// Handle live reload
	h.configuration.WatchConfig()
	h.configuration.OnConfigChange(func(e fsnotify.Event) {
		h.Logger.Info("Reloading commands configuration file.")
		h.ReloadConfiguration()
	})
}

// Self interface implementation
func (h *run) Self() (i interface{}) {
	return h
}

// GetMetadata interface implementation
func (h *run) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

func (h *run) processCommand(message slack.Msg, cmd command, args []string, user slack.User) {

	var msg string
	var command string

	cargs := args

	if len(cmd.Command) > 0 {
		command = cmd.Command[0]
		args = append(cmd.Command[1:], args...)
	} else {
		command = cmd.Name
	}

	r := new(plugin.SlackResponse)
	r.Channel = message.Channel
	r.Options = append(r.Options, slack.MsgOptionText("thinking...", true), slack.MsgOptionMeMessage())
	// ACK the order while processing
	h.sink <- r

	// Reset the message
	r = new(plugin.SlackResponse)
	r.Channel = message.Channel

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	h.Logger.Debug("Run command", zap.String("command", command), zap.Strings("args", args))
	c := exec.CommandContext(ctx, command, args...)

	// Set some en var for scripts
	c.Env = os.Environ()
	c.Env = append(c.Env, []string{
		fmt.Sprintf("USER=%s", user.Name),
	}...)

	t, err := c.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			msg = fmt.Sprintf("Command `%s %s` timed out and got killed.", cmd.Name, strings.Join(cargs, " "))
		} else {
			msg = fmt.Sprintf("Command `%s %s` failed with error: `%s`.\n Output: ```%s```", cmd.Name, strings.Join(cargs, " "), err, t)
		}
	} else {
		if len(t) == 0 {
			msg = "This is done."
		} else {
			msg = "Here you go: \n```" + string(t) + "```"
		}
	}

	// Upload as file if too big
	if len(msg) > 10000 {

		f := slack.FileUploadParameters{
			Filename: "output.txt",
			Content:  msg,
			Title:    message.Text,
			Channels: []string{message.Channel},
		}

		_, err := h.bot.RTM.UploadFile(f)
		if err != nil {

			h.Logger.Error("Failed to upload file", zap.Error(err))
			r.Options = []slack.MsgOption{slack.MsgOptionText("Uhoh, something went wrong while uploading the file.", false)}

		} else {

			r.Options = []slack.MsgOption{slack.MsgOptionText("Alright, you can see the output above.", false)}

		}

	} else {

		r.Options = []slack.MsgOption{slack.MsgOptionText(msg, false)}
	}
	// Update the previous message
	h.sink <- r
}

// ProcessMessage interface implementation
func (h *run) ProcessMessage(cmd string, message slack.Msg) bool {

	user := h.bot.GetUserInfos(message.User)
	cmdArgs := strings.Split(message.Text, " ")

	// Locate args if any, they are after the command
	for i, arg := range cmdArgs {
		if strings.ToLower(arg) == cmd {
			if i < len(cmdArgs) {
				cmdArgs = cmdArgs[i+1:]
			} else {
				cmdArgs = []string{}
			}
		}
	}

	// Check command ACL
	for _, command := range h.commands {

		if command.Name == cmd {

			if isAuthorized(user, command.AllowedUsers) {
				h.Logger.Debug("Authorized user", zap.String("email", user.Profile.Email))
				h.processCommand(message, command, cmdArgs, user)

			} else {
				h.Logger.Debug("Unathorized user", zap.String("email", user.Profile.Email))
				r := new(plugin.SlackResponse)
				r.Channel = message.Channel
				r.Options = append(r.Options, slack.MsgOptionText("Just what do you think you're doing?", false))
				h.sink <- r
			}

			return true
		}
	}

	return false
}

func isAuthorized(user slack.User, users []string) bool {

	if len(users) > 0 {
		for _, authzUser := range users {
			if authzUser == user.Profile.Email {
				return true
			}
		}
	} else {
		return true
	}
	return false
}

// init function that will register your plugin to the plugin manager
func init() {

	runner := new(run)
	runner.Metadata = plugin.NewMetadata("Runner")
	runner.Description = "Run commands."
	runner.ActiveTriggers = []plugin.Command{plugin.Command{Name: `run`, ShortDescription: "Run a command.", LongDescription: "Run commands."}}
	plugin.PluginManager.Register(runner)
}
