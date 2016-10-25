package builtins

import (
	"fmt"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/slackhal/plugin"
)

// help struct define your plugin
type help struct {
	plugin.Metadata
}

// init function that will register your plugin to the plugin manager
func init() {
	helper := new(help)
	helper.Metadata = plugin.NewMetadata("help")
	helper.Metadata.Description = "Helper plugin."
	helper.ActiveTriggers = []plugin.Command{plugin.Command{Name: "help", ShortDescription: "Will provide some help :)"},
		plugin.Command{Name: "list-plugins", ShortDescription: "List all enabled plugins"},
		plugin.Command{Name: "list-commands", ShortDescription: "List all available commands"},
		plugin.Command{Name: "list-triggers", ShortDescription: "List all passive triggers"}}
	plugin.PluginManager.Register(helper)
}

// Init interface implementation if you need to init things
// When the bot is starting.
func (h *help) Init(Logger *logrus.Entry) {
	// Nothing to do
}

// GetMetadata interface implementation
func (h *help) GetMetadata() *plugin.Metadata {
	return &h.Metadata
}

// ProcessMessage interface implementation
func (h *help) ProcessMessage(commands []string, message slack.Msg, output chan<- *plugin.SlackResponse) {
	helpPluginPattern := regexp.MustCompile(`(help)\s*(\S*)\s*(\S*)`)
	o := new(plugin.SlackResponse)
	for _, c := range commands {
		switch {
		case helpPluginPattern.MatchString(message.Text):
			p := helpPluginPattern.FindStringSubmatch(message.Text)
			o.Text = GetHelpForPlugin(p)
		case c == "list-plugins":
			o.Text = PluginList()
		case c == "list-commands":
			o.Text = PluginListActions()
		case c == "list-triggers":
			o.Text = PluginListTriggers()
		}
	}
	o.Channel = message.User
	output <- o
}

// PluginListTriggers list plugins actions
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
