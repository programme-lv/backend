package main

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid" // Import UUID package
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/rs/zerolog/log"
	"github.com/wailsapp/mimetype"
	"golang.org/x/sync/errgroup"
)

// Helper function to replace image URLs with UUIDs
func replaceImages(content string, uuidToAsset map[string]string) (string, error) {
	// Define regex to match markdown image syntax: ![alt text](image_url)
	imgRegex := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)

	// Replace function
	modifiedContent := imgRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the URL from the match
		submatches := imgRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			// If no URL is found, return the match as is
			return match
		}
		originalURL := submatches[1]

		// Generate a new UUID
		newUUID := uuid.New().String()

		// Replace the original URL with the UUID in the markdown
		newMatch := strings.Replace(match, originalURL, newUUID, 1)

		// Store the UUID and original URL in the map
		uuidToAsset[newUUID] = originalURL

		return newMatch
	})

	return modifiedContent, nil
}

func uploadTask(fsTask *fstask.Task, shortId string) error {
	log.Info().Str("shortId", shortId).Str("taskName", fsTask.FullName).Msg("Starting uploadTask")

	taskSrvc, err := tasksrvc.NewTaskSrvc()
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

	// Initialize the UUID to Asset mapping
	uuidToUrl := make(map[string]string)

	// Process Markdown Statements
	mdStatements := make([]tasksrvc.MarkdownStatement, 0)
	for _, mdStatement := range fsTask.MarkdownStatements {
		// Replace images in all relevant markdown fields
		modifiedStory, err := replaceImages(mdStatement.Story, uuidToUrl)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to replace images in Story")
			return fmt.Errorf("failed to replace images in Story: %w", err)
		}
		modifiedInput, err := replaceImages(mdStatement.Input, uuidToUrl)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to replace images in Input")
			return fmt.Errorf("failed to replace images in Input: %w", err)
		}
		modifiedOutput, err := replaceImages(mdStatement.Output, uuidToUrl)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to replace images in Output")
			return fmt.Errorf("failed to replace images in Output: %w", err)
		}
		modifiedNotes, err := replaceImages(mdStatement.Notes, uuidToUrl)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to replace images in Notes")
			return fmt.Errorf("failed to replace images in Notes: %w", err)
		}
		modifiedScoring, err := replaceImages(mdStatement.Scoring, uuidToUrl)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to replace images in Scoring")
			return fmt.Errorf("failed to replace images in Scoring: %w", err)
		}

		// Append the modified MarkdownStatement
		mdStatements = append(mdStatements, tasksrvc.MarkdownStatement{
			LangIso639: mdStatement.Language,
			Story:      modifiedStory,
			Input:      modifiedInput,
			Output:     modifiedOutput,
			Notes:      modifiedNotes,
			Scoring:    modifiedScoring,
		})
	}
	log.Debug().
		Int("count", len(mdStatements)).
		Msg("Processed markdown statements")

	// At this point, uuidToAsset contains all (UUID, original URL) pairs.
	// Now, upload each image and map UUID to S3 key.

	// Create a wait group and mutex for concurrent uploads
	var wg sync.WaitGroup
	var mu sync.Mutex
	var uploadErr error

	// Iterate through the uuidToAsset map
	for uuidKey, originalURL := range uuidToUrl {
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

			// Upload the image using UploadMarkdownImage
			s3Url, err := taskSrvc.UploadMarkdownImage(mType, assetFound.Content)
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

			mu.Lock()
			uuidToUrl[uKey] = s3Url
			mu.Unlock()
		}(uuidKey, originalURL)
	}

	// Wait for all uploads to finish
	wg.Wait()

	// Check if any upload errors occurred
	if uploadErr != nil {
		return uploadErr
	}

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

	subtasks := make([]tasksrvc.Subtask, 0)
	for _, subtask := range fsTask.Subtasks {
		subtasks = append(subtasks, tasksrvc.Subtask{
			Score:        subtask.Points,
			TestIDs:      subtask.TestIDs,
			Descriptions: subtask.Descriptions,
		})
	}

	visInpSubtasks := make([]tasksrvc.VisibleInputSubtask, 0)
	for _, visInputSubtask := range fsTask.VisibleInputSubtasks {
		if len(subtasks) < visInputSubtask {
			log.Error().
				Int("subtaskID", visInputSubtask).
				Msg("Invalid subtask ID")
			return fmt.Errorf("invalid subtask ID: %v", visInputSubtask)
		}
		subtask := subtasks[visInputSubtask-1]
		testsVis := make([]tasksrvc.VisInpSubtaskTest, 0)
		for _, testId := range subtask.TestIDs {
			if len(fsTask.Tests) < testId {
				log.Error().
					Int("testID", testId).
					Ints("tests", subtask.TestIDs).
					Msg("Invalid test ID")
				return fmt.Errorf("invalid test ID: %v", testId)
			}
			testsVis = append(testsVis, tasksrvc.VisInpSubtaskTest{
				TestId: testId,
				Input:  string(fsTask.Tests[testId-1].Input),
			})
		}
		visInpSubtasks = append(visInpSubtasks, tasksrvc.VisibleInputSubtask{
			SubtaskId: visInputSubtask,
			Tests:     testsVis,
		})
	}

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
		Subtasks:         subtasks,
		TestGroups:       testGroups,
		AssetUuidToUrl:   uuidToUrl,
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

	// Optionally, handle the UUID to S3 key mapping further here
	// For example, storing the mapping in a database or another service

	return nil
}
