package builtins

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// cat struct define your plugin
type cat struct {
	plugin.Metadata
	Logger *logrus.Entry
	sink   chan<- *plugin.SlackResponse
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *cat) Init(Logger *logrus.Entry, output chan<- *plugin.SlackResponse) {
	// cats are initless
	h.sink = output
}

// GetMetadata interface implementation
func (h *cat) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *cat) ProcessMessage(commands []string, message slack.Msg) {
	// Cat summoned !
	o := new(plugin.SlackResponse)
	o.Channel = message.Channel
	response, err := http.Get("http://thecatapi.com/api/images/get?format=src&type=gif")
	if err != nil {
		o.Text = "I cannot find a single funny cat picture on Internet... Looks like the ends of the world..."
	} else {
		o.Text = fmt.Sprintf("Hey <@%v> look what I just found for you: %v", message.User, response.Request.URL)
	}
	h.sink <- o
	defer response.Body.Close()
}

// init function that will register your plugin to the plugin manager
func init() {
	cater := new(cat)
	cater.Metadata = plugin.NewMetadata("cat")
	cater.Description = "Show cats"
	cater.ActiveTriggers = []plugin.Command{plugin.Command{Name: `cat`, ShortDescription: "Show a cat.", LongDescription: "Because cats are fun."}}
	plugin.PluginManager.Register(cater)
}
