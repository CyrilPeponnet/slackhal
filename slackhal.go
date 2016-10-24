package main

import (
	"fmt"

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

func main() {
	headline := "Slack HAL bot."
	usage := `

This is another slack bot.

Usage: slackhal [options] [--plugin-path path...]

Options:
	-h, --help              Show this help.
	-t, --token token       The slack bot token to use.
	-f, --file confing		The configuration file to load [default ./slackhal.yml]
	-p, --plugin-path path  The paths to the plugins folder to load [default: ./plugins].
	--trigger char          The char used to detect direct commands [default: !].
	-l, --log level         Set the log level [default: error].
`

	args, _ := docopt.Parse(headline+usage, nil, true, "Slack HAL bot 1.0", true)
	setLogLevel(args["--log"].(string))

	// Load configuraiton file and override some args if needed.

	// Connect to slack and start runloop
	if args["--token"] == nil {
		Log.Fatal("You need to set the slack bot token!")
	}

	api := slack.New(args["--token"].(string))
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	// output channels and start the runloop
	output := make(chan *plugin.SlackResponse)
	go DispatchResponses(output, rtm, api)

	// Loading our plugin and Init them
	for _, p := range plugin.PluginManager.Plugins {
		meta := p.GetMetadata()
		Log.WithField("prefix", "[main]").Infof("Loading plugin %v version %v", meta.Name, meta.Version)
		p.Init(Log.WithField("prefix", fmt.Sprintf("[plugin %v]", meta.Name)))
	}

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {

			case *slack.HelloEvent:
				// Ignore hello

			case *slack.ConnectedEvent:
				Log.WithFields(logrus.Fields{"prefix": "[main]", "Infos": ev.Info, "counter": ev.ConnectionCount}).Debug("Connected with:")
				info := rtm.GetInfo()
				bot.Name = info.User.Name
				bot.ID = info.User.ID
				Log.WithField("prefix", "[main]").Infof("Connected as %v", bot.Name)

			case *slack.MessageEvent:
				Log.WithField("prefix", "[main]").Debugf("Message: %+v", ev)
				// Discard messages comming from myself
				if ev.User == bot.ID {
					continue
				}
				go DispatchMessage(args["--trigger"].(string), &ev.Msg, output)

			case *slack.PresenceChangeEvent:
				// Log.WithField("prefix", "[main]").Debug("Presence Change: %v", ev)

			case *slack.ChannelJoinedEvent:
				// nothing

			case *slack.ChannelLeftEvent:
				// nothing

			case *slack.ReconnectUrlEvent:
				// experimental and not used

			case *slack.LatencyReport:
				Log.WithField("prefix", "[main]").Debugf("Current latency: %v", ev.Value)

			case *slack.RTMError:
				Log.WithField("prefix", "[main]").Errorf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				Log.WithField("prefix", "[main]").Error("Invalid credentials provided!")
				break Loop

			default:
				// ingore other events
				// Log.WithFields(logrus.Fields{"prefix": "[main]", "event": fmt.Sprintf("%+v", msg.Data), "type": fmt.Sprintf("%T", ev)}).Debug("Received:")
			}
		}
	}
}
