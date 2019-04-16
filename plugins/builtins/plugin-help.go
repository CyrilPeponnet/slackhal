package builtins

import (
	"fmt"
	"regexp"

	"go.uber.org/zap"

	"github.com/CyrilPeponnet/slackhal/plugin"
	"github.com/nlopes/slack"
)

// help struct define your plugin
type help struct {
	plugin.Metadata
	sink chan<- *plugin.SlackResponse
}

// init function that will register your plugin to the plugin manager
func init() {
	helper := new(help)
	helper.Metadata = plugin.NewMetadata("help")
	helper.Metadata.Description = "Helper plugin."
	helper.ActiveTriggers = []plugin.Command{plugin.Command{Name: "help", ShortDescription: "Will provide some help :)"},
		plugin.Command{Name: "list-plugins", ShortDescription: "List all enabled plugins."},
		plugin.Command{Name: "list-commands", ShortDescription: "List all available commands."},
		plugin.Command{Name: "list-handlers", ShortDescription: "List all available HTTP handlers."},
		plugin.Command{Name: "list-triggers", ShortDescription: "List all passive triggers."}}
	plugin.PluginManager.Register(helper)
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *help) Init(Logger *zap.Logger, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {
	h.sink = output
}

// GetMetadata interface implementation
func (h *help) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// Self interface implementation
func (h *help) Self() (i interface{}) {
	return h
}

// ProcessMessage interface implementation
func (h *help) ProcessMessage(command string, message slack.Msg) {
	helpPluginPattern := regexp.MustCompile(`(help)\s*(\S*)\s*(\S*)`)
	o := new(plugin.SlackResponse)
	switch {
	case helpPluginPattern.MatchString(message.Text):
		p := helpPluginPattern.FindStringSubmatch(message.Text)
		o.Options = append(o.Options, slack.MsgOptionText(GetHelpForPlugin(p), false))
	case command == "list-plugins":
		o.Options = append(o.Options, slack.MsgOptionText(PluginList(), false))
	case command == "list-commands":
		o.Options = append(o.Options, slack.MsgOptionText(PluginListActions(), false))
	case command == "list-handlers":
		o.Options = append(o.Options, slack.MsgOptionText(PluginListHandlers(), false))
	case command == "list-triggers":
		o.Options = append(o.Options, slack.MsgOptionText(PluginListTriggers(), false))
	}
	o.Channel = message.User
	h.sink <- o
}

// PluginListTriggers list plugins triggers
func PluginListTriggers() (o string) {
	l := ""
	for _, p := range plugin.PluginManager.Plugins {
		info := p.GetMetadata()
		if info.Disabled {
			continue
		}
		a := ""
		for _, c := range info.PassiveTriggers {
			a += fmt.Sprintf(">_%v_  - %v\n", c.Name, c.ShortDescription)
		}
		if a != "" {
			l += fmt.Sprintf("\n*%v* (%v) - %v\n", info.Name, info.Version, info.Description)
			l += a
		}
	}
	if l != "" {
		o = "Here are all the passive triggers enabled:\n"
		o += l
	} else {
		o = "Cannot find any passive triggers."
	}
	return
}

// PluginListHandlers list plugins handlers
func PluginListHandlers() (o string) {
	l := ""
	for _, p := range plugin.PluginManager.Plugins {
		info := p.GetMetadata()
		if info.Disabled {
			continue
		}
		a := ""
		for c := range info.HTTPHandler {
			a += fmt.Sprintf(">_%v_  - %v\n", c.Name, c.ShortDescription)
		}
		if a != "" {
			l += fmt.Sprintf("\n*%v* (%v) - %v\n", info.Name, info.Version, info.Description)
			l += a
		}
	}
	if l != "" {
		o = "Here are all the HTTP Handlers enabled:\n"
		o += l
	} else {
		o = "Cannot find any HTTP handlers."
	}
	return
}

// PluginListActions list plugins actions
func PluginListActions() (o string) {
	l := ""
	for _, p := range plugin.PluginManager.Plugins {
		info := p.GetMetadata()
		if info.Disabled {
			continue
		}
		a := ""
		for _, c := range info.ActiveTriggers {
			a += fmt.Sprintf(">_%v_  - %v\n", c.Name, c.ShortDescription)
		}
		if a != "" {
			l += fmt.Sprintf("\n*%v* (%v) - %v\n", info.Name, info.Version, info.Description)
			l += a
		}
	}
	if l != "" {
		o = "Here are all the commands availables:\n"
		o += l
	} else {
		o = "Cannot find any plugin actions."
	}
	return
}

// PluginList list plugins
func PluginList() (o string) {
	o = "Here is my plugin list:\n"
	for _, p := range plugin.PluginManager.Plugins {
		info := p.GetMetadata()
		if info.Disabled {
			continue
		}
		o += fmt.Sprintf(">*%v* (%v) - %v\n", info.Name, info.Version, info.Description)
	}
	return
}

// GetHelpForPlugin get help for a give plugin and commands
func GetHelpForPlugin(matches []string) (o string) {
	if matches[3] != "" || matches[2] != "" {
	loop:
		for _, p := range plugin.PluginManager.Plugins {
			info := p.GetMetadata()
			if info.Disabled {
				continue
			}
			if info.Name == matches[2] {
				o = fmt.Sprintf("*%v* (%v) - %v\n", info.Name, info.Version, info.Description)
				for _, c := range info.ActiveTriggers {
					if c.Name == matches[3] {
						o += fmt.Sprintf("> *%v*:\n```%v```\n", c.Name, c.LongDescription)
						break loop
					} else {

						o += fmt.Sprintf("> *%v* - %v\n", c.Name, c.ShortDescription)
					}
				}
			}
		}

	} else {
		o = GetHelpForPlugin([]string{"", "help", "help", ""})
	}

	if o == "" {
		o = fmt.Sprintf("Sorry but I cannot find help for `%v`", matches[0])
	}
	return o
}
