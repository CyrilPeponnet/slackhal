package main

import (
	"strings"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// Log factory
var Log = logrus.New()

func init() {
	Log.Formatter = new(prefixed.TextFormatter)
	Log.Level = logrus.ErrorLevel
}

func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		Log.Level = logrus.DebugLevel
	case "info":
		Log.Level = logrus.InfoLevel
	case "warn":
		Log.Level = logrus.WarnLevel
	case "error":
		Log.Level = logrus.ErrorLevel
	case "critical":
		Log.Level = logrus.FatalLevel
	default:
		Log.Level = logrus.ErrorLevel
	}
}
