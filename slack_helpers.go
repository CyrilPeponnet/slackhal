package main

import "github.com/nlopes/slack"

// FindChannelByName - Find a channel by its name
func FindChannelByName(rtm *slack.RTM, name string) string {
	for _, ch := range rtm.GetInfo().Channels {
		if ch.Name == name {
			return ch.ID
		}
	}
	return ""
}

// FindUserChannel - Find a IM channel by username
func FindUserChannel(api *slack.Client, user string) string {
	chans, _ := api.GetIMChannels()
	for _, ch := range chans {
		if ch.User == user {
			return ch.ID
		}
	}
	return ""
}
