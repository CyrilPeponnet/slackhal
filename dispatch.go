package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
)

// DispatchResponses will process reponses from the channel
func DispatchResponses(output chan *plugin.SlackResponse, bot *plugin.Bot) {
	for {
		select {
		case msg := <-output:
			if strings.HasPrefix(msg.Channel, "U") {
				msg.Channel = bot.GetIMChannelByUser(msg.Channel).ID
			} else if strings.HasPrefix(msg.Channel, "#") {
				// try chan
				channel := bot.GetChannelByName(msg.Channel[1:len(msg.Channel)])
				id := channel.ID
				if id == "" {
					channel := bot.GetGroupByName(msg.Channel[1:len(msg.Channel)])
					id = channel.ID
				}
				msg.Channel = id
			}
			switch {
			case msg.Text == "" && msg.Params == nil:
				Log.Warnf("Nothing to send for message %v", msg)
			case msg.Channel == "":
				Log.Warnf("No channel found for message %v", msg)
			case msg.Params != nil:
				// Use PostMessage when there is attachments
				if msg.TrackerID != 0 && bot.Tracker.GetTimeStampFor(msg.TrackerID) != "" {
					ts := bot.Tracker.GetTimeStampFor(msg.TrackerID)
					// TODO: The library is not yet handling Attachments update
					// So we are only updating the Text
					c, t, _, e := bot.RTM.UpdateMessage(msg.Channel, ts, msg.Text)
					if e != nil {
						Log.Errorf("Error while updating message %v", e)
					} else {
						Log.Debugf("Updated message %v to %v at %v", msg.Text, c, t)
						// Update the tracker
						bot.Tracker.Track(plugin.Tracker{TrackerID: msg.TrackerID, TimeStamp: ts, TTL: 300})
					}
				} else {
					// Else post message
					c, t, e := bot.RTM.PostMessage(msg.Channel, msg.Text, *msg.Params)
					if e != nil {
						Log.Errorf("Error while sending message %v", e)
					} else {
						Log.Debugf("Sent message %v to %v at %v", msg.Text, c, t)
						// If the message need to be tracked
						if msg.TrackerID != 0 && bot.Tracker.GetTimeStampFor(msg.TrackerID) == "" {
							bot.Tracker.Track(plugin.Tracker{TrackerID: msg.TrackerID, TimeStamp: t, TTL: 300})
						}
					}
				}

			default:
				// Use RTM for  as default
				tosend := slack.OutgoingMessage{Channel: msg.Channel, Text: msg.Text, Type: "message"}
				if msg.TrackerID != 0 {
					ts := bot.Tracker.GetTimeStampFor(msg.TrackerID)
					if ts == "" {
						ttl := 300
						if msg.TrackedTTL != 0 {
							ttl = msg.TrackedTTL
						}
						bot.Tracker.Track(plugin.Tracker{TrackerID: msg.TrackerID, TTL: ttl})
						tosend.ID = msg.TrackerID
					} else {
						_, _, _, err := bot.RTM.UpdateMessage(tosend.Channel, ts, tosend.Text)
						if err != nil {
							Log.Debugf("Failed to Update message %v: %v", tosend.Text, err)

						} else {
							Log.Debugf("Updated message %v", tosend.Text)
						}
						continue
					}
				}
				bot.RTM.SendMessage(&tosend)
				Log.Debugf("Sent message %v", tosend)

			}

		}
	}
}

// checkForCommand will try to detect a comamnd in a message
// It will tokenise the message (split by splace) for that.
func checkForCommand(text string, command string) bool {
	for _, word := range strings.Split(text, " ") {
		if word == command {
			return true
		}
	}
	return false
}

// DispatchMessage to plugins
func DispatchMessage(prefix string, msg *slack.Msg) {
	mentionned := strings.HasPrefix(msg.Channel, "D") || strings.Contains(msg.Text, fmt.Sprintf("<@%v>", bot.ID))

	// Process active triggers
loop:
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
					(strings.HasPrefix(msg.Text, fmt.Sprintf("<@%v> ", bot.ID)) && checkForCommand(msg.Text, c.Name)) ||
					// Look for DM with action
					(strings.HasPrefix(msg.Channel, "D") && strings.HasPrefix(msg.Text, c.Name)) {
					// Check if the user have permissions to use this plugin.
					Log.WithFields(logrus.Fields{"prefix": "[main]", "Command": c.Name, "Plugin": info.Name}).Debug("Dispatching to plugin")
					p.ProcessMessage([]string{c.Name}, *msg)
					// don't process others
					continue loop
				}
			}
		}
		// Process passive triggers
		for _, r := range info.PassiveTriggers {
			if (mentionned && info.WhenMentionned) || !info.WhenMentionned {
				// Check if the user have permissions to use this plugin.
				reg, err := regexp.Compile(r.Name)
				if err != nil {
					Log.WithField("prefix", "[main]").Errorf("Passive trigger %v for %v is not a valid regular expression.", r.Name, info.Name)
				} else {
					matches := reg.FindAllString(msg.Text, -1)
					if len(matches) > 0 {
						Log.WithFields(logrus.Fields{"prefix": "[main]", "Trigger": r.Name, "Plugin": info.Name}).Debug("Dispatching to plugin")
						p.ProcessMessage(matches, *msg)
						continue loop
					}
				}
			}
		}
	}

}
