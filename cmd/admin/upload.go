package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid" // Import UUID package
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nfnt/resize"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/task/pgrepo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/rs/zerolog/log"
	"github.com/wailsapp/mimetype"
	"golang.org/x/sync/errgroup"
)

func uploadTask(fsTask *fstask.Task, shortId string) error {
	log.Info().Str("shortId", shortId).Str("taskName", fsTask.FullName).Msg("Starting uploadTask")

	pg, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Error().Err(err).Msg("Error creating pg pool")
		return fmt.Errorf("error creating pg pool: %w", err)
	}
	defer pg.Close()

	repo := pgrepo.NewTaskPgRepo(pg)
	taskSrvc, err := srvc.NewTaskSrvc(repo)
	if err != nil {
		log.Error().Err(err).Msg("Error creating task service")
		return fmt.Errorf("error creating task service: %w", err)
	}

	// Handle Illustration Image
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

		illstrImgUrl, err = taskSrvc.UploadIllustrationImg(context.Background(), mType, compressedImage)
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
	originNotes := make([]srvc.OriginNote, 0)
	for k, v := range fsTask.OriginNotes {
		originNotes = append(originNotes, srvc.OriginNote{
			Lang: k,
			Info: v,
		})
	}
	log.Debug().
		Int("count", len(originNotes)).
		Msg("Processed origin notes")

	// Process Markdown Statements
	mdStatements := make([]srvc.MarkdownStatement, 0)
	for _, mdStatement := range fsTask.MarkdownStatements {
		processedMdStatement, err := processMdStatement(taskSrvc, fsTask, &mdStatement)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to process markdown statement")
			return fmt.Errorf("failed to process markdown statement: %w", err)
		}
		mdStatements = append(mdStatements, *processedMdStatement)
	}
	log.Debug().
		Int("count", len(mdStatements)).
		Msg("Processed markdown statements")

	// Process PDF Statements
	pdfStatements := make([]srvc.PdfStatement, 0)
	for _, pdfStatement := range fsTask.PdfStatements {
		pdfURL, err := taskSrvc.UploadStatementPdf(context.Background(), pdfStatement.Content)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to upload statement pdf")
			return fmt.Errorf("failed to upload statement pdf: %w", err)
		}
		log.Info().
			Str("url", pdfURL).
			Msg("Uploaded statement pdf")
		pdfStatements = append(pdfStatements, srvc.PdfStatement{
			LangIso639: pdfStatement.Language,
			ObjectUrl:  pdfURL,
		})
	}
	log.Debug().
		Int("count", len(pdfStatements)).
		Msg("Processed PDF statements")

	// Process Examples
	examples := make([]srvc.Example, 0)
	for _, e := range fsTask.Examples {
		examples = append(examples, srvc.Example{
			Input:  string(e.Input),
			Output: string(e.Output),
			MdNote: string(e.MdNote),
		})
	}
	log.Debug().
		Int("count", len(examples)).
		Msg("Processed examples")

	// Process Tests Concurrently
	tests := make([]srvc.Test, len(fsTask.Tests))

	// Mutex to protect access to the tests slice
	var testsMu sync.Mutex

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
			h := sha256.New()
			h.Write(t.Input)
			inpSha2 := hex.EncodeToString(h.Sum(nil))
			h.Reset()
			h.Write(t.Answer)
			ansSha2 := hex.EncodeToString(h.Sum(nil))

			// Upload test input
			if err := taskSrvc.UploadTestFile(ctx, t.Input); err != nil {
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
			if err := taskSrvc.UploadTestFile(ctx, t.Answer); err != nil {
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
			test := srvc.Test{
				InpSha2: inpSha2,
				AnsSha2: ansSha2,
			}

			// Safely append to the tests slice
			testsMu.Lock()
			tests[i] = test
			testsMu.Unlock()

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
	testGroups := make([]srvc.TestGroup, 0)
	for _, testGroup := range fsTask.TestGroups {
		testGroups = append(testGroups, srvc.TestGroup{
			Points:  testGroup.Points,
			Public:  testGroup.Public,
			TestIDs: testGroup.TestIDs,
		})
	}
	log.Debug().
		Int("count", len(testGroups)).
		Msg("Processed test groups")

	subtasks := make([]srvc.Subtask, 0)
	for _, subtask := range fsTask.Subtasks {
		subtasks = append(subtasks, srvc.Subtask{
			Score:        subtask.Points,
			TestIDs:      subtask.TestIDs,
			Descriptions: subtask.Descriptions,
		})
	}

	visInpSubtasks := make([]srvc.VisibleInputSubtask, 0)
	for _, visInputSubtask := range fsTask.VisibleInputSubtasks {
		if len(subtasks) < visInputSubtask {
			log.Error().
				Int("subtaskID", visInputSubtask).
				Msg("Invalid subtask ID")
			return fmt.Errorf("invalid subtask ID: %v", visInputSubtask)
		}
		subtask := subtasks[visInputSubtask-1]
		testsVis := make([]srvc.VisInpSubtaskTest, 0)
		for _, testId := range subtask.TestIDs {
			if len(fsTask.Tests) < testId {
				log.Error().
					Int("testID", testId).
					Ints("tests", subtask.TestIDs).
					Msg("Invalid test ID")
				return fmt.Errorf("invalid test ID: %v", testId)
			}
			testsVis = append(testsVis, srvc.VisInpSubtaskTest{
				TestId: testId,
				Input:  string(fsTask.Tests[testId-1].Input),
			})
		}
		visInpSubtasks = append(visInpSubtasks, srvc.VisibleInputSubtask{
			SubtaskId: visInputSubtask,
			Tests:     testsVis,
		})
	}

	// Assemble the Task struct
	task := srvc.Task{
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
		Subtasks:         subtasks,
		TestGroups:       testGroups,
	}
	log.Debug().
		Str("shortId", shortId).
		Msg("Task struct assembled")

	err = taskSrvc.CreateTask(context.Background(), task)
	if err != nil {
		log.Error().
			Str("shortId", shortId).
			Err(err).
			Msg("Failed to upload task")
		return fmt.Errorf("failed to upload task: %w", err)
	}

	return nil
}

// Helper function to replace image URLs with UUIDs in markdown content
func replaceImages(content string, uuidToAsset map[string]string) (string, error) {
	imgRegex := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	modifiedContent := imgRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := imgRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		originalURL := submatches[1]
		newUUID := uuid.New().String()
		newMatch := strings.Replace(match, originalURL, newUUID, 1)
		uuidToAsset[newUUID] = originalURL

		return newMatch
	})

	return modifiedContent, nil
}

