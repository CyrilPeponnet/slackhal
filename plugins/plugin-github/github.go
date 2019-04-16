package githubplugin

import (
	"github.com/CyrilPeponnet/slackhal/plugin"
	gws "github.com/f2prateek/github-webhook-server"
	"github.com/fsnotify/fsnotify"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// githook struct define your plugin
type githook struct {
	plugin.Metadata
	Logger        *zap.Logger
	sink          chan<- *plugin.SlackResponse
	repos         []Repository
	configuration *viper.Viper
}

// Repository struct
type Repository struct {
	Name     string
	Branches []string
	Channels []string
}

func (h *githook) ReloadConfiguration() {
	err := h.configuration.ReadInConfig()
	if err != nil {
		h.Logger.Error("Not able to read configuration for github plugin.", zap.Error(err))
	} else {
		if err := h.configuration.UnmarshalKey("Repositories", &h.repos); err != nil {
			h.Logger.Error("Error while reading configuration", zap.Error(err))
		}
	}
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *githook) Init(Logger *zap.Logger, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.Logger = Logger
	h.sink = output
	h.configuration = viper.New()

	// Set our webhook handler
	s := gws.New("")
	h.HTTPHandler[plugin.Command{Name: "/github", ShortDescription: "Github event hook.", LongDescription: "Github event hook."}] = s

	// Read the configuration
	h.configuration.AddConfigPath("/etc/slackhal/")
	h.configuration.AddConfigPath("$HOME/.slackhal")
	h.configuration.AddConfigPath(".")
	h.configuration.SetConfigName("plugin-github")
	h.configuration.SetConfigType("yaml")
	h.ReloadConfiguration()

	// Handle live reload
	h.configuration.WatchConfig()
	h.configuration.OnConfigChange(func(e fsnotify.Event) {
		h.Logger.Info("Reloading github-hook configuration file.")
		h.ReloadConfiguration()
	})

	// Runloop to process incoming events
	go func() {
		for event := range s.PushEvents {
			for _, msg := range h.ProcessPushEvents(event) {
				h.sink <- msg
			}
		}
	}()

}

// Self interface implementation
func (h *githook) Self() (i interface{}) {
	return h
}

// GetMetadata interface implementation
func (h *githook) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *githook) ProcessMessage(command string, message slack.Msg) {
	// Nothing to process
}

// init function that will register your plugin to the plugin manager
func init() {
	githooker := new(githook)
	githooker.Metadata = plugin.NewMetadata("github")
	githooker.Description = "Send github commit notification to channels."
	plugin.PluginManager.Register(githooker)
}
