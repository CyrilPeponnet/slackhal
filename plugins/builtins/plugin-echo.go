package builtins

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// echo struct define your plugin
type echo struct {
	plugin.Metadata
	sink chan<- *plugin.SlackResponse
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *echo) Init(Logger *logrus.Entry, output chan<- *plugin.SlackResponse) {
	h.sink = output
}

// GetMetadata interface implementation
func (h *echo) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *echo) ProcessMessage(commands []string, message slack.Msg) {
	for _, c := range commands {
		if c == "echo" {
			o := new(plugin.SlackResponse)
			o.Text = strings.Replace(message.Text, c+" ", "", 1)
			o.Channel = message.Channel
			h.sink <- o
		}
	}
}

// init function that will register your plugin to the plugin manager
func init() {
	echoer := new(echo)
	echoer.Metadata = plugin.NewMetadata("echo")
	echoer.Description = "Will repeat what you said"
	echoer.ActiveTriggers = []plugin.Command{plugin.Command{Name: "echo", ShortDescription: "Parrot style", LongDescription: "Will repeat what you put after."}}
	plugin.PluginManager.Register(echoer)
}
