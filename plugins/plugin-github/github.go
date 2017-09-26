package githubplugin

import (
	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/Sirupsen/logrus"
	"github.com/f2prateek/github-webhook-server"
	"github.com/fsnotify/fsnotify"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
)

// githook struct define your plugin
type githook struct {
	plugin.Metadata
	Logger        *logrus.Entry
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
		h.Logger.Errorf("Not able to read configuration for github plugin. (%v)", err)
	} else {
		h.configuration.UnmarshalKey("Repositories", &h.repos)
	}
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *githook) Init(Logger *logrus.Entry, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.Logger = Logger
	h.sink = output
	h.configuration = viper.New()

	// Set our webhook handler
	s := gws.New("")
	h.HTTPHandler[plugin.Command{Name: "/github", ShortDescription: "Github event hook.", LongDescription: "Github event hook."}] = s

	// Read the configuration
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
		for {
			select {
			case event := <-s.PushEvents:
				for _, msg := range h.ProcessPushEvents(event) {
					h.sink <- msg
				}
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
func (h *githook) ProcessMessage(commands []string, message slack.Msg) {
	// Nothing to process
}

// init function that will register your plugin to the plugin manager
func init() {
	githooker := new(githook)
	githooker.Metadata = plugin.NewMetadata("githook")
	githooker.Description = "Send github commit notification to channels."
	plugin.PluginManager.Register(githooker)
}
