package main

import "github.com/nlopes/slack"

var cachedChannels map[string]string
var cachedUsers map[string]string

// FindChannelByName - Find a channel by its name
func FindChannelByName(rtm *slack.RTM, name string) (id string) {
	if cachedChannels == nil {
		cachedChannels = map[string]string{}
	}
	id = ""
	if i, found := cachedChannels[name]; found {
		return i
	}
	// Rebuild the cache
	for _, ch := range rtm.GetInfo().Channels {
		cachedChannels[ch.Name] = ch.ID
		if name == ch.Name {
			id = ch.ID
		}
	}
	return id
}

// FindUserChannel - Find a IM channel by username
func FindUserChannel(rtm *slack.RTM, user string) (id string) {
	if cachedUsers == nil {
		cachedUsers = map[string]string{}
	}
	id = ""
	if i, found := cachedUsers[user]; found {
		return i
	}
	// Rebuild the cache
	chans, _ := rtm.GetIMChannels()
	for _, ch := range chans {
		cachedUsers[ch.User] = ch.ID
		if ch.User == user {
			id = ch.ID
		}
	}
	return id
}
