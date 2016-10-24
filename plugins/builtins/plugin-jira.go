package builtins

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

	"github.com/Sirupsen/logrus"
	jiralib "github.com/andygrunwald/go-jira"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
	"github.com/spf13/viper"
)

// jira struct define your plugin
type jira struct {
	plugin.Metadata
	Logger                  *logrus.Entry
	jiraClient              *jiralib.Client
	url, username, password string
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *jira) Init(Logger *logrus.Entry) {
	h.Logger = Logger
	viper.AddConfigPath(".")
	viper.SetConfigName("plugin-jira")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		h.Logger.Errorf("Not able to read configuration for jira plugin. (%v)", err)
	} else {
		h.url = viper.GetString("server.url")
		h.username = viper.GetString("server.username")
		h.password = viper.GetString("server.password")
	}
	h.jiraClient, _ = jiralib.NewClient(nil, h.url)
}

// GetMetadata interface implementation
func (h *jira) GetMetadata() *plugin.Metadata {
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

// ProcessMessage interface implementation
func (h *jira) ProcessMessage(commands []string, message slack.Msg, output chan<- *plugin.SlackResponse) {
	// Send a premessage because we can
	o := new(plugin.SlackResponse)
	o.Text = fmt.Sprintf("<@%v>, I think you are refering to:", message.User)
	o.Channel = message.Channel
	output <- o
	// Now process our entries
	o = new(plugin.SlackResponse)
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
			c = c[1:len(c)]
			issue, _, _ := h.jiraClient.Issue.Get(strings.ToUpper(c))
			if issue != nil {
				var components []string
				for _, component := range issue.Fields.Components {
					components = append(components, component.Name)
				}
				var days int
				t, _ := time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Created)
				days = int(time.Since(t).Hours() / 24)
				attachement := slack.Attachment{
					Fallback: fmt.Sprintf("%v - %v (%v)", issue.Key, issue.Fields.Summary, issue.Fields.Status.Name),
					Pretext:  fmt.Sprintf("%v days ago, %v reported this issue (%v comments):", days, issue.Fields.Reporter.DisplayName, len(issue.Fields.Comments.Comments)),
					Text:     fmt.Sprintf("*[%v]* <%v/browse/%v|%v>: *%v*", strings.ToUpper(issue.Fields.Status.Name), h.url, issue.Key, issue.Key, issue.Fields.Summary),
					Fields: []slack.AttachmentField{
						slack.AttachmentField{
							Title: "Labels",
							Value: strings.Join(issue.Fields.Labels, ", "),
							Short: true,
						},
						slack.AttachmentField{
							Title: "Priority",
							Value: issue.Fields.Priority.Name,
							Short: true,
						},
						slack.AttachmentField{
							Title: "Components",
							Value: strings.Join(components, ","),
							Short: true,
						},
						slack.AttachmentField{
							Title: "Assignee",
							Value: issue.Fields.Assignee.DisplayName,
							Short: true,
						},
					},
					MarkdownIn: []string{"title", "text", "fields", "fallback"},
					Color:      colorForStatus(issue.Fields.Status.Name),
				}
				o.Params.Attachments = append(o.Params.Attachments, attachement)
			} else {
				attachement := slack.Attachment{
					Fallback: fmt.Sprintf("Sorry %s doesn't seens to be a valid jira issue.", c),
					Pretext:  fmt.Sprintf("Sorry %s doesn't seens to be a valid jira issue.", c),
				}
				o.Params.Attachments = append(o.Params.Attachments, attachement)
			}
		}
	}
	output <- o
}

// init function that will register your plugin to the plugin manager
func init() {
	myjira := new(jira)
	myjira.Metadata = plugin.NewMetadata("jira")
	myjira.Description = "Intercept Jira bugs IDs."
	myjira.PassiveTriggers = []plugin.Command{plugin.Command{Name: `#([A-Za-z]{3,8}-{0,1}\d{1,10})`, ShortDescription: "Intercept Jira bug Ids", LongDescription: "Will intercept jira bug IDS ans try to fetch some informations."}}
	plugin.PluginManager.Register(myjira)
}

// Connect and authenticate to jira
func (h *jira) Connect() bool {
	if !h.jiraClient.Authentication.Authenticated() {
		res, err := h.jiraClient.Authentication.AcquireSessionCookie(h.username, h.password)
		if err != nil || res == false {
			h.Logger.Errorf("Error while authenticating to jira (%v)", err)
			return false
		}
	}
	return true
}
