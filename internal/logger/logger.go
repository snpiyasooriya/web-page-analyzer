package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// Init initializes the global logger with configuration
func Init() {
	Logger = logrus.New()

	// Set output to stdout
	Logger.SetOutput(os.Stdout)

	// Set log level based on environment variable, default to Info
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "DEBUG":
		Logger.SetLevel(logrus.DebugLevel)
	case "WARN":
		Logger.SetLevel(logrus.WarnLevel)
	case "ERROR":
		Logger.SetLevel(logrus.ErrorLevel)
	case "FATAL":
		Logger.SetLevel(logrus.FatalLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}

	// Set formatter based on environment variable, default to JSON in production
	env := os.Getenv("ENV")
	if env == "development" || env == "dev" {
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	} else {
		Logger.SetFormatter(&logrus.JSONFormatter{})
	}
}

// GetLogger returns the configured logger instance
func GetLogger() *logrus.Logger {
	if Logger == nil {
		Init()
	}
	return Logger
}

// WithField creates a new entry with a single field
func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

// Info logs an info message
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}
