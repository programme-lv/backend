package srvc_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/programme-lv/backend/gen/mocks/mocktasksrvc"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/taskzip/taskfs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test archive stays the same after being imported and exported
func TestImportExportTask(t *testing.T) {
	// TODO: this is still a work in progress
	// the test as of now does not pass because
	// we cant save all archive fields to db tables
	ctx := context.Background()
	mockRepo := mocktasksrvc.NewMockTaskPgRepo(t)
	mockCdnS3 := mocktasksrvc.NewMockS3BucketFacade(t)
	mockTestS3 := mocktasksrvc.NewMockS3BucketFacade(t)
	taskSrvc, err := srvc.NewTaskSrvc(mockRepo, mockCdnS3, mockTestS3)
	require.NoError(t, err)

	// 1. read the task using task archive format reader
	zipPath := "testdata/kvadrputekl.zip"
	uncropped, err := taskfs.ReadZip(zipPath)
	require.NoError(t, err)

	// 2 create a new .zip with the new expected values
	expected := replaceLongContentWithChecksum(uncropped)
	newZipPath := filepath.Join(t.TempDir(), "kvadrputekl.zip")
	err = taskfs.WriteZip(expected, newZipPath)
	require.NoError(t, err)
	newZipBytes, err := os.ReadFile(newZipPath)
	require.NoError(t, err)

	// 3. import and fetch task from database
	mockCdnS3.EXPECT().Exists(mock.Anything).Return(false, nil)
	for _, img := range expected.Statement.Images {
		mime, _ := srvc.MimeFromFname(img.Fname)
		mockCdnS3.EXPECT().Upload(img.Content, mock.Anything, mime).RunAndReturn(func(content []byte, key string, mimeType string) (string, error) {
			mockCdnS3.EXPECT().Download(key).Return(content, nil)
			return "", nil
		})
	}
	for _, img := range expected.Archive.GetIllustrImgs() {
		mime, _ := srvc.MimeFromFname(img.Fname)
		mockCdnS3.EXPECT().Upload(img.Content, mock.Anything, mime).RunAndReturn(func(content []byte, key string, mimeType string) (string, error) {
			mockCdnS3.EXPECT().Download(key).Return(content, nil)
			return "", nil
		})
	}
	for _, pdf := range expected.Archive.GetOgStatementPdfs() {
		mockCdnS3.EXPECT().Upload(pdf.Content, mock.Anything, "application/pdf").Return("", nil)
		mockCdnS3.EXPECT().Download(mock.Anything).Return(pdf.Content, nil)
	}
	for _, test := range expected.Testing.Tests {
		inBytes := []byte(test.Input)
		ansBytes := []byte(test.Answer)
		mockTestS3.EXPECT().Exists(srvc.GetTestfileS3Key(inBytes)).Return(false, nil)
		mockTestS3.EXPECT().Exists(srvc.GetTestfileS3Key(ansBytes)).Return(false, nil)
		mockTestS3.EXPECT().Upload(srvc.MustCompressWithZstd(inBytes), mock.Anything, "application/zstd").RunAndReturn(func(content []byte, key string, mimeType string) (string, error) {
			mockTestS3.EXPECT().Download(key).Return(content, nil)
			return "", nil
		})
		mockTestS3.EXPECT().Upload(srvc.MustCompressWithZstd(ansBytes), mock.Anything, "application/zstd").RunAndReturn(func(content []byte, key string, mimeType string) (string, error) {
			mockTestS3.EXPECT().Download(key).Return(content, nil)
			return "", nil
		})
	}
	mockRepo.EXPECT().Exists(ctx, "kvadrputekl").Return(false, nil).Once()
	var createdTask srvc.Task
	mockRepo.EXPECT().CreateTask(ctx, mock.Anything).RunAndReturn(func(ctx context.Context, task srvc.Task) error {
		createdTask = task
		return nil
	})
	importedTaskId, err := taskSrvc.ImportTaskFromZip(ctx, newZipBytes)
	require.NoError(t, err, "Failed to import task from ZIP")

	// 4. map the imported task to the archive format
	mockRepo.EXPECT().Exists(ctx, "kvadrputekl").Return(true, nil).Once()
	mockRepo.EXPECT().GetTask(ctx, "kvadrputekl").Return(createdTask, nil).Once()
	newZip, err := taskSrvc.ExportTaskAsZip(ctx, importedTaskId)
	require.NoError(t, err)

	// 5. reinterpret the exported task as a taskfs.Task
	exportPath := filepath.Join(t.TempDir(), "exported.zip")
	err = os.WriteFile(exportPath, newZip, 0644)
	require.NoError(t, err)
	exported, err := taskfs.ReadZip(exportPath)
	require.NoError(t, err)
	exported = replaceLongContentWithChecksum(exported)

	// 5. compare the retrieved task with the expected
	require.Equal(t, uncropped, exported)
}

