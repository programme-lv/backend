package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/programme-lv/backend/fstask/lio2023"
	"github.com/rs/zerolog/log"
)

func transformLio2023Task(src string, dst string) error {
	log.Info().
		Str("src", src).
		Str("dst", dst).
		Msg("Starting transformLio2023Task")

	datetime := time.Now().Format("2006-01-02-15-04-05")
	path := filepath.Join(dst, filepath.Base(src)+"-"+datetime)
	log.Debug().
		Str("path", path).
		Msg("Constructed destination path")

	task, err := lio2023.ParseLio2023TaskDir(src)
	if err != nil {
		log.Error().
			Str("src", src).
			Err(err).
			Msg("Error parsing LIO2023 task")
		return fmt.Errorf("error parsing task: %w", err)
	}
	log.Debug().
		Str("taskName", task.FullName).
		Msg("Parsed LIO2023 task")

	err = task.Store(path)
	if err != nil {
		log.Error().
			Str("path", path).
			Err(err).
			Msg("Error storing transformed task")
		return fmt.Errorf("error storing task: %w", err)
	}
	log.Info().
		Str("path", path).
		Msg("Stored transformed task")

	return nil
}
