package pluginfacts

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// logger struct define your plugin
type facts struct {
	plugin.Metadata
	sink          chan<- *plugin.SlackResponse
	factDB        factStorer
	bot           *plugin.Bot
	configuration *viper.Viper
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *facts) Init(output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.sink = output
	h.bot = bot
	h.configuration = viper.New()
	h.configuration.AddConfigPath("/etc/slackhal/")
	h.configuration.AddConfigPath("$HOME/.slackhal")
	h.configuration.AddConfigPath(".")
	h.configuration.SetConfigName("plugin-facts")
	h.configuration.SetConfigType("yaml")
	err := h.configuration.ReadInConfig()
	if err != nil {
		zap.L().Error("Not able to read configuration for facts plugin.", zap.Error(err))
		h.Disabled = true
		return
	}
	dbPath := h.configuration.GetString("database.path")
	h.factDB = new(stormDB)
	err = h.factDB.Connect(dbPath)
	if err != nil {
		zap.L().Error("Error while opening the facts database!", zap.Error(err))
		h.Disabled = true
		return
	}
}

// GetMetadata interface implementation
func (h *facts) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// simpleResponse will send a response to the channel it comme from.
func (h *facts) simpleResponse(message slack.Msg, text string) {
	if text == "" {
		return
	}
	r := new(plugin.SlackResponse)
	r.Channel = message.Channel
	r.Options = append(r.Options, slack.MsgOptionText(text, false))
	h.sink <- r
}

// ProcessMessage interface implementation
func (h *facts) ProcessMessage(command string, message slack.Msg) bool {

	switch command {
	case cmdNew, cmdUpdate:
		var text string

		if command == cmdNew {
			text = strings.TrimSpace(message.Text[strings.Index(message.Text, cmdNew)+len(cmdNew) : len(message.Text)])
		} else {
			text = strings.TrimSpace(message.Text[strings.Index(message.Text, cmdUpdate)+len(cmdUpdate) : len(message.Text)])
		}

		f := fact{}
		// Split our command in to 4 parts we are looking for tokens AS WHEN and IN
		parts := strings.Split(text, "/as")
		if len(parts) != 2 {
			h.simpleResponse(message, "A fact must have the from `my fact /as my content /when this /or that [/in #chan1 #chan2]")
			return false
		}

		f.Name = strings.TrimSpace(parts[0])

		parts = strings.Split(parts[1], "/when")
		if len(parts) != 2 {
			h.simpleResponse(message, "A fact must have the from `my fact /as my content /when this /or that [/in #chan1 #chan2]")
			return false
		}

		f.Content = strings.TrimSpace(parts[0])

		parts = strings.Split(parts[1], "/in")

		p := strings.Split(parts[0], "/or")
		for _, i := range p {
			f.Patterns = append(f.Patterns, strings.TrimSpace(i))
		}

		if len(parts) == 2 {
			c := h.bot.ExtractFeaturesFromMessage(parts[1])
			for _, i := range c {
				f.RestrictToChannelsID = append(f.RestrictToChannelsID, i.ID)
			}
		}

		if h.factDB.FindFactByName(f.Name) != nil && command == cmdNew {
			h.simpleResponse(message, "I'm afraid I cannot do that. There is already a fact registered with that name.")
			return false
		}

		if err := h.factDB.AddFact(&f); err != nil {
			zap.L().Error("Failed to save fact", zap.Error(err))
			h.simpleResponse(message, "I'm afraid I cannot do that. Something went wrong.")
		}

		h.simpleResponse(message, "Thanks, I will remember that.")

	case cmddel:
		name := strings.TrimSpace(message.Text[strings.Index(message.Text, cmddel)+len(cmddel) : len(message.Text)])
		foundFact := h.factDB.FindFactByName(name)
		if foundFact != nil {
			err := h.factDB.DelFact(name)
			if err != nil {
				h.simpleResponse(message, fmt.Sprintf("Error while deleting this fact (%v)", err))
			} else {
				h.simpleResponse(message, "Ok, I will forget this fact.")
			}
		} else {
			h.simpleResponse(message, fmt.Sprintf("Sorry cannot find a fact with name _%v_", name))
		}

	case cmdlist:

		factsList, err := h.factDB.ListFacts()
		if err != nil {
			zap.L().Error("Error while getting facts", zap.Error(err))
			return false
		}

		content := "Here is the facts I know:\n"

		tpl := `
{{- range .}}
• {{.Name}} */as* {{ .Content }} */when* {{ Join .Patterns " */or* " }} {{- if .RestrictToChannelsID}} */in* {{ range .RestrictToChannelsID}}<#{{.}}> {{ end }} {{- end }}
{{- end}}
`
		t, err := template.New("output").Funcs(template.FuncMap{"Join": strings.Join}).Parse(tpl)
		if err != nil {
			zap.L().Error("Error while parsing template", zap.Error(err))
			return false
		}

		buf := new(bytes.Buffer)
		err = t.Execute(buf, factsList)
		if err != nil {
			zap.L().Error("Error while rendering template", zap.Error(err))
			return false
		}

		h.simpleResponse(message, content+buf.String())

	case cmdremind:
		mentionned := strings.TrimSpace(message.Text[strings.Index(message.Text, cmdremind)+len(cmdremind) : len(message.Text)])
		foundFact := h.factDB.FindFact(message.Text)
		if foundFact != nil {
			if !allowedChan(foundFact, message) {
				h.simpleResponse(message, fmt.Sprintf("Sorry <@%v>, this fact is not allowed in that channel.", message.User))

			} else {
				if foundFact.Content != "" {
					h.simpleResponse(message, mentionned+"\n"+foundFact.Content)
				}
			}
		}
	default:
		foundFact := h.factDB.FindFact(message.Text)
		if foundFact != nil {
			if allowedChan(foundFact, message) {
				if foundFact.Content != "" {
					h.simpleResponse(message, fmt.Sprintf("<@%v>: %v", message.User, foundFact.Content))
					return true
				}
			}
		}
		return false
	}
	return true
}