// getImageDimensions decodes the image and returns its width and height in pixels.
func getImageDimensions(imgData []byte) (int, int, error) {
	reader := bytes.NewReader(imgData)
	img, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}
	return img.Width, img.Height, nil
}

// resizeImage resizes the image to ensure the width does not exceed maxWidth.
// It maintains the aspect ratio and returns the resized image data along with new dimensions.
func resizeImage(imgData []byte, maxWidth uint) ([]byte, int, int, error) {
	// Decode the image
	reader := bytes.NewReader(imgData)
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get original dimensions
	originalBounds := img.Bounds()
	originalWidth := originalBounds.Dx()
	originalHeight := originalBounds.Dy()

	// If the width is already less than or equal to maxWidth, no resizing needed
	if originalWidth <= int(maxWidth) {
		return imgData, originalWidth, originalHeight, nil
	}

	// Calculate new dimensions while maintaining aspect ratio
	newImg := resize.Resize(maxWidth, 0, img, resize.Lanczos3)
	newBounds := newImg.Bounds()
	newWidth := newBounds.Dx()
	newHeight := newBounds.Dy()

	// Encode the resized image back to its original format
	var buf bytes.Buffer
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 80}) // Adjust quality as needed
	case "png":
		err = png.Encode(&buf, newImg)
	default:
		return nil, 0, 0, fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf.Bytes(), newWidth, newHeight, nil
}

