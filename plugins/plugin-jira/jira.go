package jiraplugin

/* This is a jira plugin. A configuration file is needed in the same folder as the binary in yaml format.

plugin-jira.yaml

server:
  url: <jira url>
  username: <user>
  password: <password

*/

import (
	"fmt"
	"strings"
	"time"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/andygrunwald/go-jira"
	"github.com/fsnotify/fsnotify"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Jira struct define your plugin
type Jira struct {
	plugin.Metadata
	Logger                  *zap.Logger
	JiraClient              *jira.Client
	url, username, password string
	sink                    chan<- *plugin.SlackResponse
	configuration           *viper.Viper
	projects                []Project
}

// Project struct
type Project struct {
	Name     string
	Channels []string
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *Jira) Init(Logger *zap.Logger, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.Logger = Logger
	h.sink = output
	h.configuration = viper.New()
	h.configuration.AddConfigPath("/etc/slackhal/")
	h.configuration.AddConfigPath("$HOME/.slackhal")
	h.configuration.AddConfigPath(".")
	h.configuration.SetConfigName("plugin-jira")
	h.configuration.SetConfigType("yaml")
	h.ReloadConfiguration()

	// Handle live reload
	h.configuration.WatchConfig()
	h.configuration.OnConfigChange(func(e fsnotify.Event) {
		h.Logger.Info("Reloading jira configuration file.")
		h.ReloadConfiguration()
	})

	h.url = h.configuration.GetString("Server.url")
	h.username = h.configuration.GetString("Server.username")
	h.password = h.configuration.GetString("Server.password")
	h.JiraClient, _ = jira.NewClient(nil, h.url)

	// Jira webhook handler
	s := newEventHandler()
	h.HTTPHandler[plugin.Command{Name: "/jira", ShortDescription: "Jira issue event hook.", LongDescription: "Will trap new issue created and send a notification to channels."}] = s

	// Runloop to process incoming events
	go func() {
		for event := range s.IssueEvents {
			for _, msg := range h.ProcessIssueEvent(event) {
				h.sink <- msg
			}
		}
	}()

}

// ProcessIssueEvent from webhooks
func (h *Jira) ProcessIssueEvent(event *jiraEvent) (responses []*plugin.SlackResponse) {
	// Extract project name
	project := event.Issue.Fields.Project.Name
	for _, p := range h.projects {
		if p.Name == project {
			for _, c := range p.Channels {
				o := new(plugin.SlackResponse)
				o.Channel = c
				o.Options = append(o.Options, slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
					Username: "Jira",
					IconURL:  "http://support.zendesk.com/api/v2/apps/4/assets/logo.png",
				}))
				o.Options = append(o.Options, slack.MsgOptionAttachments(h.CreateAttachement(&event.Issue)))
				responses = append(responses, o)
			}

		}
	}
	return responses
}

// ReloadConfiguration reload the configuration on changes
func (h *Jira) ReloadConfiguration() {
	err := h.configuration.ReadInConfig()
	if err != nil {
		h.Logger.Error("Not able to read configuration for jira plugin. ", zap.Error(err))
	} else {
		err = h.configuration.UnmarshalKey("Notify", &h.projects)
		if err != nil {
			h.Logger.Error("Error unmarshalling configuration", zap.Error(err))
		}
	}
}

// Self interface implementation
func (h *Jira) Self() (i interface{}) {
	return h
}

// GetMetadata interface implementation
func (h *Jira) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

func colorForStatus(status string) (color string) {
	switch strings.ToLower(status) {
	case "open":
		return "danger"
	case "resolved", "closed", "fixed":
		return "good"
	default:
		return "warning"
	}
}

