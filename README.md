# SLACKHAL - Yet another slack bot in Golang

## Usage

```console
Usage: slackhal [options] [--plugin-path path...]

Options:
  -h, --help               Show this help.
  -t, --token token        The slack bot token to use.
  -f, --file confing       The configuration file to load [default ./slackhal.yml]
  --trigger char           The char used to detect direct commands [default: !].
  --http-handler-port port The Port of the http handler [default: :8080].
  -l, --log level          Set the log level [default: error].
```

## Configuration file

Example of `yaml` configuration file:

```yaml
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

## Plugins

You can take a look at the builtins plugins to understand how it works.

## Plugins implementation

Your plugin must implement the following interface:

```go
type Plugin interface {
  Init(Logger *zap.Logger, output chan<- *SlackResponse, bot *Bot)
  GetMetadata() *Metadata
  ProcessMessage(command string, message slack.Msg)
  Self() interface{}
}
```

### The `Init` function

Will be called upon plugin login. You can use it if you need to init some stuff.

You can use the `Logger` as a logger factory for your plugin.

You will use `output` chan to send your responses back.

### The `GetMetadata` function

Must return a `plugin.Metadata` struct. This is used to register you plugin.

### The `ProcessMessage` function

Will be called when an action or a trigger is found in an incoming message. You can check `slack.Msg` type [here](https://godoc.org/github.com/nlopes/slack#Msg).

To send your response you can send `*plugin.SlackResponse` instances to the `output` channel provided by `Init` above.

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
  loggerer.PassiveTriggers = []plugin.Command{plugin.Command{Name: `(?s:.*)`, ShortDescription: "Log everything", LongDescription: "Will intercept all messages to log them."}}
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

where `pluginjira` is the subfolder where your package is stored.

### Example

```go
package builtins

import (
  "strings"

  "go.uber.org/zap"

  "github.com/CyrilPeponnet/slackhal/plugin"
  "github.com/nlopes/slack"
)

// echo struct define your plugin
type echo struct {
  plugin.Metadata
  sink chan<- *plugin.SlackResponse
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *echo) Init(Logger *zap.Logger, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
  h.sink = output
}

// GetMetadata interface implementation
func (h *echo) GetMetadata() *plugin.Metadata {
  return &h.Metadata
}

// ProcessMessage interface implementation
func (h *echo) ProcessMessage(command string, message slack.Msg) {

  if len(strings.Split(message.Text, " ")) == 1 {
    return
  }

  o := new(plugin.SlackResponse)
  o.Options = append(o.Options, slack.MsgOptionText(message.Text[strings.Index(message.Text, command)+len(command)+1:len(message.Text)], false))
  o.Channel = message.Channel
  // This is a test to implement tracking of message
  o.TrackerID = 42
  h.sink <- o
}

// Self interface implementation
func (h *echo) Self() (i interface{}) {
  return h
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

## Plugin behavior

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
  WhenMentioned bool
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

Will parse every message to find a match using the POSIX regular expression. If you want to mach all message just put `(?s:.*)`

### HTTP Handlers

You can add a HTTP Handler by defining:

```go
s := newEventHandler()
h.HTTPHandler[plugin.Command{Name: "/jira", ShortDescription: "Jira issue event hook.", LongDescription: "Will trap new issue created and send a notification to channels."}] = s
```

You handler must implement the [http.Handler interface](https://golang.org/pkg/net/http/#Handler).

### `WhenMentioned`

Only call the plugin when mentioned or within a DM conversation.

## Response channel

The response channel `output` will take `*SlackResponse` struct like:

```go
// SlackResponse struct
type SlackResponse struct {
  Channel    string
  TrackerID  int
  TrackedTTL int
  Options    []slack.MsgOption
}
```

Be sure to set the `Channel field` (you can take it from `message.Channel`).

- If you set a `userID` as a channel, it will find for your proper DM `Channel` before sending for you.
- If you set a channel as a string with a leading `#`, it will try to resolve it to the good channel id.

The `TrackerID` is used if you want to edit sent message later. Your plugin must set the `trackerID` with a positive integer that will be used as an identifier to edit the message later. The `TrackedTTL` field is used to set a TTL of tracking. If you send two `SlackResponse` with the same `TrackerID`, it will edit the message instead of posting a new one.

The `Options` field is used to set your message options as described [here](https://godoc.org/github.com/nlopes/slack#MsgOption).

You can find details for advanced attachments formatting [here](https://api.slack.com/docs/message-attachments).

*TIPS:* You can use [this website](http://davestevens.github.io/slack-message-builder/) to check the attachment syntax.