func processMdStatement(taskSrvc srvc.TaskSrvcClient, fsTask *fstask.Task, mdStatement *fstask.MarkdownStatement) (*srvc.MarkdownStatement, error) {
	sttmntImgUuidToUrl := make(map[string]string)
	// Replace images in all relevant markdown fields
	modifiedStory, err := replaceImages(mdStatement.Story, sttmntImgUuidToUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to replace images in Story")
		return nil, fmt.Errorf("failed to replace images in Story: %w", err)
	}
	modifiedInput, err := replaceImages(mdStatement.Input, sttmntImgUuidToUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to replace images in Input")
		return nil, fmt.Errorf("failed to replace images in Input: %w", err)
	}
	modifiedOutput, err := replaceImages(mdStatement.Output, sttmntImgUuidToUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to replace images in Output")
		return nil, fmt.Errorf("failed to replace images in Output: %w", err)
	}
	modifiedNotes, err := replaceImages(mdStatement.Notes, sttmntImgUuidToUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to replace images in Notes")
		return nil, fmt.Errorf("failed to replace images in Notes: %w", err)
	}
	modifiedScoring, err := replaceImages(mdStatement.Scoring, sttmntImgUuidToUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to replace images in Scoring")
		return nil, fmt.Errorf("failed to replace images in Scoring: %w", err)
	}
	modifiedTalk, err := replaceImages(mdStatement.Talk, sttmntImgUuidToUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to replace images in Talk")
		return nil, fmt.Errorf("failed to replace images in Talk: %w", err)
	}
	modifiedExample, err := replaceImages(mdStatement.Example, sttmntImgUuidToUrl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to replace images in Example")
		return nil, fmt.Errorf("failed to replace images in Example: %w", err)
	}

	// Create a wait group and mutex for concurrent uploads
	var wg sync.WaitGroup
	var mu sync.Mutex
	var uploadErr error

	images := make([]srvc.MdImgInfo, 0)

	// Iterate through the uuidToAsset map
	for uuidKey, originalURL := range sttmntImgUuidToUrl {
		wg.Add(1)
		go func(uKey, oURL string) {
			defer wg.Done()

			// Find the asset in fsTask.Assets with RelativePath == originalURL
			var assetFound *fstask.AssetFile
			for _, asset := range fsTask.Assets {
				if asset.RelativePath == oURL {
					assetFound = &asset
					break
				}
			}

			if assetFound == nil {
				log.Error().
					Str("uuid", uKey).
					Str("originalURL", oURL).
					Msg("Asset not found for the original URL")
				mu.Lock()
				uploadErr = fmt.Errorf("asset not found for URL: %s", oURL)
				mu.Unlock()
				return
			}

			// Determine MIME type
			mType := mime.TypeByExtension(filepath.Ext(assetFound.RelativePath))
			if mType == "" {
				detectedType := mimetype.Detect(assetFound.Content)
				if detectedType == nil {
					log.Error().
						Str("uuid", uKey).
						Str("relativePath", assetFound.RelativePath).
						Msg("Failed to detect MIME type for asset")
					mu.Lock()
					uploadErr = fmt.Errorf("failed to detect MIME type for asset: %s", assetFound.RelativePath)
					mu.Unlock()
					return
				}
				mType = detectedType.String()
				log.Debug().
					Str("uuid", uKey).
					Str("mimeType", mType).
					Msg("Detected MIME type for asset")
			}

			// Detect image width and height
			_, _, err := getImageDimensions(assetFound.Content)
			if err != nil {
				log.Error().
					Str("uuid", uKey).
					Err(err).
					Msg("Failed to get image dimensions")
				mu.Lock()
				uploadErr = fmt.Errorf("failed to get image dimensions for asset: %s", assetFound.RelativePath)
				mu.Unlock()
				return
			}

			// Compress the image if necessary
			const maxWidth = 800
			resizedContent, newWidth, newHeight, err := resizeImage(assetFound.Content, maxWidth)
			if err != nil {
				log.Error().
					Str("uuid", uKey).
					Err(err).
					Msg("Failed to resize image")
				mu.Lock()
				uploadErr = fmt.Errorf("failed to resize image for asset: %s", assetFound.RelativePath)
				mu.Unlock()
				return
			}

			// Upload the resized image using UploadMarkdownImage
			s3Url, err := taskSrvc.UploadMarkdownImage(context.Background(), mType, resizedContent)
			if err != nil {
				log.Error().
					Str("uuid", uKey).
					Str("s3Key", s3Url).
					Err(err).
					Msg("Failed to upload markdown image")
				mu.Lock()
				uploadErr = fmt.Errorf("failed to upload markdown image for UUID %s: %w", uKey, err)
				mu.Unlock()
				return
			}
			log.Info().
				Str("uuid", uKey).
				Str("s3Key", s3Url).
				Msg("Uploaded markdown image")

			// Optionally, update widthEm based on some logic or markdown parsing
			widthEm := 0
			for _, imgInfo := range mdStatement.ImgSizes {
				if imgInfo.ImgPath == oURL {
					widthEm = imgInfo.WidthEm
					break
				}
			}

			mu.Lock()
			images = append(images, srvc.MdImgInfo{
				Uuid:     uKey,
				WidthPx:  newWidth,
				HeightPx: newHeight,
				WidthEm:  widthEm,
				S3Url:    s3Url,
			})
			mu.Unlock()
		}(uuidKey, originalURL)
	}

	// Wait for all uploads to finish
	wg.Wait()

	// Check if any upload errors occurred
	if uploadErr != nil {
		return nil, uploadErr
	}
	// Append the modified MarkdownStatement
	return &srvc.MarkdownStatement{
		LangIso639: mdStatement.Language,
		Story:      modifiedStory,
		Input:      modifiedInput,
		Output:     modifiedOutput,
		Notes:      modifiedNotes,
		Scoring:    modifiedScoring,
		Talk:       modifiedTalk,
		Example:    modifiedExample,
		Images:     images,
	}, nil
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
