package plugin

import (
	"github.com/nlopes/slack"
)

// Bot is the bot structure
type Bot struct {
	API             *slack.Client
	RTM             *slack.RTM
	Name            string
	ID              string
	Tracker         TrackerManager
	cachedChannels  map[string]slack.Channel
	cachedGroups    map[string]slack.Group
	cachedIM        map[string]slack.IM
	cachedUserInfos map[string]slack.User
}

// WarmUpCaches fill the caches
func (s *Bot) WarmUpCaches() {
	_ = s.GetChannelByName("")
	_ = s.GetIMChannelByUser("")
	_ = s.GetGroupByName("")
	_ = s.GetUserInfos("")
}

// GetNameFromID return a name from an ID
func (s *Bot) GetNameFromID(id string) (name string) {
	name = s.getNameFromID(id)
	if name == "" {
		s.WarmUpCaches()
		name = s.getNameFromID(id)

	}
	return name
}

// getNameFromID return a name from an ID using cache
func (s *Bot) getNameFromID(id string) (name string) {
	switch string(id[0]) {
	// Channels
	case "C":
		for _, channel := range s.cachedChannels {
			if channel.ID == id {
				return channel.Name
			}
		}
	// Groups
	case "G":
		for _, group := range s.cachedGroups {
			if group.ID == id {
				return group.Name
			}
		}
	// Users
	case "U", "W":
		for _, user := range s.cachedUserInfos {
			if user.ID == id {
				return user.RealName
			}
		}
	}
	return name
}

// GetChannelByName - Find a channel by its name
func (s *Bot) GetChannelByName(name string) (channel slack.Channel) {
	if s.cachedChannels == nil {
		s.cachedChannels = map[string]slack.Channel{}
	}
	channel = slack.Channel{}
	if it, found := s.cachedChannels[name]; found {
		return it
	}
	// Rebuild the cache
	chans, _ := s.RTM.GetChannels(false)
	s.cachedChannels = map[string]slack.Channel{}
	for _, ch := range chans {
		s.cachedChannels[ch.Name] = ch
		if name == ch.Name {
			channel = ch
		}
	}
	return channel
}

// GetIMChannelByUser - Find a IM channel by username
func (s *Bot) GetIMChannelByUser(user string) (im slack.IM) {
	if s.cachedIM == nil {
		s.cachedIM = map[string]slack.IM{}
	}
	im = slack.IM{}
	if it, found := s.cachedIM[user]; found {
		return it
	}
	// Rebuild the cache
	chans, _ := s.RTM.GetIMChannels()
	for _, ch := range chans {
		s.cachedIM[ch.User] = ch
		if ch.User == user {
			im = ch
		}
	}
	return im
}

// GetGroupByName - Find a group channel by name
func (s *Bot) GetGroupByName(name string) (group slack.Group) {
	if s.cachedGroups == nil {
		s.cachedGroups = map[string]slack.Group{}
	}
	group = slack.Group{}
	if it, found := s.cachedGroups[name]; found {
		return it
	}
	// Rebuild the cache
	chans, _ := s.RTM.GetGroups(false)
	for _, ch := range chans {
		s.cachedGroups[ch.Name] = ch
		if ch.Name == name {
			group = ch
		}
	}
	return group
}

// GetUserInfos - Find user info for a username
func (s *Bot) GetUserInfos(user string) (infos slack.User) {
	if s.cachedUserInfos == nil {
		s.cachedUserInfos = map[string]slack.User{}
	}
	if i, found := s.cachedUserInfos[user]; found {
		return i
	}
	// Rebuild the cache
	users, _ := s.RTM.GetUsers()
	for _, u := range users {
		s.cachedUserInfos[u.ID] = u
		if u.ID == user {
			infos = u
		}

	}
	return infos
}
