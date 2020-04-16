package plugin

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/karlseguin/ccache"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// Bot is the bot structure
type Bot struct {
	API              *slack.Client
	RTM              *slack.RTM
	Name             string
	ID               string
	Tracker          TrackerManager
	cachedUserInfos  *ccache.Cache
	cachedUserChans  *ccache.Cache
	cachedChanInfos  *ccache.Cache
	cachedGroupInfos *ccache.Cache
}

// FeatureType represent a feature
type FeatureType int

// Represent Types
const (
	TypePublicChannel FeatureType = iota
	TypePrivateChannel
	TypeGroup
	TypeUser
	TypeLink
)

// MessageFeature is a message feature
type MessageFeature struct {
	Type  FeatureType
	Value interface{}
	ID    string
}

// String representation of a feature
func (m MessageFeature) String() string {
	switch m.Type {
	case TypePublicChannel:
		return fmt.Sprintf("<#%s>", m.ID)
	case TypePrivateChannel:
		return fmt.Sprintf("<#%s>", m.Value.(slack.Channel).ID)
	case TypeGroup:
		return fmt.Sprintf("<!subteam^%s>", m.ID)
	case TypeUser:
		return fmt.Sprintf("<@%s>", m.ID)
	case TypeLink:
		return fmt.Sprintf("<%s|%s>", m.ID, m.Value.(string))
	}
	return ""
}

// ExtractFeaturesFromMessage extract feature from a message
func (s *Bot) ExtractFeaturesFromMessage(message string) (features []MessageFeature) {
	r := regexp.MustCompile(`<(.*?)>`)
	matches := r.FindAllStringSubmatch(message, -1)
	for _, m := range matches {
		switch {
		// Channel
		case strings.HasPrefix(m[1], "#C"):
			c := strings.Split(m[1], "|")[0]
			ci, err := s.GetCachedChanInfos(c[1:])
			if err != nil {
				continue
			}
			features = append(features, MessageFeature{
				Type:  TypePublicChannel,
				Value: ci,
				ID:    ci.ID,
			})
			// User
		case strings.HasPrefix(m[1], "@U") || strings.HasPrefix(m[1], "@W"):
			u := strings.Split(m[1], "|")[0]
			ui, err := s.GetCachedUserInfos(u[1:])
			if err != nil {
				continue
			}
			features = append(features, MessageFeature{
				Type:  TypeUser,
				Value: ui,
				ID:    ui.ID,
			})
			// Group
		case strings.HasPrefix(m[1], "!subteam^"):
			g := strings.Split(m[1], "|")[0]
			gi, err := s.GetCachedGroupInfos(g[9:])
			if err != nil {
				continue
			}
			features = append(features, MessageFeature{
				Type:  TypeGroup,
				Value: gi,
				ID:    g[9:],
			})
		// Special
		case strings.HasPrefix(m[1], "!"):
			// regular link
		default:
			l := strings.Split(m[1], "|")
			if len(l) == 2 {
				features = append(features, MessageFeature{
					Type:  TypeLink,
					Value: l[1],
					ID:    l[0],
				})
			}
		}
	}

	// Remove what matched
	message = r.ReplaceAllString(message, "")

	// Need to extract the private chan we may have set like #private-chan
	// For that we need to check the chan of the bot. If the chan is in here we can create a features.
	r = regexp.MustCompile(`#(\S+)`)
	matches = r.FindAllStringSubmatch(message, -1)

	if len(matches) > 0 {
		// Try to retrieve private info through the bot chan list
		myChans, err := s.GetCachedUserChans(s.ID)
		if err != nil {
			return features
		}

		for _, m := range matches {
			for _, c := range myChans {
				if c.Name == m[1] {
					features = append(features, MessageFeature{
						Type:  TypePrivateChannel,
						Value: c,
						ID:    c.ID,
					})
				}
			}
		}
	}

	return features
}

