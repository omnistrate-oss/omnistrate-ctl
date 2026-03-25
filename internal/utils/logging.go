package utils

import (
	"os"
	"sync"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var once sync.Once

func ConfigureLoggingFromEnvOnce() {
	once.Do(func() {
		logLevel := config.GetLogLevel()
		logFormat := config.GetLogFormat()

		ConfigureLogging(logLevel, logFormat)
	})
}

func ConfigureLogging(logLevel string, logFormat string) {
	log.Logger = log.With().Timestamp().Logger().Level(zerolog.DebugLevel)

	if logLevel != "" {
		level, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			log.Panic().Err(err).Str("level", logLevel).Msg("Invalid log level")
		}
		log.Logger = log.Logger.Level(level)
	}

	switch logFormat {
	case "json":
		// Defaults to JSON already nothing to do
	case "", "pretty":
		log.Logger = log.Logger.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			NoColor:    false,
			TimeFormat: time.RFC3339,
		})
	default:
		log.Panic().Str("log_format", logFormat).Msg("Unknown log format")
	}
}
