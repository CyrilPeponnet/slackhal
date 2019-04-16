package pluginarchiver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/jinzhu/gorm"

	gormzap "github.com/wantedly/gorm-zap"
	"go.uber.org/zap"

	// Load postgres handler
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
)

// archiver struct define your plugin
type archiver struct {
	plugin.Metadata
	Logger        *zap.Logger
	DB            *gorm.DB
	URL           string
	ChatBot       *ChatBot
	sink          chan<- *plugin.SlackResponse
	configuration *viper.Viper
	Channels      []Channel
	bot           *plugin.Bot
}

// simpleResponse will send a response to the channel it comme from.
func (h *archiver) simpleResponse(message slack.Msg, text string) {
	if text == "" {
		return
	}
	r := new(plugin.SlackResponse)
	r.Channel = message.Channel
	r.Options = append(r.Options, slack.MsgOptionText(text, false))
	r.Options = append(r.Options, slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{UnfurlLinks: true, AsUser: true}))
	h.sink <- r
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *archiver) Init(Logger *zap.Logger, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.Logger = Logger
	h.sink = output
	h.bot = bot
	h.configuration = viper.New()
	h.configuration.AddConfigPath(".")
	h.configuration.SetConfigName("plugin-archiver")
	h.configuration.SetConfigType("yaml")

	err := h.configuration.ReadInConfig()
	if err != nil {
		h.Logger.Error("Not able to read configuration for archiver plugin.", zap.Error(err))
		h.Disabled = true
		return
	}

	host := h.configuration.GetString("Database.host")
	user := h.configuration.GetString("Database.user")
	password := h.configuration.GetString("Database.password")
	database := h.configuration.GetString("Database.database")
	h.URL = h.configuration.GetString("UI.url")

	url := fmt.Sprintf("host=%v user=%v dbname=%v sslmode=disable password=%v", host, user, database, password)
	db, err := gorm.Open("postgres", url)
	db.LogMode(true)
	db.SetLogger(gormzap.New(h.Logger))
	if err != nil {
		h.Logger.Error("Cannot connect to database. Disabling archiver plugin", zap.Error(err))
		h.Disabled = true
		return
	}
	h.DB = db
	// Create or Get the bot
	h.ChatBot = GetChatBotFromDB(db, "slack")
	// Get all chanels.
	h.DB.Find(&h.Channels)
}

// GetMetadata interface implementation
func (h *archiver) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *archiver) ProcessMessage(command string, message slack.Msg) {
	channel := message.Channel
	name := h.bot.GetNameFromID(message.Channel)
	public := true
	if strings.HasPrefix(channel, "G") {
		public = false
	}
	// If there is a command check if a channel is provided as arg.
	// If so then override the vars above
	if command != "" {
		re := regexp.MustCompile(`.*<(\S+\|\S+)>`)
		for _, sub := range re.FindAllStringSubmatch(message.Text, -1) {
			if string(sub[1][0]) == "#" {
				channel = strings.Split(sub[1], "|")[0]
				// remove leading #
				channel = channel[1:len(channel)]
				name = strings.Split(sub[1], "|")[1]
			}
		}
	}
	switch command {
	case cmdlog:
		ch := GetChannelFromDB(h.DB, h.ChatBot.ID, name, channel, public)
		if ch.Status != chanActive {
			h.DB.Model(&ch).Update("status", chanActive)
			h.simpleResponse(message, "Ok I will start logging again activities on that channel.")
		} else {
			h.simpleResponse(message, "Ok I will start logging activities on that channel.")
		}
		h.DB.Find(&h.Channels)

	case cmdnolog:
		ch := GetChannelFromDB(h.DB, h.ChatBot.ID, name, channel, public)
		if ch.Status != chanArchived {
			h.DB.Model(&ch).Update("status", chanArchived)
			h.simpleResponse(message, "Ok I will stop logging activities on that channel.")
		} else {
			h.simpleResponse(message, "Archiving for that channel is already disabled.")
		}
		// Rebuild our channel list
		h.DB.Find(&h.Channels)

	case cmdarchive:
		msg := "This channel doesn't have any archives."
		for _, ch := range h.Channels {
			fmt.Printf("%v - %v", ch.Slug, channel)
			if ch.Slug == channel {
				url := "<" + h.URL + "/" + h.ChatBot.Slug + "/" + ch.Slug + "|archive link>"
				l := "logging"
				if ch.Status != chanActive {
					l = "not logging"
				}
				msg = fmt.Sprintf("Chan is currently %v, here is the %v", l, url)
				break
			}
		}
		h.simpleResponse(message, msg)
	default:
		// Check if the channel accept logging
		for _, ch := range h.Channels {
			if ch.Slug == channel && ch.Status == chanActive {
				if message.User == "" {
					break
				}
				username := h.bot.GetNameFromID(message.User)
				// Replace channel and user id by their names
				re := regexp.MustCompile(`<(\S+)>`)
				for _, sub := range re.FindAllStringSubmatch(message.Text, -1) {
					if string(sub[1][0]) == "@" {
						repl := h.bot.GetNameFromID(sub[1][1:len(sub[1])])
						message.Text = strings.Replace(message.Text, sub[0], repl, -1)
					}
					if string(sub[1][0]) == "#" {
						message.Text = strings.Replace(message.Text, sub[0], "#"+strings.Split(sub[1], "|")[1], -1)
					}
				}
				NewLogToDB(h.DB, h.ChatBot.ID, ch.ID, username, message.Text)
				break
			}
		}
	}

}

// Self interface implementation
func (h *archiver) Self() (i interface{}) {
	return h
}

// Cmds are const for the package.
const (
	cmdlog     = "log"
	cmdnolog   = "no-log"
	cmdarchive = "archive"
)

// init function that will register your plugin to the plugin manager
func init() {
	archiverer := new(archiver)
	archiverer.Metadata = plugin.NewMetadata("archiver")
	archiverer.Description = "Archive channel messages"
	archiverer.ActiveTriggers = []plugin.Command{
		plugin.Command{Name: cmdlog, ShortDescription: "Start to log.", LongDescription: "Will start to log activity on the current channel."},
		plugin.Command{Name: cmdnolog, ShortDescription: "Stop to log.", LongDescription: "Will stop to log activity on the current channel."},
		plugin.Command{Name: cmdarchive, ShortDescription: "Get archive url.", LongDescription: "Get the link for archive of the current channel."}}
	archiverer.PassiveTriggers = []plugin.Command{plugin.Command{Name: `(?s:.*)`, ShortDescription: "Log everything", LongDescription: "Will intercept all messages to log them."}}
	plugin.PluginManager.Register(archiverer)
}
