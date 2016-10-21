package builtins

import (
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// jiratrigger struct define your plugin
type jiratrigger struct {
	plugin.Metadata
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *jiratrigger) Init() {
	// Nothing to do
}

// GetMetadata interface implementation
func (h *jiratrigger) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *jiratrigger) ProcessMessage(cmds []string, m slack.Msg) (o *plugin.SlackResponse, e error) {
	o = new(plugin.SlackResponse)
	o.Text = "I found some jira issue there "
	for _, c := range cmds {
		o.Text += c + " "
	}
	return
}

// init function that will register your plugin to the plugin manager
func init() {
	jiratriggerer := new(jiratrigger)
	jiratriggerer.Metadata = plugin.NewMetadata("jiratrigger")
	jiratriggerer.Description = "Intercept Jira bugs IDs."
	jiratriggerer.PassiveTriggers = []plugin.Command{plugin.Command{Name: `#([A-Za-z]{3,8}-{0,1}\d{1,10})`, ShortDescription: "Intercept Jira bug Ids", LongDescription: "Will intercept jira bug IDS ans try to fetch some informations."}}
	plugin.PluginManager.Register(jiratriggerer)
}
