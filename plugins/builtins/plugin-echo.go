package builtins

import (
	"strings"

	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// echo struct define your plugin
type echo struct {
	plugin.Metadata
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *echo) Init() {
	// Nothing to do
}

// GetMetadata interface implementation
func (h *echo) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *echo) ProcessMessage(cmds []string, m *slack.Msg) (o *plugin.SlackResponse, e error) {
	for _, c := range cmds {
		if c == "echo" {
			o = new(plugin.SlackResponse)
			o.Text = strings.Replace(m.Text, c+" ", "", 1)
		}
	}
	return
}

// init function that will register your plugin to the plugin manager
func init() {
	echoer := new(echo)
	echoer.Metadata = plugin.NewMetadata("echo")
	echoer.Description = "Will repeat what you said"
	echoer.ActiveTriggers = []plugin.Command{plugin.Command{Name: "echo", ShortDescription: "Parrot style", LongDescription: "Will repeat what you put after."}}
	plugin.PluginManager.Register(echoer)
}
