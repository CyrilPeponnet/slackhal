# SLACKHAL - Yet another slack bot in golang.

## Usage

```
Usage: slackhal [options] [--plugin-path path...]

Options:
	-h, --help               Show this help.
	-t, --token token        The slack bot token to use.
	-f, --file confing		 The configuration file to load [default ./slackhal.yml]
	-p, --plugins-path path  The paths to the plugins folder to load [default: ./plugins].
	--trigger char           The char used to detect direct commands [default: !].
	--http-handler-port port The Port of the http handler [default: :8080].
	-l, --log level          Set the log level [default: error].
```

## Configuration file

Example of yaml configuration file:

```
bot:
  token: "yourtoken"
  trigger: "!"
  httpHandlerPort: ":8080"
  log:
    level: debug
  plugins:
    disabled:
      - echo
      - logger
```

# Plugins

## Internal plugins

Your plugin must implement the following interfaces:

```go
Init(Logger *logrus.Entry, output chan<- *SlackResponse)
GetMetadata() *Metadata
ProcessMessage(commands []string, message slack.Msg)
Self() interface{}
```

### The `Init` function

Will be called upon plugin login. You can use it if you need to init some stuff.

You can use the `Logrus.Entry` as a logger factory for your plugin.

You will use `output` chan to send your responses back.

### The `GetMetadata` function

Must return a `plugin.Metadata` struct. This is used to register you plugin.

### The `ProcessMessage` function

Will be called when an action or a trigger is found in an incoming message. You can check `slack.Msg` type [here](https://godoc.org/github.com/nlopes/slack#Msg).

To send your reponse you can send `*plugin.SlackResponse` instances to the `output` channel provided by `Init` above.

### The `Self` function

This is a dirty hack to access plugin from another plugin.

You will need to expose your plugin structure like:

```go
type Jira struct {
```

(note the Uppercase)

Then implement `Self` as:

```go
// Self interface implementation
func (h *Jira) Self()  interface{} {
	return h
}
```

You can then call a plugin from another plugin using the PluginManager map and realize an assertion:

```go
// Check if the plugin is loaded and not disabled
if pg, ok := plugin.PluginManager.Plugins["jira"]; ok {
	// Retrieve metadata
	info := pg.GetMetadata()
	// Check if not disabled
	if ! info.Disabled {
		// Realize a safe assertion
		if jp, found := .(*jiraplugin.Jira); found {
			jp.Connect()
			...
		}
	}
}
```

(note that the function you want to call from another plugin needs to be exported as well).

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

This is where you will initialize the `plugin.Metadata` struct and add your commands / triggers. It must end with a call to `plugin.PluginManager.Register()` function call to load you plugin.

*NOTE:* You should not use this function to init your plugin, this is only meant for registration process. Use the `Init` function for that.

### Package consideration

If you are not creating your plugin under the `builtins` package you will need to update `slackhal.go` to import your module like:

```go
_ "github.com/CyrilPeponnet/slackhal/plugins/plugin-jira"
```

where `pluginjira` is the subfolder where your parckage is stored.

__TODO: Maybe we could use go-generate for that__

### Example:

```go
package builtins

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/CyrilPeponnet/slackhal/plugin"
)

// echo struct define your plugin
type echo struct {
	plugin.Metadata
	sink chan<- *plugin.SlackResponse
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *echo) Init(Logger *logrus.Entry, output chan<- *plugin.SlackResponse) {
	h.sink = output
}

// GetMetadata interface implementation
func (h *echo) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *echo) ProcessMessage(commands []string, message slack.Msg) {
	for _, c := range commands {
		if c == "echo" {
			o := new(plugin.SlackResponse)
			o.Text = strings.Replace(message.Text, c+" ", "", 1)
			o.Channel = message.Channel
			h.sink <- o
		}
	}
}

func (h *echo) Self() interface{}{
	//Nothing
}

// init function that will register your plugin to the plugin manager
func init() {
	echoer := new(echo)
	echoer.Metadata = plugin.NewMetadata("echo")
	echoer.Description = "Will repeat what you said"
	echoer.ActiveTriggers = []plugin.Command{plugin.Command{Name: "echo", ShortDescription: "Parrot style", LongDescription: "Will repeat what you put after."}}
	plugin.PluginManager.Register(echoer)
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
	// Webhook handler
	HTTPHandler map[Command]http.Handler
	// Only trigger this plugin if the bot is mentionned
	WhenMentionned bool
	// Disabled state
	Disabled bool
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

### HTTP Handlers

You can add a HTTP Handler by defining:

```
s := newEventHandler()
h.HTTPHandler[plugin.Command{Name: "/jira", ShortDescription: "Jira issue event hook.", LongDescription: "Will trap new issue created and send a notification to channels."}] = s
```

You handler must implement the [http.Handler interface](https://golang.org/pkg/net/http/#Handler).

### WhenMentionned

Only call the plugin when mentionned or within a DM conversation.

## Response channel

The response channel `output` will take `*SlackResponse` struct like:

```go
// SlackResponse struct
type SlackResponse struct {
	Channel string
	Text      string
	Params    *slack.PostMessageParameters
}
```

Be sure to set the `Channel field` (you can take it from `message.Channel`).

- If you set a `userID` as a channel, it will find for your proper DM `Channel` before sending for you.
- If you set a channel as a string with a leading `#`, it will try to resolve it to the good channel id.

The `Text` field follow the basic message formatting rules defined [here](https://api.slack.com/docs/message-formatting).

The `Params` field is used to create rich format message using attachments as described [here](https://godoc.org/github.com/nlopes/slack#PostMessageParameters).

You can find details for advanced attachments formatting [here](https://api.slack.com/docs/message-attachments).

*TIPS:* You can use [this website](http://davestevens.github.io/slack-message-builder/) to check the attachment syntax.
