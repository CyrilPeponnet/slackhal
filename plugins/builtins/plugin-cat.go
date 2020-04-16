package builtins

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/slack-go/slack"
)

// cat struct define your plugin
type cat struct {
	plugin.Metadata
	Logger *zap.Logger
	sink   chan<- *plugin.SlackResponse
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *cat) Init(output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	// cats are initless
	h.sink = output
}

// GetMetadata interface implementation
func (h *cat) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *cat) ProcessMessage(command string, message slack.Msg) bool {
	// Cat summoned !
	o := new(plugin.SlackResponse)
	o.Channel = message.Channel
	response, err := http.Get("http://thecatapi.com/api/images/get?format=src&type=gif")
	if err != nil {
		o.Options = append(o.Options, slack.MsgOptionText("I cannot find a single funny cat picture on Internet... Looks like the ends of the world...", false))
	} else {
		o.Options = append(o.Options, slack.MsgOptionText(fmt.Sprintf("Hey <@%v> look what I just found for you: %v", message.User, response.Request.URL), false))
	}
	h.sink <- o
	defer response.Body.Close() // nolint
	return true
}

// Self interface implementation
func (h *cat) Self() (i interface{}) {
	return h
}

// init function that will register your plugin to the plugin manager
func init() {
	cater := new(cat)
	cater.Metadata = plugin.NewMetadata("cat")
	cater.Description = "Show cats"
	cater.ActiveTriggers = []plugin.Command{{Name: `cat`, ShortDescription: "Show a cat.", LongDescription: "Because cats are fun."}}
	plugin.PluginManager.Register(cater)
}
