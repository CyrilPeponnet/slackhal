package builtins

import (
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// help struct define your plugin
type help struct {
	plugin.Metadata
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *help) Init() {
	// Nothing to do
}

// GetMetadata interface implementation
func (h *help) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *help) ProcessMessage(cmds []string, m *slack.Msg) (o *plugin.SlackResponse, e error) {
	for _, c := range cmds {
		if c == "help" {
			o = new(plugin.SlackResponse)
			o.Text = "zob"
		}
	}
	return
}

// init function that will register your plugin to the plugin manager
func init() {
	helper := new(help)
	helper.Metadata = plugin.NewMetadata("help")
	helper.ActiveTriggers = []plugin.Command{plugin.Command{Name: "help"}}
	plugin.PluginManager.Register(helper)
}
