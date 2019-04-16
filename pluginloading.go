package main

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/CyrilPeponnet/slackhal/plugin"
)

func initPLugins(disabledPlugins []string, httpPort string, output chan<- *plugin.SlackResponse, bot *plugin.Bot) {

	// Loading our plugin and Init them
	handlers := false
	if len(disabledPlugins) != 0 {
		zap.L().Info("Plugins disabled", zap.String("plugins", strings.Join(disabledPlugins, ", ")))
	}
	zap.L().Info("Loading plugins")

Loading:
	for _, p := range plugin.PluginManager.Plugins {
		meta := p.GetMetadata()
		for _, disabled := range disabledPlugins {
			if meta.Name == disabled {
				meta.Disabled = true
				continue Loading
			}
		}
		zap.L().Info("Loading", zap.String("plugin", meta.Name), zap.String("version", meta.Version))
		p.Init(zap.L().Named(meta.Name), output, bot)
		// Register handlers if any
		for route, handler := range meta.HTTPHandler {
			handlers = true
			zap.L().Info("Registering HTTP handler", zap.String("plugin", route.Name), zap.String("address", httpPort))
			http.Handle(route.Name, handler)
		}
	}

	// Start the http handler if we have some handlers registered.
	if handlers {
		go func() {
			if err := http.ListenAndServe(httpPort, nil); err != nil {
				zap.L().Fatal("Failed to register HTTP handler", zap.String("address", httpPort), zap.Error(err))
			}
		}()
	}
}
