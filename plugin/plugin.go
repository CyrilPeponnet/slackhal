package plugin

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/nlopes/slack"
)

// Metadata struct
type Metadata struct {
	Name        string
	Description string
	Version     string
	// Active trigers are commands
	ActiveTriggers []Command
	// Passive triggers are regex patterns that will try to get matched
	PassiveTriggers []Command
	// Webhook handler
	HTTPHandler map[Command]http.Handler
	// Only trigger this plugin if the bot is mentionned
	WhenMentioned bool
	// Disabled state
	Disabled bool
}

// Command is a Command implemented by a plugin
type Command struct {
	Name             string
	ShortDescription string
	LongDescription  string
}

// NewMetadata return a new Metadata instance
func NewMetadata(name string) (m Metadata) {
	m.Name = name
	m.Description = fmt.Sprintf("%v's description", name)
	m.Version = "1.0"
	m.WhenMentioned = false
	m.Disabled = false
	m.HTTPHandler = make(map[Command]http.Handler)
	return
}

// SlackResponse struct
type SlackResponse struct {
	Channel    string
	TrackerID  int
	TrackedTTL int
	Options    []slack.MsgOption
}

// Plugin Interface
type Plugin interface {
	Init(Logger *zap.Logger, output chan<- *SlackResponse, bot *Bot)
	GetMetadata() *Metadata
	ProcessMessage(command string, message slack.Msg) bool
	Self() interface{}
}
