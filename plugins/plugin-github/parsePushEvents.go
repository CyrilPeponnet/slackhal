package githubplugin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/CyrilPeponnet/slackhal/plugins/plugin-jira"
	"github.com/google/go-github/github"
	"github.com/nlopes/slack"
)

// ProcessPushEvents will transform an event into a slack message.
func (h *githook) ProcessPushEvents(event *github.PushEvent) (messages []*plugin.SlackResponse) {
	// Filter our event
	branch := *event.Ref
	branch = branch[strings.LastIndex(branch, "/")+1 : len(branch)]
	repodata := FilterRepo(*event.Repo.FullName, branch, h.repos)
	if repodata.Name == "" {
		return
	}
	// Create base response
	message := new(plugin.SlackResponse)

	// Add the PostMessage Parameter
	message.Params = &slack.PostMessageParameters{
		Username: "Github",
		IconURL:  "https://assets-cdn.github.com/images/modules/logos_page/GitHub-Mark.png",
	}
	message.Text = fmt.Sprintf("*%v* just pushed %v commit(s) to *%v:%v*\n> <%v|%v>",
		*event.HeadCommit.Author.Name,
		len(event.Commits),
		*event.Repo.FullName,
		branch,
		*event.HeadCommit.URL,
		*event.HeadCommit.Message)

	// Look if it closed some Jira tickets :)
	// Is the plugin available
	if jp, ok := plugin.PluginManager.Plugins["jira"]; ok {
		info := jp.GetMetadata()
		if !info.Disabled {
			for _, trigger := range info.PassiveTriggers {
				reg, err := regexp.Compile(trigger.Name)
				if err == nil {
					// If we have matches
					matches := reg.FindAllString(*event.HeadCommit.Message, -1)
					for _, m := range matches {
						// HACK: We are forcing the assertion here
						if jc, found := jp.Self().(*jiraplugin.Jira); found {
							if jc.Connect() {
								issue, _, _ := jc.JiraClient.Issue.Get(strings.ToUpper(m[1:]), nil)
								if issue != nil {
									message.Params.Attachments = append(message.Params.Attachments, jc.CreateAttachement(issue))
								}
							}
						}
					}
				}
			}
		}
	}

	if len(message.Params.Attachments) > 1 {
		message.Text += fmt.Sprintf("\n\n:tada: %v looks like you closed some issue today :tada:", *event.HeadCommit.Author.Name)
	}

	// Create a new message per channel we need to notify
	for _, ch := range repodata.Channels {
		n := new(plugin.SlackResponse)
		n.Params = message.Params
		n.Text = message.Text
		n.Channel = ch
		messages = append(messages, n)
	}

	// Add a small comments for the commiter :)

	return
}

// FilterRepo check if the repo is in the filter list,
func FilterRepo(name string, branch string, repos []Repository) (repo Repository) {
	for _, repodata := range repos {
		matched, err := regexp.MatchString(repodata.Name, name)
		if err == nil {
			if matched {
				for _, br := range repodata.Branches {
					if strings.HasSuffix(branch, br) {
						return repodata

					}
				}
			}
		}
	}
	return repo
}
