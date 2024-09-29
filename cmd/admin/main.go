package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/nfnt/resize"
	"github.com/programme-lv/backend/fstask" // Assuming fstask is defined here
	"github.com/programme-lv/backend/fstask/lio2023"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/spf13/cobra"
	"github.com/wailsapp/mimetype"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "proglv",
		Short: "Admin CLI tool for programme.lv",
	}

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
			switch format {
			case "lio2023":
				err := transformLio2023Task(src, dst)
				if err != nil {
					log.Fatal(err)
				}
			default:
				log.Fatalf("Unsupported format: %s\n", format)
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
			for _, path := range args {
				// Resolve absolute path
				absPath, err := filepath.Abs(path)
				if err != nil {
					log.Printf("Error resolving path '%s': %v", path, err)
					continue
				}

				// Check if the path exists
				info, err := os.Stat(absPath)
				if err != nil {
					log.Printf("Path '%s' does not exist: %v", absPath, err)
					continue
				}
				if !info.IsDir() {
					log.Printf("Path '%s' is not a directory (skipping)", absPath)
					continue
				}

				// Parse as fstask (Assuming a ParseFSTask function exists)
				task, err := fstask.Read(absPath)
				if err != nil {
					log.Printf("Error parsing task from '%s': %v", absPath, err)
					continue
				}

				fmt.Printf("Successfully parsed task '%s' from '%s'\n", task.FullName, absPath)

				shortId := filepath.Base(absPath)
				err = uploadTask(task, shortId)
				if err != nil {
					log.Printf("Error uploading task '%s': %v", task.FullName, err)
					continue
				}
			}
		},
	}

	// Add 'upload' command as a subcommand of 'task'
	taskCmd.AddCommand(uploadCmd)

	// Build the command hierarchy
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskTransformCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func uploadTask(fsTask *fstask.Task, shortId string) error {
	taskSrvc, err := tasksrvc.NewTaskSrvc()
	if err != nil {
		return fmt.Errorf("error creating task service: %w", err)
	}

	illstrImg := fsTask.GetIllustrationImage()
	compressedImage, err := downscaleImage(illstrImg.Content, 600)
	if err != nil {
		return fmt.Errorf("failed to compress image: %w", err)
	}

	mType := mime.TypeByExtension(filepath.Ext(illstrImg.RelativePath))
	if mType == "" {
		detectedType := mimetype.Detect(compressedImage)
		if detectedType == nil {
			return fmt.Errorf("failed to detect file type")
		}
		mType = detectedType.String()
	}

	url, err := taskSrvc.UploadIllustrationImg(mType, compressedImage)
	if err != nil {
		return fmt.Errorf("failed to upload illustration image: %w", err)
	}

	task := &tasksrvc.Task{
		ShortId:          shortId,
		FullName:         fsTask.FullName,
		IllustrImgUrl:    url,
		MemLimMegabytes:  fsTask.MemoryLimInMegabytes,
		CpuTimeLimSecs:   fsTask.CpuTimeLimInSeconds,
		OriginOlympiad:   fsTask.OriginOlympiad,
		DifficultyRating: fsTask.DifficultyOneToFive,
		OriginNotes:      []tasksrvc.OriginNote{},
		MdStatements:     []tasksrvc.MarkdownStatement{},
		PdfStatements:    []tasksrvc.PdfStatement{},
		VisInpSubtasks:   []tasksrvc.VisInpSubtask{},
		Examples:         []tasksrvc.Example{},
		Tests:            []tasksrvc.Test{},
		Checker:          fsTask.TestlibChecker,
		Interactor:       fsTask.TestlibInteractor,
		Subtasks:         []tasksrvc.Subtask{},
		TestGroups:       []tasksrvc.TestGroup{},
	}

	return taskSrvc.PutTask(task)
}

func transformLio2023Task(src string, dst string) error {
	datetime := time.Now().Format("2006-01-02-15-04-05")
	path := filepath.Join(dst, filepath.Base(src)+"-"+datetime)

	task, err := lio2023.ParseLio2023TaskDir(src)
	if err != nil {
		return fmt.Errorf("error parsing task: %w", err)
	}

	err = task.Store(path)
	if err != nil {
		return fmt.Errorf("error storing task: %w", err)
	}

	return nil
}

// downscaleImage resizes and compresses the image to the specified maximum width.
// It returns the compressed image bytes or an error if the process fails.
func downscaleImage(imgContent []byte, maxWidth uint) ([]byte, error) {
	mType := mimetype.Detect(imgContent)
	if mType == nil {
		return nil, fmt.Errorf("unknown image type")
	}

	var img image.Image
	var err error

	switch mType.String() {
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imgContent))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(imgContent))
	default:
		return nil, fmt.Errorf("unsupported image format: %s", mType.String())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize the image while maintaining aspect ratio
	width := uint(img.Bounds().Dx())
	if width > maxWidth {
		width = maxWidth
	}
	resizedImg := resize.Resize(width, 0, img, resize.Lanczos3)

	var compressedImg bytes.Buffer
	// Encode the resized image to JPEG format with quality 85
	err = jpeg.Encode(&compressedImg, resizedImg, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	return compressedImg.Bytes(), nil
}
