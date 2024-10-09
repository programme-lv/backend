// main.go
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
	"github.com/programme-lv/backend/fstask" // Assuming fstask is defined here
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wailsapp/mimetype"
)

var (
	logLevel    string
	logToFile   bool
	logFilePath string
)

func main() {
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
				err = uploadTask(task, shortId)
				if err != nil {
					log.Error().
						Str("taskName", task.FullName).
						Err(err).
						Msg("Error uploading task")
					break
				}

				log.Info().
					Str("taskName", task.FullName).
					Str("shortId", shortId).
					Msg("Task uploaded successfully")
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

// downscaleImage resizes and compresses the image to the specified maximum width.
// It returns the compressed image bytes or an error if the process fails.
func downscaleImage(imgContent []byte, maxWidth uint) ([]byte, error) {
	log.Debug().
		Int("originalSize", len(imgContent)).
		Uint("maxWidth", maxWidth).
		Msg("Starting image downscaling")

	mType := mimetype.Detect(imgContent)
	if mType == nil {
		log.Error().Msg("Unknown image type")
		return nil, fmt.Errorf("unknown image type")
	}
	log.Debug().
		Str("mimeType", mType.String()).
		Msg("Detected MIME type")

	var img image.Image
	var err error

	switch mType.String() {
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imgContent))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(imgContent))
	default:
		log.Error().
			Str("mimeType", mType.String()).
			Msg("Unsupported image format")
		return nil, fmt.Errorf("unsupported image format: %s", mType.String())
	}

	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to decode image")
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize the image while maintaining aspect ratio
	width := uint(img.Bounds().Dx())
	if width > maxWidth {
		width = maxWidth
	}
	resizedImg := resize.Resize(width, 0, img, resize.Lanczos3)
	log.Debug().
		Int("newWidth", resizedImg.Bounds().Dx()).
		Msg("Image resized")

	var compressedImg bytes.Buffer
	// Encode the resized image to JPEG format with quality 85
	err = jpeg.Encode(&compressedImg, resizedImg, &jpeg.Options{Quality: 85})
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to encode image to JPEG")
		return nil, fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	log.Debug().Msg("Image downscaled and compressed successfully")
	return compressedImg.Bytes(), nil
}
