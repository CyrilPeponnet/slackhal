package plugin

import (
	"fmt"

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
	// Only trigger this plugin if the bot is mentionned
	WhenMentionned bool
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
	Init(Logger *logrus.Entry)
	GetMetadata() *Metadata
	ProcessMessage(commands []string, message slack.Msg, output chan<- *SlackResponse)
}