// when using require.Equal, too much gets printed to the console
// therefore we replace the test content with a checksum
func replaceLongContentWithChecksum(task taskfs.Task) taskfs.Task {
	result := task

	// Replace ReadMe with checksum if it's long
	if len(result.ReadMe) > 100 {
		result.ReadMe = checksumString(result.ReadMe)
	}

	// Replace statement stories with checksums
	for lang, story := range result.Statement.Stories {
		newStory := story
		if len(newStory.Story) > 100 {
			newStory.Story = checksumString(newStory.Story)
		}
		if len(newStory.Input) > 100 {
			newStory.Input = checksumString(newStory.Input)
		}
		if len(newStory.Output) > 100 {
			newStory.Output = checksumString(newStory.Output)
		}
		if len(newStory.Notes) > 100 {
			newStory.Notes = checksumString(newStory.Notes)
		}
		if len(newStory.Scoring) > 100 {
			newStory.Scoring = checksumString(newStory.Scoring)
		}
		if len(newStory.Talk) > 100 {
			newStory.Talk = checksumString(newStory.Talk)
		}
		if len(newStory.Example) > 100 {
			newStory.Example = checksumString(newStory.Example)
		}
		result.Statement.Stories[lang] = newStory
	}

	// Replace test inputs and answers with checksums
	for i, test := range result.Testing.Tests {
		if len(test.Input) > 100 {
			result.Testing.Tests[i].Input = checksumString(test.Input)
		}
		if len(test.Answer) > 100 {
			result.Testing.Tests[i].Answer = checksumString(test.Answer)
		}
	}

	// Replace checker and interactor with checksums if they're long
	if len(result.Testing.Checker) > 100 {
		result.Testing.Checker = checksumString(result.Testing.Checker)
	}
	if len(result.Testing.Interactor) > 100 {
		result.Testing.Interactor = checksumString(result.Testing.Interactor)
	}

	// Replace solution content with checksums
	for i, solution := range result.Solutions {
		if len(solution.Content) > 100 {
			result.Solutions[i].Content = checksumString(solution.Content)
		}
	}

	// Replace archive file contents with checksums
	for i, file := range result.Archive.Files {
		if strings.Contains(file.RelPath, ".png") {
			result.Archive.Files[i].Content = get1x1Png()
			continue
		}
		if strings.Contains(file.RelPath, ".jpg") || strings.Contains(file.RelPath, ".jpeg") {
			result.Archive.Files[i].Content = get1x1Jpeg()
			continue
		}
		if len(file.Content) > 100 {
			result.Archive.Files[i].Content = checksumBytes(file.Content)
		}
	}

	// Replace statement images with checksums
	for i, img := range result.Statement.Images {
		if strings.Contains(img.Fname, ".png") {
			result.Statement.Images[i].Content = get1x1Png()
			continue
		}
		if strings.Contains(img.Fname, ".jpg") || strings.Contains(img.Fname, ".jpeg") {
			result.Statement.Images[i].Content = get1x1Jpeg()
			continue
		}
		if len(img.Content) > 100 {
			result.Statement.Images[i].Content = checksumBytes(img.Content)
		}
	}

	return result
}

// checksumString returns a SHA256 checksum of the string prefixed with "checksum:"
func checksumString(s string) string {
	if s == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("checksum:%x", hash)
}

// checksumBytes returns a SHA256 checksum of the bytes prefixed with "checksum:"
func checksumBytes(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	hash := sha256.Sum256(b)
	return []byte(fmt.Sprintf("checksum:%x", hash))
}

func get1x1Png() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 255, 255, 255})
	bytesBuffer := bytes.NewBuffer(nil)
	if err := png.Encode(bytesBuffer, img); err != nil {
		panic(err)
	}
	return bytesBuffer.Bytes()
}
func get1x1Jpeg() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 255, 255, 255})

	bytesBuffer := bytes.NewBuffer(nil)
	opts := &jpeg.Options{Quality: 100}
	if err := jpeg.Encode(bytesBuffer, img, opts); err != nil {
		panic(err)
	}
	return bytesBuffer.Bytes()
}
