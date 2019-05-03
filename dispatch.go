package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/nlopes/slack"
)

// DispatchResponses will process responses from the channel
func DispatchResponses(output chan *plugin.SlackResponse, bot *plugin.Bot) {

	for msg := range output {

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

		case msg.Channel == "":
			zap.L().Warn("No channel found", zap.Reflect("message", msg))

		case msg.Options == nil:
			zap.L().Warn("Nothing to send", zap.Reflect("message", msg))

		default:
			if msg.TrackerID != 0 && bot.Tracker.GetTimeStampFor(msg.TrackerID) != "" {
				ts := bot.Tracker.GetTimeStampFor(msg.TrackerID)
				c, _, _, e := bot.RTM.UpdateMessage(msg.Channel, ts, msg.Options...)
				if e != nil {
					zap.L().Error("Error while updating message", zap.Error(e))
				} else {
					zap.L().Debug("Updated message", zap.String("channel", c))
					// Update the tracker
					bot.Tracker.Track(plugin.Tracker{TrackerID: msg.TrackerID, TimeStamp: ts, TTL: 300})
				}
			} else {
				// Else post message
				c, t, e := bot.RTM.PostMessage(msg.Channel, msg.Options...)
				if e != nil {
					zap.L().Error("Error while sending message", zap.Error(e))
				} else {
					zap.L().Debug("Sent message", zap.String("channel", c))
					// If the message need to be tracked
					if msg.TrackerID != 0 && bot.Tracker.GetTimeStampFor(msg.TrackerID) == "" {
						bot.Tracker.Track(plugin.Tracker{TrackerID: msg.TrackerID, TimeStamp: t, TTL: 300})
					}
				}
			}
		}
	}
}

// checkForCommand will try to detect a command in a message
// It will tokenise the message (split by space) for that.
func checkForCommand(text string, command string) bool {

	for _, word := range strings.Split(text, " ") {
		if strings.ToLower(word) == command {
			return true
		}
	}
	return false
}

// DispatchMessage to plugins
func DispatchMessage(prefix string, msg *slack.MessageEvent, output chan *plugin.SlackResponse) {

	// Check if this is an edited message
	if msg.SubType == "message_changed" {
		msg.Msg.Text = msg.SubMessage.Text
		msg.User = msg.SubMessage.User
	}

	message := msg.Msg

	// mentionned is true id direct message or message contains mention to us
	mentionned := strings.HasPrefix(msg.Channel, "D") || strings.Contains(message.Text, fmt.Sprintf("<@%v>", bot.ID))

	// Process active triggers
	// For each plugins
	func() {

		replied := false
		for _, p := range plugin.PluginManager.Plugins {

			// Get metadata
			info := p.GetMetadata()
			if info.Disabled {
				continue
			}

			// Process active triggers
			for _, c := range info.ActiveTriggers {
				if (mentionned && info.WhenMentioned) || !info.WhenMentioned {
					// Look for !action or @bot action or DM with action
					if strings.Contains(message.Text, prefix+c.Name) ||
						(strings.HasPrefix(message.Text, fmt.Sprintf("<@%v> ", bot.ID)) && checkForCommand(message.Text, c.Name)) ||
						(strings.HasPrefix(msg.Channel, "D") && strings.HasPrefix(strings.ToLower(message.Text), c.Name)) {

						zap.L().Debug("Dispatching to active plugin", zap.String("plugin", info.Name), zap.String("command", c.Name))
						// Replace our prefixed action with the action
						message.Text = strings.Replace(message.Text, prefix+c.Name, c.Name, 1)
						p.ProcessMessage(c.Name, message)

						// stop processing if active is matching
						return
					}
				}
			}

			// Process one or many passive triggers
			for _, r := range info.PassiveTriggers {
				// Check for mention if required by plugin
				if (mentionned && info.WhenMentioned) || !info.WhenMentioned {

					// Check if the user have permissions to use this plugin.
					reg, err := regexp.Compile(r.Name)
					if err != nil {
						zap.L().Error("Passive trigger is not a valid regular expression", zap.String("trigger", r.Name), zap.String("plugin", info.Name))
					} else {
						matches := reg.FindAllString(message.Text, -1)
						if len(matches) > 0 {
							zap.L().Debug("Dispatching to passive plugin", zap.String("trigger", r.Name), zap.String("plugin", info.Name))
							for _, m := range matches {
								replied = p.ProcessMessage(m, message)
							}
						}
					}
				}
			}

		}

		// If I was mentioned or in dm and nothing matched send a reponse
		// From our default reponse list.
		if (mentionned || strings.HasPrefix(msg.Channel, "D")) && !replied {
			rand.Seed(time.Now().Unix())
			o := new(plugin.SlackResponse)
			o.Channel = message.Channel
			o.Options = append(o.Options, slack.MsgOptionText(defaultAnswers[rand.Intn(len(defaultAnswers))], false))
			output <- o
		}

	}()

}
