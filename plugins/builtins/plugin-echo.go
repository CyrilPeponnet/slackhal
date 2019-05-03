package builtins

import (
	"strings"

	"go.uber.org/zap"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/nlopes/slack"
)

// echo struct define your plugin
type echo struct {
	plugin.Metadata
	sink chan<- *plugin.SlackResponse
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *echo) Init(Logger *zap.Logger, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.sink = output
}

// GetMetadata interface implementation
func (h *echo) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *echo) ProcessMessage(command string, message slack.Msg) bool {

	if len(strings.Split(message.Text, " ")) == 1 {
		return false
	}

	start := strings.Index(message.Text, command)
	size := len(command)
	if start+size+1 >= len(message.Text) {
		return false
	}
	msg := message.Text[start+size+1 : len(message.Text)]

	o := new(plugin.SlackResponse)
	o.Options = append(o.Options, slack.MsgOptionText(msg, false))
	o.Channel = message.Channel
	// This is a test to implement tracking of message
	o.TrackerID = 42
	h.sink <- o
	return true
}

// Self interface implementation
func (h *echo) Self() (i interface{}) {
	return h
}

// init function that will register your plugin to the plugin manager
func init() {
	echoer := new(echo)
	echoer.Metadata = plugin.NewMetadata("echo")
	echoer.Description = "Will repeat what you said"
	echoer.ActiveTriggers = []plugin.Command{plugin.Command{Name: "echo", ShortDescription: "Parrot style", LongDescription: "Will repeat what you put after."}}
	plugin.PluginManager.Register(echoer)
}
