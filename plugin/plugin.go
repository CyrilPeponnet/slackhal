package plugin

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
)

// Metadata struct
type Metadata struct {
	Name        string
	Description string
	Version     string
	// Active trigers are commands
	ActiveTriggers []Command
	// Passive triggers are regex parterns that will try to get matched
	PassiveTriggers []Command
	// Webhook handler
	HTTPHandler map[Command]http.Handler
	// Only trigger this plugin if the bot is mentionned
	WhenMentionned bool
	// Disabled state
	Disabled bool
	// self
	Self interface{}
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
	m.WhenMentionned = false
	m.Disabled = false
	m.HTTPHandler = make(map[Command]http.Handler)
	return
}

// SlackResponse struct
type SlackResponse struct {
	Channel string
	Text    string
	Params  *slack.PostMessageParameters
}

// Plugin Interface
type Plugin interface {
	Init(Logger *logrus.Entry, output chan<- *SlackResponse)
	GetMetadata() *Metadata
	ProcessMessage(commands []string, message slack.Msg)
}
