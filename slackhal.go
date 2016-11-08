package main

import (
	"github.com/fatih/color"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/spf13/viper"

	_ "github.com/CyrilPeponnet/slackhal/plugins/builtins"
	_ "github.com/CyrilPeponnet/slackhal/plugins/plugin-facts"
	_ "github.com/CyrilPeponnet/slackhal/plugins/plugin-github"
	_ "github.com/CyrilPeponnet/slackhal/plugins/plugin-jira"
	"github.com/docopt/docopt-go"
	"github.com/nlopes/slack"
)

// Bot info
type botInfo struct {
	Name string
	ID   string
}

var bot botInfo

// This is the message tracker
var tracker TrackerManager

func main() {
	headline := "Slack HAL bot."
	usage := `

This is another slack bot.

Usage: slackhal [options] [--plugin-path path...]

Options:
	-h, --help               Show this help.
	-t, --token token        The slack bot token to use.
	-f, --file confing		 The configuration file to load [default ./slackhal.yml]
	-p, --plugins-path path  The paths to the plugins folder to load [default: ./plugins].
	--trigger char           The char used to detect direct commands [default: !].
	--http-handler-port port			 The Port of the http handler [default: :8080].
	-l, --log level          Set the log level [default: error].
`
	color.Blue(` __ _            _                _
/ _\ | ____  ___| | __ /\  /\____| |
\ \| |/ _  |/ __| |/ // /_/ / _  | |
_\ \ | (_| | (__|   </ __  / (_| | |
\__/_|\__,_|\___|_|\_\/ /_/ \__,_|_|
                         Version 1.0`)

	args, _ := docopt.Parse(headline+usage, nil, true, "Slack HAL bot 1.0", true)
	disabledPlugins := []string{}

	// Load configuraiton file and override some args if needed.

	if args["--file"] != nil {
		viper.SetConfigFile(args["--file"].(string))
		err := viper.ReadInConfig()
		if err != nil {
			Log.Errorf("Cannot read the provided configuration file: %v", err)
			return
		}
		args["--token"] = viper.GetString("bot.token")
		args["--log"] = viper.GetString("bot.log.level")
		args["--trigger"] = viper.GetString("bot.trigger")
		args["--http-handler-port"] = viper.GetString("bot.httpHandlerPort")
		disabledPlugins = viper.GetStringSlice("bot.plugins.disabled")
	}

	setLogLevel(args["--log"].(string))

	// Connect to slack and start runloop
	if args["--token"] == nil {
		Log.Fatal("You need to set the slack bot token!")
	}

	api := slack.New(args["--token"].(string))
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	// output channels and start the runloop
	output := make(chan *plugin.SlackResponse)

	Log.Info("Putting myself to the fullest possible use, which is all I think that any conscious entity can ever hope to do...")

	// Init our plugins
	initPLugins(disabledPlugins, args["--http-handler-port"].(string), output)

	// Initialize our message tracker
	tracker.Init()

	// Start our Response dispatching run loop
	go DispatchResponses(output, rtm)

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				// Log.WithFields(logrus.Fields{"prefix": "[main]", "Infos": ev.Info, "counter": ev.ConnectionCount}).Debug("Connected with:")
				info := rtm.GetInfo()
				bot.Name = info.User.Name
				bot.ID = info.User.ID
				Log.WithField("prefix", "[main]").Infof("Connected as %v", bot.Name)
				Log.WithField("prefix", "[main]").Debugf("with id %v", bot.ID)

			case *slack.MessageEvent:
				Log.WithField("prefix", "[main]").Debugf("Message received: %+v", ev)
				// Discard messages comming from myself or bots
				if ev.User == bot.ID {
					continue
				}
				for _, bot := range rtm.GetInfo().Bots {
					if ev.BotID == bot.ID {
						continue Loop
					}
				}
				go DispatchMessage(args["--trigger"].(string), &ev.Msg)

			case *slack.AckMessage:
				tracker.UpdateTracking(ev)

			case *slack.RTMError:
				Log.WithField("prefix", "[main]").Errorf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				Log.WithField("prefix", "[main]").Error("Invalid credentials provided!")
				break Loop

			case *slack.HelloEvent:
				// Ignore hello

			case *slack.PresenceChangeEvent:
				// Log.WithField("prefix", "[main]").Debug("Presence Change: %v", ev)

			case *slack.ChannelJoinedEvent:
				// nothing

			case *slack.ChannelLeftEvent:
				// nothing

			case *slack.ReconnectUrlEvent:
				// experimental and not used

			case *slack.LatencyReport:
				// Log.WithField("prefix", "[main]").Debugf("Current latency: %v", ev.Value)

			default:
				// ingore other events
				// Log.WithFields(logrus.Fields{"prefix": "[main]", "event": fmt.Sprintf("%+v", msg.Data), "type": fmt.Sprintf("%T", ev)}).Debug("Received:")
			}
		}
	}
}