// allowedChan return if we are in an allowed chan
func allowedChan(f *fact, m slack.Msg) bool {
	if len(f.RestrictToChannelsID) > 0 {
		for _, rc := range f.RestrictToChannelsID {
			if rc == m.Channel {
				return true
			}
		}
	} else {
		return true
	}

	return false
}

// Self interface implementation
func (h *facts) Self() (i interface{}) {
	return h
}

// Cmds are const for the package.
const (
	cmdNew    = "new-fact"
	cmdUpdate = "update-fact"
	cmdlist   = "list-facts"
	cmddel    = "remove-fact"
	cmdremind = "tell-fact"
)

// init function that will register your plugin to the plugin manager
func init() {
	learner := new(facts)
	learner.Metadata = plugin.NewMetadata("facts")
	learner.Description = "Tell facts given patterns."
	learner.ActiveTriggers = []plugin.Command{
		{Name: cmdNew, ShortDescription: "Add a fact.", LongDescription: "Will add a fact must follow the form `new-fact a fact name /as a fact content /when this will trigger /or this will also trigger [/in #chan1 #chan2]`."},
		{Name: cmdUpdate, ShortDescription: "Update a fact.", LongDescription: "Will update a fact must follow the form `new-fact a fact name /as a fact content /when this will trigger /or this will also trigger [/in #chan1 #chan2]`."},
		{Name: cmdlist, ShortDescription: "List all learned facts.", LongDescription: "Will list all the registered facts."},
		{Name: cmdremind, ShortDescription: "Tell someone about a fact.", LongDescription: "Will metion a person with the content of a fact."},
		{Name: cmddel, ShortDescription: "Remove a given fact.", LongDescription: "Allow you to remove a registered fact."}}
	learner.PassiveTriggers = []plugin.Command{{Name: `(?s:.*)`, ShortDescription: "Look for facts", LongDescription: "Will look for registered facts to replay."}}
	plugin.PluginManager.Register(learner)
}
