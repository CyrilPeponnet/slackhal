# SLACKAHL - Yet another slack bot in golang.

## TODO

[] Use a config file
[] allow to disable plugins
[] permission plugin

## Usage

```
Usage: slackhal [options] [--plugin-path path...]

Options:
	-h, --help              Show this help.
	-t, --token token       The slack bot token to use.
	-p, --plugin-path path  The paths to the plugins folder to load [default: ./plugins].
	--trigger char          The char used to detect direct commands [default: !].
	-l, --log level         Set the log level [default: error].
```

# Plugins

## Internal plugins

You plugin must implement the following interfaces:

```go
Init(Logger *logrus.Entry)
GetMetadata() *Metadata
ProcessMessage(commands []string, message slack.Msg, output chan<- *SlackResponse)
```

### The `Init` function

Will be called upon plugin login. You can use it if you need to init some stuff. You can use the `Logrus.Entry` as a logger factory for your plugin.

### The `GetMetadata` function

Must return a `plugin.Metadata` struct. This is used to register you plugin.

### The `ProcessMessage` function

Will be called when an action or a trigger is found in an incoming message. You can check `slack.Msg` type [here](https://godoc.org/github.com/nlopes/slack#Msg).

To send your reponse you can send `*plugin.SlackResponse` instances to the `output` channel provided.


### Registering your plugin

The registration process is done in the `init` function

```go
func init() {
	loggerer := new(logger)
	loggerer.Metadata = plugin.NewMetadata("logger")
	loggerer.Description = "Logger messages"
	loggerer.PassiveTriggers = []plugin.Command{plugin.Command{Name: `.*`, ShortDescription: "Log everything", LongDescription: "Will intercept all messages to log them."}}
	plugin.PluginManager.Register(loggerer)
}
```

This is where you will initialise the `plugin.Metadata` struct and add your commands / triggers. It must end with a call to `plugin.PluginManager.Register()` function call to load you plugin.

### Package consideration

If you are not creating your plugin under the `buitin` package you will need to update `slackhal.go` to import your module like:

```go
_ "github.com/slackhal/plugins/myplugin"
```

where `myplugin` is your package name (`package myplugin` as first line in your source code files).

### Example:

```go
package builtins

import (
	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// logger struct define your plugin
type logger struct {
	plugin.Metadata
	Logger *logrus.Entry
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *logger) Init(Logger *logrus.Entry) {
	h.Logger = Logger
}

// GetMetadata interface implementation
func (h *logger) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *logger) ProcessMessage(commands []string, message slack.Msg, output chan<- *plugin.SlackResponse) {
	h.Logger.Infof("Will log message %v", message.Text)
}

// init function that will register your plugin to the plugin manager
func init() {
	loggerer := new(logger)
	loggerer.Metadata = plugin.NewMetadata("logger")
	loggerer.Description = "Logger messages"
	loggerer.PassiveTriggers = []plugin.Command{plugin.Command{Name: `.*`, ShortDescription: "Log everything", LongDescription: "Will intercept all messages to log them."}}
	plugin.PluginManager.Register(loggerer)
}
```

## Plugin behaviour

From the defined Metadata struct:

```go
// Metadata struct
type Metadata struct {
	Name        string
	Description string
	Version     string
	// Active trigers are commands
	ActiveTriggers []Command
	// Passive triggers are regex parterns that will try to get matched
	PassiveTriggers []Command
	// Only trigger this plugin if the bot is mentionned
	WhenMentionned bool
}

// Command is a Command implemented by a plugin
type Command struct {
	Name             string
	ShortDescription string
	LongDescription  string
}
```


### Active triggers

Define a command like `help`. The bot will look for either:

- `!help`
- `@bot help`
- Direct message starting with `help`

### Passive triggers

Will parse every message to find a match using the POSIX regexp. If you want to mach all message just put `.*`

### WhenMentionned

Only call the plugin when mentionned or withing a DM conversation.


## Response type

The response a plugin will return must be a type:

```go
// SlackResponse struct
type SlackResponse struct {
	Channel string
	Text      string
	Params    *slack.PostMessageParameters
}
```

Be sure to set the `Channel field` (you can take it from `message.Channel`)
If you set a `userID` as a channel, it will find for your proper DM `Channel` before sending for you.

The `Text` field follow the basic message formatting rules defined [here](https://api.slack.com/docs/message-formatting).

The `Params` field is used to create rich format message using attachments as described [here](https://godoc.org/github.com/nlopes/slack#PostMessageParameters).

You can find details for advanced attachments formating [here](https://api.slack.com/docs/message-attachments).

*TIPS:* You can use [this website](http://davestevens.github.io/slack-message-builder/) to check the attachement syntax.
