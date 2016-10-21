package main

import "github.com/nlopes/slack"

// FindChannelByName - Find a channel by its name
func FindChannelByName(rtm *slack.RTM, name string) *slack.Channel {
	for _, ch := range rtm.GetInfo().Channels {
		if ch.Name == name {
			return &ch
		}
	}
	return nil
}
