package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/CyrilPeponnet/slackhal/plugin"
)

func initPLugins(disabledPlugins []string, output chan<- *plugin.SlackResponse) {
	// Loading our plugin and Init them
	handlers := false
	Log.WithField("prefix", "[main]").Infof("Plugins %v are disabled", strings.Join(disabledPlugins, ", "))
	Log.WithField("prefix", "[main]").Info("Loading plugins")

Loading:
	for _, p := range plugin.PluginManager.Plugins {
		meta := p.GetMetadata()
		for _, disabled := range disabledPlugins {
			if meta.Name == disabled {
				meta.Disabled = true
				continue Loading
			}
		}
		Log.WithField("prefix", "[main]").Infof(" - %v version %v", meta.Name, meta.Version)
		p.Init(Log.WithField("prefix", fmt.Sprintf("[plugin %v]", meta.Name)), output)
		// Register handlers if any
		for route, handler := range meta.HTTPHandler {
			handlers = true
			Log.WithField("prefix", "[main]").Infof("  -> Registering HTTP handler for %v", route.Name)
			http.Handle(route.Name, handler)
		}
	}

	// Start the http handler if we have some handlers registered.
	if handlers {
		Log.WithField("prefix", "[main]").Info("HTTP Handler Started")
		go func() { Log.Fatal(http.ListenAndServe(":8080", nil)) }()
	}
}