// CreateAttachement from a give issue.
func (h *Jira) CreateAttachement(issue *jira.Issue) (attachement slack.Attachment) {

	var days int
	t, err := time.Parse("2006-01-02T15:04:05.999-0700", fmt.Sprint(issue.Fields.Created))
	if err != nil {
		t = time.Now()
	}

	days = int(time.Since(t).Hours() / 24)
	timeText := fmt.Sprintf("%v days ago, %v", days, issue.Fields.Reporter.DisplayName)
	if days == 0 {
		timeText = fmt.Sprintf("%v just", issue.Fields.Reporter.DisplayName)
	}

	attachement = slack.Attachment{
		Fallback: fmt.Sprintf("%v - %v (%v)", issue.Key, issue.Fields.Summary, issue.Fields.Status.Name),
		Pretext:  fmt.Sprintf("%v reported this issue (%v comments):", timeText, len(issue.Fields.Comments.Comments)),
		Text:     fmt.Sprintf("*[%v]* <%v/browse/%v|%v>: *%v*", strings.ToUpper(issue.Fields.Status.Name), h.url, issue.Key, issue.Key, issue.Fields.Summary),
		Fields: []slack.AttachmentField{
			// slack.AttachmentField{
			// 	Title: "Labels",
			// 	Value: strings.Join(issue.Fields.Labels, ", "),
			// 	Short: true,
			// },
			slack.AttachmentField{
				Title: "Priority",
				Value: issue.Fields.Priority.Name,
				Short: true,
			},
			// slack.AttachmentField{
			// 	Title: "Components",
			// 	Value: strings.Join(components, ","),
			// 	Short: true,
			// },
			slack.AttachmentField{
				Title: "Assignee",
				Value: issue.Fields.Assignee.DisplayName,
				Short: true,
			},
		},
		MarkdownIn: []string{"title", "text", "fields", "fallback"},
		Color:      colorForStatus(issue.Fields.Status.Name),
	}
	return attachement
}

// ProcessMessage interface implementation
func (h *Jira) ProcessMessage(command string, message slack.Msg) {
	// Process our entries
	o := new(plugin.SlackResponse)
	o.Channel = message.Channel
	if !h.Connect() {
		o.Options = append(o.Options, slack.MsgOptionText(fmt.Sprintf("Sorry <@%v>, I'm having hard time to reach your jira instance. Please check my logs.", message.User), false))
	} else {
		o.Options = append(o.Options, slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
			Username: "Jira",
			IconURL:  "http://support.zendesk.com/api/v2/apps/4/assets/logo.png",
		}))

		o.Options = append(o.Options, slack.MsgOptionText(fmt.Sprintf("%v is refering to:", message.Username), false))

		// Strip the leading #
		command = strings.ToUpper(command)
		issue, _, err := h.JiraClient.Issue.Get(command, nil)
		if err != nil {
			h.Logger.Debug("An error occurs while fetching an issue ", zap.Error(err))
			return
		}
		if issue != nil {
			o.Options = append(o.Options, slack.MsgOptionAttachments(h.CreateAttachement(issue)))
		}
	}
	if len(o.Options) > 0 {
		h.sink <- o
	}
	err := h.JiraClient.Authentication.Logout() //nolint
	if err != nil {
		h.Logger.Error("Error while logging out", zap.Error(err))
	}
}

// init function that will register your plugin to the plugin manager
func init() {
	myjira := new(Jira)
	myjira.Metadata = plugin.NewMetadata("jira")
	myjira.Description = "Intercept jira bugs IDs."
	myjira.PassiveTriggers = []plugin.Command{plugin.Command{Name: `#([A-Za-z]{2,8}-{0,1}\d{1,10})`, ShortDescription: "Intercept Jira bug Ids", LongDescription: "Will intercept jira bug IDS ans try to fetch some informations."}}
	plugin.PluginManager.Register(myjira)
}

// Connect and authenticate to jira
func (h *Jira) Connect() bool {
	if !h.JiraClient.Authentication.Authenticated() {
		res, err := h.JiraClient.Authentication.AcquireSessionCookie(h.username, h.password) // nolint
		if err != nil || !res {
			h.Logger.Error("Error while authenticating to jira", zap.Error(err))
			return false
		}
	}
	return true
}
