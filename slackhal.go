package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/slackhal/plugin"

	"github.com/Sirupsen/logrus"
	"github.com/docopt/docopt-go"
	"github.com/nlopes/slack"
	_ "github.com/slackhal/plugins/builtins"
)

// Bot info
type botInfo struct {
	Name string
	ID   string
}

var bot botInfo

// Send a message accoring to what's need to be sent.
func Send(m *slack.Msg, r *plugin.SlackResponse, rtm *slack.RTM, api *slack.Client) {
	// If channed id is not set, then set it as where it came from.
	if r.ChannelID == "" {
		r.ChannelID = m.Channel
	}
	if r.Params != nil {
		c, t, e := api.PostMessage(r.ChannelID, r.Text, *r.Params)
		if e != nil {
			log.Errorf("Error while sending message %v", e)
		} else {
			log.Debugf("Send message %v to %v at %v", r.Text, c, t)
		}
	} else {
		msg := slack.OutgoingMessage{Channel: r.ChannelID, Text: r.Text, Type: "message"}
		rtm.SendMessage(&msg)
	}
}

func main() {
	headline := "Slack HAL bot."
	usage := `

This is another slack bot.

Usage: zabbix-autohost [options] [--plugin-path path...]

Options:
	-h, --help              Show this help.
	-t, --token token       The slack bot token to use [default: xoxb-91603848178-q19vBaxCqfUQPm2kNQ9hlvWv].
	-p, --plugin-path path  The paths to the plugins folder to load [default: ./plugins].
	--trigger char          The char used to detect direct commands [default: !].
	-l, --log level         Set the log level [default: debug].
`
	args, _ := docopt.Parse(headline+usage, nil, true, "Slack HAL bot 1.0", true)
	setLogLevel(args["--log"].(string))

	api := slack.New(args["--token"].(string))
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	// Loading our plugin if needed
	for _, p := range plugin.PluginManager.Plugins {
		log.Infof("Loading plugin %v version %v", p.GetMetadata().Name, p.GetMetadata().Version)
		p.Init()
	}

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {

			case *slack.HelloEvent:
				// Ignore hello

			case *slack.ConnectedEvent:
				log.WithFields(logrus.Fields{"Infos": ev.Info, "counter": ev.ConnectionCount}).Debug("Connected with:")
				info := rtm.GetInfo()
				bot.Name = info.User.Name
				bot.ID = info.User.ID
				log.Infof("Connected as %v", bot.Name)

			case *slack.MessageEvent:
				log.Debugf("Message: %+v", ev)
				// Discard messages comming from myself
				if ev.User == bot.ID {
					continue
				}
				go func() {
					// Process our plugins
					// TODO: If we need to use atachement we need to use the api.PostMessage.

					// Action are always the first word and must starts with a !
					// @mention + command without the ! works too
					// DM with ! or not but first word are considered as actions too

					commandPrefix := args["--trigger"].(string)
					mentionned := strings.HasPrefix(ev.Msg.Channel, "D") || strings.Contains(ev.Msg.Text, fmt.Sprintf("<@%v>", bot.ID))

					// Process active triggers
					for _, p := range plugin.PluginManager.Plugins {
						info := p.GetMetadata()
						for _, c := range info.ActiveTriggers {
							if (mentionned && info.WhenMentionned) || !info.WhenMentionned {
								// Look for !action
								if strings.Contains(ev.Msg.Text, commandPrefix+c.Name) ||
									// Look for @bot action
									strings.HasPrefix(ev.Msg.Text, fmt.Sprintf("<@%v> ", bot.ID)+c.Name) ||
									// Look for DM with action
									(strings.HasPrefix(ev.Msg.Channel, "D") && strings.HasPrefix(ev.Msg.Text, c.Name)) {
									response, err := p.ProcessMessage([]string{c.Name}, &ev.Msg)
									if err == nil && response != nil {
										Send(&ev.Msg, response, rtm, api)
									}
								}
							}
						}
						// Process passive triggers
						for _, r := range info.PassiveTriggers {
							if (mentionned && info.WhenMentionned) || !info.WhenMentionned {
								reg, err := regexp.Compile(r.Name)
								if err != nil {
									log.Errorf("Passive trigger %v for %v is not a valid regular expression.", r, info.Name)
								} else {
									matches := reg.FindAllString(ev.Msg.Text, -1)
									if len(matches) > 0 {
										response, err := p.ProcessMessage(matches, &ev.Msg)
										if err == nil && response != nil {
											Send(&ev.Msg, response, rtm, api)
										}
									}

								}
							}
						}
					}
				}()

			case *slack.PresenceChangeEvent:
				log.Debug("Presence Change: %v", ev)

			case *slack.ChannelJoinedEvent:
				// nothing

			case *slack.ChannelLeftEvent:
				// nothing

			case *slack.ReconnectUrlEvent:
				// experimental and not used

			case *slack.LatencyReport:
				log.Debugf("Current latency: %v", ev.Value)

			case *slack.RTMError:
				log.Errorf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				log.Error("Invalid credentials provided!")
				break Loop

			default:
				// ingore other events
				log.WithFields(logrus.Fields{"event": fmt.Sprintf("%+v", msg.Data), "type": fmt.Sprintf("%T", ev)}).Debug("Received:")
			}
		}
	}
}
