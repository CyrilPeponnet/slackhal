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
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Jira struct define your plugin
type Jira struct {
	plugin.Metadata
	Logger                  *logrus.Entry
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
func (h *Jira) Init(Logger *logrus.Entry, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.Logger = Logger
	h.sink = output
	h.configuration = viper.New()
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
		for {
			select {
			case event := <-s.IssueEvents:
				for _, msg := range h.ProcessIssueEvent(event) {
					h.sink <- msg
				}
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
				o.Params = &slack.PostMessageParameters{
					Username: "Jira",
					IconURL:  "http://support.zendesk.com/api/v2/apps/4/assets/logo.png",
				}
				o.Params.Attachments = append(o.Params.Attachments, h.CreateAttachement(&event.Issue))
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
		h.Logger.Errorf("Not able to read configuration for jira plugin. (%v)", err)
	} else {
		h.configuration.UnmarshalKey("Notify", &h.projects)
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
	var components []string
	for _, component := range issue.Fields.Components {
		components = append(components, component.Name)
	}
	var days int
	t, _ := time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Created)
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
	return
}

// ProcessMessage interface implementation
func (h *Jira) ProcessMessage(commands []string, message slack.Msg) {
	// Process our entries
	o := new(plugin.SlackResponse)
	o.Channel = message.Channel
	if !h.Connect() {
		o.Text = fmt.Sprintf("Sorry <@%v>, I'm having hard time to reach your jira instance. Please check my logs.", message.User)
	} else {
		o.Params = &slack.PostMessageParameters{
			Username: "Jira",
			IconURL:  "http://support.zendesk.com/api/v2/apps/4/assets/logo.png",
			Text:     fmt.Sprintf("%v is refering to:", message.Username),
		}

		for _, c := range commands {
			// Strip the leading #
			c = strings.ToUpper(c[1:])
			issue, _, err := h.JiraClient.Issue.Get(c, nil)
			if err != nil {
				h.Logger.Debug("An error occurs while fetching an issue ", err)
				continue
			}
			if issue != nil {
				o.Params.Attachments = append(o.Params.Attachments, h.CreateAttachement(issue))
			}
		}
	}
	if len(o.Params.Attachments) > 0 {
		h.sink <- o
	}
	h.JiraClient.Authentication.Logout()
}

// init function that will register your plugin to the plugin manager
func init() {
	myjira := new(Jira)
	myjira.Metadata = plugin.NewMetadata("jira")
	myjira.Description = "Intercept jira bugs IDs."
	myjira.PassiveTriggers = []plugin.Command{plugin.Command{Name: `#([A-Za-z]{3,8}-{0,1}\d{1,10})`, ShortDescription: "Intercept Jira bug Ids", LongDescription: "Will intercept jira bug IDS ans try to fetch some informations."}}
	plugin.PluginManager.Register(myjira)
}

// Connect and authenticate to jira
func (h *Jira) Connect() bool {
	if !h.JiraClient.Authentication.Authenticated() {
		res, err := h.JiraClient.Authentication.AcquireSessionCookie(h.username, h.password)
		if err != nil || !res {
			h.Logger.Errorf("Error while authenticating to jira (%v)", err)
			return false
		}
	}
	return true
}
