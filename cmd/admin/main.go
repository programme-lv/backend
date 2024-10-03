// main.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nfnt/resize"
	"github.com/programme-lv/backend/fstask" // Assuming fstask is defined here
	"github.com/programme-lv/backend/fstask/lio2023"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wailsapp/mimetype"
	"golang.org/x/sync/errgroup"
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

func uploadTask(fsTask *fstask.Task, shortId string) error {
	log.Info().
		Str("shortId", shortId).
		Str("taskName", fsTask.FullName).
		Msg("Starting uploadTask")

	taskSrvc, err := tasksrvc.NewTaskSrvc()
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error creating task service")
		return fmt.Errorf("error creating task service: %w", err)
	}

	illstrImg := fsTask.GetIllustrationImage()
	illstrImgUrl := ""
	if illstrImg != nil {
		log.Debug().
			Str("relativePath", illstrImg.RelativePath).
			Msg("Compressing illustration image")

		compressedImage, err := downscaleImage(illstrImg.Content, 600)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to compress image")
			return fmt.Errorf("failed to compress image: %w", err)
		}

		mType := mime.TypeByExtension(filepath.Ext(illstrImg.RelativePath))
		if mType == "" {
			detectedType := mimetype.Detect(compressedImage)
			if detectedType == nil {
				log.Error().Msg("Failed to detect file type for image")
				return fmt.Errorf("failed to detect file type")
			}
			mType = detectedType.String()
			log.Debug().
				Str("mimeType", mType).
				Msg("Detected MIME type for image")
		}

		illstrImgUrl, err = taskSrvc.UploadIllustrationImg(mType, compressedImage)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to upload illustration image")
			return fmt.Errorf("failed to upload illustration image: %w", err)
		}
		log.Info().
			Str("url", illstrImgUrl).
			Msg("Uploaded illustration image")
	}

	// Process Origin Notes
	originNotes := make([]tasksrvc.OriginNote, 0)
	for k, v := range fsTask.OriginNotes {
		originNotes = append(originNotes, tasksrvc.OriginNote{
			Lang: k,
			Info: v,
		})
	}
	log.Debug().
		Int("count", len(originNotes)).
		Msg("Processed origin notes")

	// Process Markdown Statements
	mdStatements := make([]tasksrvc.MarkdownStatement, 0)
	for _, mdStatement := range fsTask.MarkdownStatements {
		mdStatements = append(mdStatements, tasksrvc.MarkdownStatement{
			LangIso639: mdStatement.Language,
			Story:      mdStatement.Story,
			Input:      mdStatement.Input,
			Output:     mdStatement.Output,
			Notes:      mdStatement.Notes,
			Scoring:    mdStatement.Scoring,
		})
	}
	log.Debug().
		Int("count", len(mdStatements)).
		Msg("Processed markdown statements")

	// Process PDF Statements
	pdfStatements := make([]tasksrvc.PdfStatement, 0)
	for _, pdfStatement := range fsTask.PdfStatements {
		pdfURL, err := taskSrvc.UploadStatementPdf(pdfStatement.Content)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to upload statement pdf")
			return fmt.Errorf("failed to upload statement pdf: %w", err)
		}
		log.Info().
			Str("url", pdfURL).
			Msg("Uploaded statement pdf")
		pdfStatements = append(pdfStatements, tasksrvc.PdfStatement{
			LangIso639: pdfStatement.Language,
			ObjectUrl:  pdfURL,
		})
	}
	log.Debug().
		Int("count", len(pdfStatements)).
		Msg("Processed PDF statements")

	// Process Examples
	examples := make([]tasksrvc.Example, 0)
	for _, e := range fsTask.Examples {
		examples = append(examples, tasksrvc.Example{
			Input:  string(e.Input),
			Output: string(e.Output),
			MdNote: string(e.MdNote),
		})
	}
	log.Debug().
		Int("count", len(examples)).
		Msg("Processed examples")

	// Process Tests Concurrently
	tests := make([]tasksrvc.Test, len(fsTask.Tests))

	// Mutex to protect access to the tests slice
	var mu sync.Mutex

	// Use errgroup for managing goroutines and error handling
	g, ctx := errgroup.WithContext(context.Background())

	// Semaphore to limit the number of concurrent uploads (e.g., 10)
	concurrencyLimit := 10
	sem := make(chan struct{}, concurrencyLimit)

	log.Debug().
		Int("concurrencyLimit", concurrencyLimit).
		Msg("Starting concurrent test uploads")

	// Iterate over tests and launch goroutines for uploading
	for i, t := range fsTask.Tests {
		i, t := i, t // Capture loop variables
		g.Go(func() error {
			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				// Acquired
			case <-ctx.Done():
				return ctx.Err()
			}
			defer func() { <-sem }() // Release semaphore

			log.Debug().
				Int("testID", i+1).
				Msg("Uploading test files")

			// Compute SHA2 hashes
			inpSha2 := taskSrvc.Sha2Hex(t.Input)
			ansSha2 := taskSrvc.Sha2Hex(t.Answer)

			// Upload test input
			if err := taskSrvc.UploadTestFile(t.Input); err != nil {
				log.Error().
					Int("testID", i+1).
					Err(err).
					Msg("Failed to upload test input")
				return fmt.Errorf("failed to upload test input for test ID %v: %w", i+1, err)
			}
			log.Debug().
				Int("testID", i+1).
				Msg("Uploaded test input")

			// Upload test answer
			if err := taskSrvc.UploadTestFile(t.Answer); err != nil {
				log.Error().
					Int("testID", i+1).
					Err(err).
					Msg("Failed to upload test answer")
				return fmt.Errorf("failed to upload test answer for test ID %v: %w", i+1, err)
			}
			log.Debug().
				Int("testID", i+1).
				Msg("Uploaded test answer")

			// Create the Test struct
			test := tasksrvc.Test{
				InpSha2: inpSha2,
				AnsSha2: ansSha2,
			}

			// Safely append to the tests slice
			mu.Lock()
			tests[i] = test
			mu.Unlock()

			log.Debug().
				Int("testID", i+1).
				Msg("Test struct created")

			return nil
		})
	}

	// Wait for all goroutines to finish
	if err := g.Wait(); err != nil {
		log.Error().
			Err(err).
			Msg("Error during concurrent test uploads")
		return err
	}
	log.Info().
		Int("count", len(tests)).
		Msg("Uploaded tests")

	// Process Test Groups
	testGroups := make([]tasksrvc.TestGroup, 0)
	for _, testGroup := range fsTask.TestGroups {
		testGroups = append(testGroups, tasksrvc.TestGroup{
			Points:  testGroup.Points,
			Public:  testGroup.Public,
			TestIDs: testGroup.TestIDs,
		})
	}
	log.Debug().
		Int("count", len(testGroups)).
		Msg("Processed test groups")

	visInpSubtasks := make([]tasksrvc.VisibleInputSubtask, 0)
	// Assemble the Task struct
	task := &tasksrvc.Task{
		ShortId:          shortId,
		FullName:         fsTask.FullName,
		IllustrImgUrl:    illstrImgUrl,
		MemLimMegabytes:  fsTask.MemoryLimitMegabytes,
		CpuTimeLimSecs:   fsTask.CPUTimeLimitSeconds,
		OriginOlympiad:   fsTask.OriginOlympiad,
		DifficultyRating: fsTask.DifficultyOneToFive,
		OriginNotes:      originNotes,
		MdStatements:     mdStatements,
		PdfStatements:    pdfStatements,
		VisInpSubtasks:   visInpSubtasks,
		Examples:         examples,
		Tests:            tests,
		Checker:          fsTask.TestlibChecker,
		Interactor:       fsTask.TestlibInteractor,
		Subtasks:         []tasksrvc.Subtask{},
		TestGroups:       testGroups,
	}
	log.Debug().
		Str("shortId", shortId).
		Msg("Task struct assembled")

	// Upload the Task
	err = taskSrvc.PutTask(task)
	if err != nil {
		log.Error().
			Str("shortId", shortId).
			Err(err).
			Msg("Failed to upload task")
		return fmt.Errorf("failed to upload task: %w", err)
	}

	return nil
}

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
