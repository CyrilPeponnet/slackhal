package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// DispatchResponses will process reponses from the channel
func DispatchResponses(output chan *plugin.SlackResponse, rtm *slack.RTM, api *slack.Client) {
	for {
		select {
		case msg := <-output:
			if strings.HasPrefix(msg.Channel, "U") {
				msg.Channel = FindUserChannel(api, msg.Channel)
			}
			switch {
			case msg.Text == "" && msg.Params == nil:
				Log.Warnf("Nothing to send for message %v", msg)
			case msg.Channel == "":
				Log.Warnf("No channel found for message %v", msg)
			case msg.Params != nil:
				c, t, e := api.PostMessage(msg.Channel, msg.Text, *msg.Params)
				if e != nil {
					Log.Errorf("Error while sending message %v", e)
				} else {
					Log.Debugf("Sent message %v to %v at %v", msg.Text, c, t)
				}

			default:
				msg := slack.OutgoingMessage{Channel: msg.Channel, Text: msg.Text, Type: "message"}
				rtm.SendMessage(&msg)
				Log.Debugf("Sent message %v", msg)
			}

		}
	}
}

// DispatchMessage to plugins
func DispatchMessage(prefix string, msg *slack.Msg, output chan<- *plugin.SlackResponse) {
	mentionned := strings.HasPrefix(msg.Channel, "D") || strings.Contains(msg.Text, fmt.Sprintf("<@%v>", bot.ID))

	// Process active triggers
	for _, p := range plugin.PluginManager.Plugins {
		info := p.GetMetadata()
		if info.Disabled {
			continue
		}
		for _, c := range info.ActiveTriggers {
			if (mentionned && info.WhenMentionned) || !info.WhenMentionned {
				// Look for !action
				if strings.Contains(msg.Text, prefix+c.Name) ||
					// Look for @bot action
					(strings.HasPrefix(msg.Text, fmt.Sprintf("<@%v> ", bot.ID)) && strings.Contains(msg.Text, c.Name)) ||
					// Look for DM with action
					(strings.HasPrefix(msg.Channel, "D") && strings.Contains(msg.Text, c.Name)) {
					// Check if the user have permissions to use this plugin.
					Log.WithFields(logrus.Fields{"prefix": "[main]", "Command": c.Name, "Plugin": info.Name}).Debug("Dispatching to plugin")
					p.ProcessMessage([]string{c.Name}, *msg, output)
				}
			}
		}
		// Process passive triggers
		for _, r := range info.PassiveTriggers {
			if (mentionned && info.WhenMentionned) || !info.WhenMentionned {
				reg, err := regexp.Compile(r.Name)
				if err != nil {
					Log.WithField("prefix", "[main]").Errorf("Passive trigger %v for %v is not a valid regular expression.", r.Name, info.Name)
				} else {
					matches := reg.FindAllString(msg.Text, -1)
					if len(matches) > 0 {
						Log.WithFields(logrus.Fields{"prefix": "[main]", "Trigger": r.Name, "Plugin": info.Name}).Debug("Dispatching to plugin")
						p.ProcessMessage(matches, *msg, output)
					}

				}
			}
		}
	}

}
