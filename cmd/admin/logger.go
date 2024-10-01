package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitializeLogger sets up the Zerolog logger based on the provided configuration.
func InitializeLogger(logLevel string, logToFile bool, logFilePath string) error {
	// Parse the log level
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)

	// Configure the logger output
	if logToFile {
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		log.Logger = log.Output(file)
	} else {
		// Human-friendly console output with color
		log.Logger = log.With().Caller().Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.TimeOnly, NoColor: true})
	}

	return nil
}
