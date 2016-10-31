package pluginfacts

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
	"github.com/spf13/viper"
)

// logger struct define your plugin
type facts struct {
	plugin.Metadata
	Logger        *logrus.Entry
	sink          chan<- *plugin.SlackResponse
	learner       *learn
	factDB        factStorer
	configuration *viper.Viper
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *facts) Init(Logger *logrus.Entry, output chan<- *plugin.SlackResponse) {
	h.Logger = Logger
	h.sink = output
	h.learner = new(learn)
	h.configuration = viper.New()
	h.configuration.AddConfigPath(".")
	h.configuration.SetConfigName("plugin-facts")
	h.configuration.SetConfigType("yaml")
	err := h.configuration.ReadInConfig()
	if err != nil {
		h.Logger.Errorf("Not able to read configuration for facts plugin. (%v)", err)
		h.Disabled = true
		return
	}
	dbPath := h.configuration.GetString("database.path")
	h.factDB = new(stormDB)
	err = h.factDB.Connect(dbPath)
	if err != nil {
		h.Logger.Errorf("Error while opening the facts database! (%v)", err)
		h.Disabled = true
		return
	}
}

// GetMetadata interface implementation
func (h *facts) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// simpleResponse will send a reponse to the channel it comme from.
func (h *facts) simpleResponse(message slack.Msg, text string) {
	if text == "" {
		return
	}
	r := new(plugin.SlackResponse)
	r.Channel = message.Channel
	r.Text = text
	h.sink <- r
}

// ProcessMessage interface implementation
func (h *facts) ProcessMessage(commands []string, message slack.Msg) {
	for _, cmd := range commands {
		switch cmd {
		case cmdnew:
			h.simpleResponse(message, h.learner.New(message))
		case cmdcancel:
			h.simpleResponse(message, h.learner.Cancel(message))
		case cmdlist:
			factsList := h.factDB.ListFacts()
			content := "Here is the facts I know:\n"
			for _, f := range factsList {
				content += fmt.Sprintf(">%v\n", f.Name)
			}
			h.simpleResponse(message, content)
		case cmdremind:
			mentionned := strings.TrimSpace(message.Text[strings.Index(message.Text, cmdremind)+len(cmdremind) : len(message.Text)])
			foundFact := h.factDB.FindFact(message.Text)
			if foundFact != nil {
				if (foundFact.OnlyInChan == "" || foundFact.OnlyInChan == message.Channel) && foundFact.Content != "" {
					h.simpleResponse(message, mentionned+"\n"+foundFact.Content)
				}
			}
		default:
			foundFact := h.factDB.FindFact(message.Text)
			if foundFact != nil {
				if (foundFact.OnlyInChan == "" || foundFact.OnlyInChan == message.Channel) && foundFact.Content != "" {
					h.simpleResponse(message, fmt.Sprintf("<@%v>: %v", message.User, foundFact.Content))
				}
			}
			// continue learning if any
			f, r := h.learner.Learn(message)
			h.simpleResponse(message, r)
			if f.Name != "" {
				h.factDB.AddFact(&f)
				h.simpleResponse(message, fmt.Sprintf("I now know %v facts.", h.factDB.NumberOfFacts()))
			}
		}
	}
}

// Self interface implementation
func (h *facts) Self() (i interface{}) {
	return h
}

// Cmds are const for the pacakge.
const (
	cmdnew    = "new-fact"
	cmdcancel = "stop-learning"
	cmdlist   = "list-fact"
	cmdedit   = "edit-fact"
	cmddel    = "remove-fact"
	cmdremind = "tell-fact"
)

// init function that will register your plugin to the plugin manager
func init() {
	learner := new(facts)
	learner.Metadata = plugin.NewMetadata("facts")
	learner.Description = "Logger messages"
	learner.ActiveTriggers = []plugin.Command{
		plugin.Command{Name: cmdnew, ShortDescription: "Start a learning session", LongDescription: "Will start a learning session to add new facts."},
		plugin.Command{Name: cmdcancel, ShortDescription: "Stop a learning session", LongDescription: "Will stop a current learning session"},
		plugin.Command{Name: cmdlist, ShortDescription: "List all learned facts", LongDescription: "Will list all the registered facts."},
		plugin.Command{Name: cmdremind, ShortDescription: "Tell someone about a facit.", LongDescription: "Will metion a person with the content of a fact."},
		plugin.Command{Name: cmdedit, ShortDescription: "Edit a given fact", LongDescription: "Allow you to edit registered facts."},
		plugin.Command{Name: cmddel, ShortDescription: "Remove a give fact", LongDescription: "Allow you to remove a registered fact."}}
	learner.PassiveTriggers = []plugin.Command{plugin.Command{Name: `.*`, ShortDescription: "Look for facts", LongDescription: "Will look for registered facts to replay."}}
	plugin.PluginManager.Register(learner)
}