// GetCachedUserChans retrieve the list of chans the user belongs to
func (s *Bot) GetCachedUserChans(user string) ([]slack.Channel, error) {

	if s.cachedUserChans == nil {
		s.cachedUserChans = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(100))
	}

	items := s.cachedUserChans.Get(user)
	var chans []slack.Channel

	// If not set build it
	if items == nil {

		for {
			p := slack.GetConversationsForUserParameters{
				UserID:          user,
				Types:           []string{"public_channel,private_channel"},
				Limit:           0,
				ExcludeArchived: true,
			}

			currentChans, n, err := s.API.GetConversationsForUser(&p)

			if err != nil {
				if rateLimitedError, ok := err.(*slack.RateLimitedError); ok {
					zap.L().Debug("Reach rate limit on conversation.list, will backoff", zap.Error(err))
					select {
					case <-time.After(rateLimitedError.RetryAfter):
						continue
					}
				} else {
					zap.L().Error("Error while getting the chans", zap.Error(err))
				}
			}

			chans = append(chans, currentChans...)
			if n == "" {
				s.cachedUserChans.Set(user, chans, 24*time.Hour)
				return chans, nil
			}

			p.Cursor = n

		}

	}

	if chans, ok := items.Value().([]slack.Channel); ok {
		return chans, nil
	}
	zap.L().Error("Error while casting cache to []slack.Channels")
	return nil, fmt.Errorf("Error while casting cache to []slack.Channels")

}

// MemberOf tell if a user is member of a channel
func (s *Bot) MemberOf(channel, user string) bool {

	chans, err := s.GetCachedUserChans(user)
	if err != nil {
		return false
	}

	for _, c := range chans {
		if channel == c.ID {
			return true
		}
	}

	return false

}

// GetCachedUserInfos - Find user info for a username
func (s *Bot) GetCachedUserInfos(user string) (slack.User, error) {
	if s.cachedUserInfos == nil {
		s.cachedUserInfos = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(100))
	}
	item := s.cachedUserInfos.Get(user)

	// if item is nil get it from API
	if item == nil {
		infos, err := s.API.GetUserInfo(user)
		if err != nil {
			zap.L().Error("Error while getting user info", zap.String("user", user), zap.Error(err))
			return slack.User{}, err
		}
		s.cachedUserInfos.Set(infos.ID, infos, 24*time.Hour)
		return *infos, nil
	}

	if infos, ok := item.Value().(*slack.User); ok {
		return *infos, nil
	}

	return slack.User{}, fmt.Errorf("Cannot cast cache item to slack.User")

}

// GetCachedChanInfos - Find user info for a chan
func (s *Bot) GetCachedChanInfos(channel string) (slack.Channel, error) {
	if s.cachedChanInfos == nil {
		s.cachedChanInfos = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(100))
	}
	item := s.cachedChanInfos.Get(channel)

	// if item is nil get it from API
	if item == nil {
		infos, err := s.API.GetConversationInfo(channel, true)
		if err != nil {
			zap.L().Error("Error while getting channel info", zap.String("channel", channel), zap.Error(err))
			return slack.Channel{}, err
		}
		s.cachedChanInfos.Set(infos.ID, infos, 24*time.Hour)
		return *infos, nil
	}

	if infos, ok := item.Value().(*slack.Channel); ok {
		return *infos, nil
	}

	return slack.Channel{}, fmt.Errorf("Cannot cast cache item to slack.Channel")

}

// GetCachedGroupInfos - Find user info for a group
func (s *Bot) GetCachedGroupInfos(group string) ([]string, error) {
	if s.cachedGroupInfos == nil {
		s.cachedGroupInfos = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(100))
	}
	members := s.cachedGroupInfos.Get(group)

	// if members is nil get it from API
	if members == nil {
		infos, err := s.API.GetUserGroupMembers(group)
		if err != nil {
			zap.L().Error("Error while getting group info", zap.String("group", group), zap.Error(err))
			return nil, err
		}
		s.cachedGroupInfos.Set(group, infos, 24*time.Hour)
		return infos, nil
	}

	if infos, ok := members.Value().([]string); ok {
		return infos, nil
	}

	return nil, fmt.Errorf("Cannot cast cache members to slack.UserGroup")

}
