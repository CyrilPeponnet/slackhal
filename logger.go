package main

import (
	"strings"

	"github.com/Sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var log = logrus.New()

func init() {
	log.Formatter = new(prefixed.TextFormatter)
	log.Level = logrus.ErrorLevel
}

func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		log.Level = logrus.DebugLevel
	case "info":
		log.Level = logrus.InfoLevel
	case "warn":
		log.Level = logrus.WarnLevel
	case "error":
		log.Level = logrus.ErrorLevel
	case "critical":
		log.Level = logrus.FatalLevel
	default:
		log.Level = logrus.ErrorLevel
	}
}
