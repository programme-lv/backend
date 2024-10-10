package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/programme-lv/backend/fstask/lio2023"
	"github.com/programme-lv/backend/fstask/lio2024"
	"github.com/rs/zerolog/log"
)

func transformLio2023Task(src string, dst string) error {
	logger := log.With().
		Str("src", src).
		Str("dst", dst).
		Logger()

	logger.Info().
		Msg("Starting transformLio2023Task")

	datetime := time.Now().Format("2006-01-02-15-04-05")
	path := filepath.Join(dst, filepath.Base(src)+"-"+datetime)
	logger = logger.With().
		Str("path", path).
		Logger()
	logger.Debug().
		Msg("Constructed destination path")

	task, err := lio2023.ParseLio2023TaskDir(src)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error parsing LIO2023 task")
		return fmt.Errorf("error parsing task: %w", err)
	}
	logger.Debug().
		Str("taskName", task.FullName).
		Msg("Parsed LIO2023 task")

	err = task.Store(path)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error storing transformed task")
		return fmt.Errorf("error storing task: %w", err)
	}
	logger.Info().
		Msg("Stored transformed task")

	return nil
}

func transformLio2024Task(src string, dst string) error {
	logger := log.With().
		Str("src", src).
		Str("dst", dst).
		Logger()

	logger.Info().
		Msg("Starting transformLio2024Task")

	datetime := time.Now().Format("2006-01-02-15-04-05")
	path := filepath.Join(dst, filepath.Base(src)+"-"+datetime)
	logger = logger.With().
		Str("path", path).
		Logger()
	logger.Debug().
		Msg("Constructed destination path")

	task, err := lio2024.ParseLio2024TaskDir(src)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error parsing LIO2024 task")
		return fmt.Errorf("error parsing task: %w", err)
	}
	logger.Debug().
		Str("taskName", task.FullName).
		Msg("Parsed LIO2024 task")

	err = task.Store(path)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error storing transformed task")
		return fmt.Errorf("error storing task: %w", err)
	}
	logger.Info().
		Msg("Stored transformed task")

	return nil
}
