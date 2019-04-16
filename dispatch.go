package main

import (
	"fmt"
	"regexp"
	"strings"

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

		case msg.Options != nil:

			// Use PostMessage when there is attachments
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

		default:

			// Use RTM as default
			if msg.TrackerID != 0 {
				ts := bot.Tracker.GetTimeStampFor(msg.TrackerID)
				if ts == "" {
					ttl := 300
					if msg.TrackedTTL != 0 {
						ttl = msg.TrackedTTL
					}
					bot.Tracker.Track(plugin.Tracker{TrackerID: msg.TrackerID, TTL: ttl})
				} else {
					_, _, _, err := bot.RTM.UpdateMessage(msg.Channel, ts, msg.Options...)
					if err != nil {
						zap.L().Error("Failed to Update message", zap.Reflect("message", msg.Options), zap.Error(err))

					} else {
						zap.L().Debug("Updated message")
					}
					continue
				}
			}
			_, _, e := bot.RTM.PostMessage(msg.Channel, msg.Options...)
			if e != nil {
				zap.L().Error("Error while sending message", zap.Error(e))
			} else {
				zap.L().Debug("Sent message", zap.Reflect("options", msg.Options))
			}
		}
	}
}

// checkForCommand will try to detect a comamnd in a message
// It will tokenise the message (split by space) for that.
func checkForCommand(text string, command string) bool {

	for _, word := range strings.Split(text, " ") {
		if word == command {
			return true
		}
	}
	return false
}

// DispatchMessage to plugins
func DispatchMessage(prefix string, msg *slack.MessageEvent) {

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
	for _, p := range plugin.PluginManager.Plugins {

		// Get metadata
		info := p.GetMetadata()
		if info.Disabled {
			continue
		}

		func() {
			// Process active triggers
			for _, c := range info.ActiveTriggers {
				if (mentionned && info.WhenMentioned) || !info.WhenMentioned {
					// Look for !action or @bot action or DM with action
					if strings.Contains(message.Text, prefix+c.Name) ||
						(strings.HasPrefix(message.Text, fmt.Sprintf("<@%v> ", bot.ID)) && checkForCommand(message.Text, c.Name)) ||
						(strings.HasPrefix(msg.Channel, "D") && strings.HasPrefix(message.Text, c.Name)) {

						zap.L().Debug("Dispatching to active plugin", zap.String("plugin", info.Name), zap.String("command", c.Name))
						p.ProcessMessage(c.Name, message)

						// stop processing if active is matching
						return
					}
				}
			}

			// Process one or many passive triggers
			for _, r := range info.PassiveTriggers {
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
								p.ProcessMessage(m, message)
							}
						}
					}
				}
			}

		}()

	}

}
