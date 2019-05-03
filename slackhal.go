package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/CyrilPeponnet/slackhal/pkg/logutils"
	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/docopt/docopt-go"
	"github.com/fatih/color"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"

	_ "github.com/CyrilPeponnet/slackhal/plugins/builtins"
	// _ "github.com/CyrilPeponnet/slackhal/plugins/plugin-archiver"
	_ "github.com/CyrilPeponnet/slackhal/plugins/plugin-facts"
	// _ "github.com/CyrilPeponnet/slackhal/plugins/plugin-github"
	// _ "github.com/CyrilPeponnet/slackhal/plugins/plugin-jira"
	_ "github.com/CyrilPeponnet/slackhal/plugins/plugin-run"
)

var bot plugin.Bot

var defaultAnswers = []string{"Sorry, I'm not sure what you mean by that."}

func main() {

	headline := "Slack HAL bot."
	usage := `

This is another slack bot.

Usage: slackhal [options] [--plugin-path path...]

Options:
	-h, --help               Show this help.
	-t, --token token        The slack bot token to use.
	-f, --file config        The configuration file to load [default ./slackhal.yml]
	--trigger char           The char used to detect direct commands [default: !].
	--http-handler-port port The Port of the http handler [default: :8080].
	--log-level level        Set the log level [default: error].
	--log-format format      Set the log format [default: console].
`
	color.Blue(` __ _            _                _
/ _\ | ____  ___| | __ /\  /\____| |
\ \| |/ _  |/ __| |/ // /_/ / _  | |
_\ \ | (_| | (__|   </ __  / (_| | |
\__/_|\__,_|\___|_|\_\/ /_/ \__,_|_|
                         Version 2.0

`)

	args, _ := docopt.Parse(headline+usage, nil, true, "Slack HAL bot 1.0", true)
	disabledPlugins := []string{}

	// Load configuration file and override some args if needed.

	if args["--file"] != nil {

		viper.AddConfigPath("/etc/slackhal/")
		viper.AddConfigPath("$HOME/.slackhal")
		viper.AddConfigPath(".")
		viper.SetConfigFile(args["--file"].(string))

		err := viper.ReadInConfig()
		if err != nil {
			panic(fmt.Sprintf("Cannot read the provided configuration file: %v", err))
		}

		viper.SetDefault("bot.token", args["--token"])
		viper.SetDefault("bot.log.level", args["--log-level"])
		viper.SetDefault("bot.log.format", args["--log-format"])
		viper.SetDefault("bot.trigger", args["--trigger"])
		viper.SetDefault("bot.httpHandlerPort", args["--http-handler-port"])

		disabledPlugins = viper.GetStringSlice("bot.plugins.disabled")
	}

	logutils.ConfigureWithOptions(viper.GetString("bot.log.level"), viper.GetString("bot.log.format"), "", false, false)

	// Connect to slack and start runloop
	if viper.GetString("bot.token") == "nil" {
		zap.L().Fatal("You need to set the slack bot token!")
	}

	bot.API = slack.New(viper.GetString("bot.token"))
	bot.RTM = bot.API.NewRTM()

	go bot.RTM.ManageConnection()

	// output channels and start the runloop
	output := make(chan *plugin.SlackResponse)

	zap.L().Info("Putting myself to the fullest possible use, which is all I think that any conscious entity can ever hope to do...")

	// Init our plugins
	initPlugins(disabledPlugins, viper.GetString("bot.httpHandlerPort"), output, &bot)

	// Initialize our message tracker
	bot.Tracker.Init()

	// Start our Response dispatching run loop
	go DispatchResponses(output, &bot)

Loop:
	for msg := range bot.RTM.IncomingEvents {

		switch ev := msg.Data.(type) {

		case *slack.ConnectedEvent:
			// Log.WithFields(logrus.Fields{"prefix": "[main]", "Infos": ev.Info, "counter": ev.ConnectionCount}).Debug("Connected with:")
			info := bot.RTM.GetInfo()
			bot.Name = info.User.Name
			bot.ID = info.User.ID
			zap.L().Info("Connected", zap.String("name", bot.Name), zap.String("id", bot.ID))
			zap.L().Debug("Warming up caches for group and users.")
			bot.WarmUpCaches()

		case *slack.MessageEvent:
			zap.L().Debug("Message event received", zap.Reflect("event", ev))
			// Discard messages coming from myself or bots
			if ev.SubType == "bot_message" || ev.User == bot.ID {
				continue
			}
			if ev.SubType == "message_changed" {
				if ev.SubMessage.SubType == "bot_message" || ev.SubMessage.User == bot.ID {
					continue
				}
			}

			go DispatchMessage(viper.GetString("bot.trigger"), ev, output)

		case *slack.AckMessage:
			bot.Tracker.UpdateTracking(ev)

		case *slack.RTMError:
			zap.L().Error("RTM error", zap.String("error", ev.Error()))

		case *slack.InvalidAuthEvent:
			zap.L().Error("Invalid credentials provided!")
			break Loop

		case *slack.HelloEvent:
			// Ignore hello

		case *slack.PresenceChangeEvent:
			// zap.L().Debug("Presence Change: %v", ev)

		case *slack.ChannelJoinedEvent:
			// nothing

		case *slack.ChannelLeftEvent:
			// nothing

		case *slack.ReconnectUrlEvent:
			// experimental and not used

		case *slack.LatencyReport:
			// zap.L().Debugf("Current latency: %v", ev.Value)

		case *slack.ReactionAddedEvent:
			// if reaction added on our message

		case *slack.ReactionRemovedEvent:
			// If reaction removed on our message

		default:
			// ingore other events
			zap.L().Debug("event", zap.Reflect("data", msg.Data))
		}
	}
}
