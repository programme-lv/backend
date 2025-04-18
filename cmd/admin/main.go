// main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/fstask" // Assuming fstask is defined here
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	logLevel    string
	logToFile   bool
	logFilePath string
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	var rootCmd = &cobra.Command{
		Use:   "proglv",
		Short: "Admin CLI tool for programme.lv",
	}

	// Logging flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error, fatal)")
	rootCmd.PersistentFlags().BoolVar(&logToFile, "log-to-file", false, "Enable logging to a file")
	rootCmd.PersistentFlags().StringVar(&logFilePath, "log-file", "cli.log", "Path to the log file")

	// Initialize logger before executing commands
	cobra.OnInitialize(initLogger)

	var taskCmd = &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}

	// Transform Command
	var src string
	var dst string
	var format string

	var taskTransformCmd = &cobra.Command{
		Use:   "transform",
		Short: "Transform task format to proglv format",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().
				Str("format", format).
				Str("src", src).
				Str("dst", dst).
				Msg("Transform command started")

			switch format {
			case "lio2023":
				err := transformLio2023Task(src, dst)
				if err != nil {
					log.Fatal().Err(err).Msg("Transform task failed")
				}
				log.Info().Msg("Transform task completed successfully")
			case "lio2024":
				err := transformLio2024Task(src, dst)
				if err != nil {
					log.Fatal().Err(err).Msg("Transform task failed")
				}
				log.Info().Msg("Transform task completed successfully")
			default:
				log.Fatal().Str("format", format).Msg("Unsupported format")
			}
		},
	}

	// Define flags for the 'transform' command
	taskTransformCmd.Flags().StringVarP(&src, "src", "s", "", "Source directory path (required)")
	taskTransformCmd.Flags().StringVarP(&dst, "dst", "d", "", "Destination directory path (required)")
	taskTransformCmd.Flags().StringVarP(&format, "format", "f", "", "Format of the import [lio2023, lio2024] (required)")

	// Mark 'src', 'dst', and 'format' as required flags
	taskTransformCmd.MarkFlagRequired("src")
	taskTransformCmd.MarkFlagRequired("dst")
	taskTransformCmd.MarkFlagRequired("format")

	// Upload Command
	var uploadCmd = &cobra.Command{
		Use:   "upload [paths...]",
		Short: "Upload tasks from specified paths",
		Long: `Upload tasks by providing one or more file or directory paths.
Each path will be parsed as an fstask and uploaded accordingly.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().
				Int("pathsCount", len(args)).
				Msg("Upload command started")

			for _, path := range args {
				log.Debug().
					Str("path", path).
					Msg("Processing path")

				// Resolve absolute path
				absPath, err := filepath.Abs(path)
				if err != nil {
					log.Error().
						Str("path", path).
						Err(err).
						Msg("Error resolving path")
					break
				}

				// Check if the path exists
				info, err := os.Stat(absPath)
				if err != nil {
					log.Error().
						Str("path", absPath).
						Err(err).
						Msg("Path does not exist")
					break
				}
				if !info.IsDir() {
					log.Warn().
						Str("path", absPath).
						Msg("Path is not a directory, skipping")
					break
				}

				// Parse as fstask (Assuming a Read function exists)
				task, err := fstask.Read(absPath)
				if err != nil {
					log.Error().
						Str("path", absPath).
						Err(err).
						Msg("Error parsing task")
					break
				}

				log.Info().
					Str("taskName", task.FullName).
					Str("path", absPath).
					Msg("Task parsed successfully")

				shortId := filepath.Base(absPath)
				log.Info().
					Str("taskName", task.FullName).
					Str("shortId", shortId).
					Msg("Task uploaded successfully")
				panic("not maintained")
				// err = uploadTask(task, shortId)
				// if err != nil {
				// 	log.Error().
				// 		Str("taskName", task.FullName).
				// 		Err(err).
				// 		Msg("Error uploading task")
				// 	break
				// }

			}

			log.Info().Msg("Upload command completed")
		},
	}

	// Add 'upload' command as a subcommand of 'task'
	taskCmd.AddCommand(uploadCmd)

	// Add 'transform' command as a subcommand of 'task'
	taskCmd.AddCommand(taskTransformCmd)

	// Build the command hierarchy
	rootCmd.AddCommand(taskCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		if log.Logger.GetLevel() != zerolog.Disabled {
			log.Fatal().Err(err).Msg("CLI execution failed")
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func initLogger() {
	err := InitializeLogger(logLevel, logToFile, logFilePath)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
}
