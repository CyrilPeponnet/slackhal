package builtins

import (
	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// logger struct define your plugin
type logger struct {
	plugin.Metadata
	Logger *logrus.Entry
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *logger) Init(Logger *logrus.Entry) {
	h.Logger = Logger
}

// GetMetadata interface implementation
func (h *logger) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *logger) ProcessMessage(commands []string, message slack.Msg, output chan<- *plugin.SlackResponse) {
	h.Logger.Infof("Will log message %v", message.Text)
}

// init function that will register your plugin to the plugin manager
func init() {
	loggerer := new(logger)
	loggerer.Metadata = plugin.NewMetadata("logger")
	loggerer.Description = "Logger messages"
	loggerer.PassiveTriggers = []plugin.Command{plugin.Command{Name: `.*`, ShortDescription: "Log everything", LongDescription: "Will intercept all messages to log them."}}
	plugin.PluginManager.Register(loggerer)
}