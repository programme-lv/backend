package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"mime"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput" // Import textinput
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nfnt/resize"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/wailsapp/mimetype"
	"golang.org/x/exp/maps"
)

// Define upload states
type uploadState int

const (
	uploadStateEnterID uploadState = iota
	uploadStateConfirm
	uploadStateUploading
	uploadStateDone
)

// Define the upload model
type uploadModel struct {
	state       uploadState
	uplSpinner  spinner.Model
	previewObj  TaskPreview
	taskDir     string
	success     bool
	err         error
	taskIDInput textinput.Model // Add text input field
}

// Initialize a new upload model
func newUploadModel(dir string) uploadModel {
	res := uploadModel{}
	res.taskDir = dir

	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		res.err = fmt.Errorf("failed to get absolute path: %w", err)
		return res
	}
	res.taskDir = dirAbs

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db"))
	res.uplSpinner = s

	// Get task preview
	preview, err := getPreview(dir)
	res.previewObj = preview
	if err != nil {
		res.err = err
		return res
	}

	// Initialize text input for Task ID
	ti := textinput.New()
	ti.SetValue(filepath.Base(dir))
	ti.CharLimit = 26
	ti.Width = 26
	ti.Prompt = ""
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9b59b6"))
	ti.Focus() // Set focus to the input field when entering this state
	res.taskIDInput = ti

	// Set initial state to Preview
	res.state = uploadStateEnterID

	return res
}

// Initialize the model with appropriate commands
func (u uploadModel) Init() tea.Cmd {
	// Do not start the spinner here
	return nil
}

// Update function to handle messages and state transitions
func (u uploadModel) Update(msg tea.Msg) (uploadModel, tea.Cmd) {
	switch u.state {
	case uploadStateEnterID:
		// Update the text input model
		var tiCmd tea.Cmd
		u.taskIDInput, tiCmd = u.taskIDInput.Update(msg)

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC:
				return u, tea.Quit
			case tea.KeyEnter:
				// Ensure Task ID is not empty
				taskID := strings.TrimSpace(u.taskIDInput.Value())
				if taskID == "" {
					// Optionally, you can add feedback for empty input
					return u, nil
				}
				// Transition to Confirm state
				u.state = uploadStateConfirm
				u.taskIDInput.Blur()
				return u, nil
			case tea.KeyEsc:
				return u, tea.Quit
			}
		}
		return u, tiCmd

	case uploadStateConfirm:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y":
				// Start uploading
				u.state = uploadStateUploading
				return u, tea.Batch(u.uplSpinner.Tick, u.uploadTask())
			case "n", "N", "q":
				return u, tea.Quit
			case "ctrl+c":
				return u, tea.Quit
			}
		}

	case uploadStateUploading:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" {
				return u, tea.Quit
			}
		case uploadResultMsg:
			// Handle upload result
			u.err = msg.err
			u.success = msg.err == nil
			u.state = uploadStateDone
			return u, nil // Stop spinner by not returning any spinner commands
		}

	case uploadStateDone:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				return u, tea.Quit
			}
		}
	}

	var inputCmd tea.Cmd
	u.taskIDInput, inputCmd = u.taskIDInput.Update(msg)

	var spinnerCmd tea.Cmd
	u.uplSpinner, spinnerCmd = u.uplSpinner.Update(msg)

	return u, tea.Batch(inputCmd, spinnerCmd)
}

// View function to render the UI based on the current state
func (u uploadModel) View() string {
	s := "Selected action: Upload\n\n"
	s += "Task Preview:\n"
	s += u.previewObj.View()

	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9b59b6"))
	switch u.state {
	case uploadStateEnterID:
		s += fmt.Sprintf("\n\nEnter Task ID: %s\n", u.taskIDInput.View())
	case uploadStateConfirm:
		s += fmt.Sprintf("\n\nProceed with uploading task %s? (y/n)\n", valueStyle.Render(u.taskIDInput.Value()))
	case uploadStateUploading:
		s += fmt.Sprintf("\n\n%s Uploading...\n", u.uplSpinner.View())
	case uploadStateDone:
		if u.success {
			s += "\n\nUpload successful! Press enter to continue...\n"
		} else {
			s += "\n\nUpload failed! Error message: " + u.err.Error() + "\nPress enter to continue...\n"
		}
	}
	return s
}

// Define a message type for upload results
type uploadResultMsg struct {
	err error
}

// Command to handle the upload process
func (u uploadModel) uploadTask() tea.Cmd {
	return func() tea.Msg {
		taskSrvc := tasksrvc.NewTaskSrvc()
		task, err := fstask.Read(u.taskDir)
		if err != nil {
			return uploadResultMsg{err: fmt.Errorf("failed to read task: %w", err)}
		}
		// Retrieve the entered Task ID
		taskId := strings.TrimSpace(u.taskIDInput.Value())

		var illstrS3ObjKey *string = nil
		illstrImg := task.GetTaskIllustrationImage()
		if illstrImg != nil {
			s3Key, err := uploadIllustrationImage(illstrImg, taskSrvc)
			if err != nil {
				return uploadResultMsg{err: fmt.Errorf("failed to upload illustration image: %w", err)}
			}
			illstrS3ObjKey = &s3Key
		}

		// Process markdown statements
		mdImgUuidMap := make(map[string]string) // uuid -> original image url
		mdSttmntsWithMappedImgs := make([]tasksrvc.MarkdownStatement, len(task.GetMarkdownStatements()))
		for i, sttmnt := range task.GetMarkdownStatements() {
			story, storyImgUuidMap := mapMarkdownImageURLs(sttmnt.Story)
			maps.Copy(mdImgUuidMap, storyImgUuidMap)

			input, inputImgUuidMap := mapMarkdownImageURLs(sttmnt.Input)
			maps.Copy(mdImgUuidMap, inputImgUuidMap)

			output, outputImgUuidMap := mapMarkdownImageURLs(sttmnt.Output)
			maps.Copy(mdImgUuidMap, outputImgUuidMap)

			res := tasksrvc.MarkdownStatement{
				LangISO639: sttmnt.Language,
				Story:      story,
				Input:      input,
				Output:     output,
				Notes:      nil,
				Scoring:    nil,
			}

			if sttmnt.Notes != nil {
				notesStr := *sttmnt.Notes
				notes, notesImgUuidMap := mapMarkdownImageURLs(notesStr)
				maps.Copy(mdImgUuidMap, notesImgUuidMap)

				res.Notes = &notes
			}

			if sttmnt.Scoring != nil {
				scoringStr := *sttmnt.Scoring
				scoring, scoringImgUuidMap := mapMarkdownImageURLs(scoringStr)
				maps.Copy(mdImgUuidMap, scoringImgUuidMap)

				res.Scoring = &scoring
			}

			mdSttmntsWithMappedImgs[i] = res
		}

		// Upload markdown images
		imgUuidToS3KeyMap := make(map[string]string)
		assets := task.GetAssets()
		for k, v := range mdImgUuidMap {
			// k - uuid, v - original image url
			found := false
			for _, asset := range assets {
				if asset.RelativePath == v {
					found = true
					extMediaType := mime.TypeByExtension(filepath.Ext(asset.RelativePath))
					s3Key, err := taskSrvc.UploadMarkdownImage(extMediaType, asset.Content)
					if err != nil {
						return uploadResultMsg{err: fmt.Errorf("failed to upload markdown image: %w", err)}
					}

					imgUuidToS3KeyMap[k] = s3Key
				}
			}
			if !found {
				log.Fatalf("Failed to find asset corresponding to URL: %s", v)
			}
		}

		// Upload PDF statements
		for _, pdf := range task.GetPdfStatements() {
			_, err = taskSrvc.UploadStatementPdf(pdf.Content)
			if err != nil {
				return uploadResultMsg{err: fmt.Errorf("failed to upload pdf statement: %w", err)}
			}
		}

		// Build the input for creating a task
		putTaskInput := buildPutTaskInput(taskId, task, illstrS3ObjKey, mdSttmntsWithMappedImgs, imgUuidToS3KeyMap)
		err = taskSrvc.PutTask(putTaskInput)

		if err != nil {
			return uploadResultMsg{err: fmt.Errorf("failed to create task: %w", err)}
		}

		return uploadResultMsg{err: nil}
	}
}

// Build the input for creating a public task, including the Task ID
func buildPutTaskInput(
	taskId string,
	task *fstask.Task,
	illstrS3Key *string,
	mdSttmnts []tasksrvc.MarkdownStatement,
	imgUuidToS3KeyMap map[string]string,
) *tasksrvc.PutPublicTaskInput {
	visInpStasks := make([]tasksrvc.VisInpSt, len(task.GetVisibleInputSubtaskIds()))
	for i, stId := range task.GetVisibleInputSubtaskIds() {
		visInpStasks[i] = tasksrvc.VisInpSt{
			Subtask: stId,
			Inputs:  []tasksrvc.TestWithOnlyInput{},
		}
		for _, test := range task.GetTestsSortedByID() {
			for _, tGroup := range task.GetTestGroups() {
				if tGroup.Subtask == stId {
					testInTGroup := false
					for _, testID := range tGroup.TestIDs {
						if test.ID == testID {
							testInTGroup = true
							break
						}
					}
					if !testInTGroup {
						continue
					}
					alreadyAdded := false
					for _, addedInput := range visInpStasks[i].Inputs {
						if addedInput.TestID == test.ID {
							alreadyAdded = true
							break
						}
					}
					if alreadyAdded {
						continue
					}
					visInpStasks[i].Inputs = append(visInpStasks[i].Inputs, tasksrvc.TestWithOnlyInput{
						TestID: test.ID,
						Input:  string(test.Input),
					})
				}
			}
		}
	}

	testGroups := make([]tasksrvc.TestGroup, len(task.GetTestGroupIDs()))
	for i, tGroup := range task.GetTestGroups() {
		testGroups[i] = tasksrvc.TestGroup{
			GroupID: tGroup.GroupID,
			Points:  tGroup.Points,
			Public:  tGroup.Public,
			Subtask: tGroup.Subtask,
			TestIDs: tGroup.TestIDs,
		}
	}

	testChsums := make([]tasksrvc.TestChecksum, len(task.GetTestsSortedByID()))
	for i, test := range task.GetTestsSortedByID() {
		testChsums[i] = tasksrvc.TestChecksum{
			TestID:  test.ID,
			InSHA2:  sha2Hex(test.Input),
			AnsSHA2: sha2Hex(test.Answer),
		}
	}

	pdfSttmnts := make([]tasksrvc.PdfStatement, len(task.GetPdfStatements()))
	for i, pdfSttmnt := range task.GetPdfStatements() {
		pdfSttmnts[i] = tasksrvc.PdfStatement{
			LangISO639: pdfSttmnt.Language,
			PdfSHA2:    sha2Hex(pdfSttmnt.Content),
		}
	}

	imgUuidToS3MapSlice := make([]tasksrvc.ImageUUIDMap, 0)
	for k, v := range imgUuidToS3KeyMap {
		imgUuidToS3MapSlice = append(imgUuidToS3MapSlice, tasksrvc.ImageUUIDMap{
			UUID:  k,
			S3Key: v,
		})
	}

	examples := make([]tasksrvc.Example, 0)
	for i, example := range task.GetExamples() {
		examples = append(examples, tasksrvc.Example{
			ExampleID: i + 1,
			Input:     string(example.Input),
			Output:    string(example.Output),
			MdNote:    string(example.MdNote),
		})
	}

	originNotes := make([]tasksrvc.OriginNote, 0)
	for lang, note := range task.GetOriginNotes() {
		originNotes = append(originNotes, tasksrvc.OriginNote{
			LangISO639: lang,
			OgInfo:     note,
		})
	}

	return &tasksrvc.PutPublicTaskInput{
		TaskCode:    taskId, // Assign the entered Task ID
		FullName:    task.FullName,
		MemMBytes:   task.MemoryLimInMegabytes,
		CpuSecs:     task.CpuTimeLimInSeconds,
		Difficulty:  &task.DifficultyOneToFive,
		OriginOlymp: task.OriginOlympiad,
		IllustrKey:  illstrS3Key,
		VisInpSts:   visInpStasks,
		TestGroups:  testGroups,
		TestChsums:  testChsums,
		PdfSttments: pdfSttmnts,
		MdSttments:  mdSttmnts,
		ImgUuidMap:  imgUuidToS3MapSlice,
		Examples:    examples,
		OriginNotes: originNotes,
	}
}

func sha2Hex(body []byte) (sha2 string) {
	hash := sha256.Sum256(body)
	sha2 = fmt.Sprintf("%x", hash[:])
	return
}

// uploadIllustrationImage uploads the illustration image to S3 and returns the S3 key.
//
// It takes a fstask.Asset and a tasksrvc.TaskService as arguments. The fstask.Asset
// must contain the image data and file extension. The tasksrvc.TaskService is used to
// upload the image to S3.
//
// The function returns the S3 key for the uploaded image and an error if something
// goes wrong.
func uploadIllustrationImage(asset *fstask.Asset, taskService *tasksrvc.TaskService) (string, error) {
	compressedImage, err := compressImage(asset.Content, 600)
	if err != nil {
		return "", fmt.Errorf("failed to compress image: %w", err)
	}

	mType := mime.TypeByExtension(filepath.Ext(asset.RelativePath))
	if mType == "" {
		detectedType := mimetype.Detect(compressedImage)
		if detectedType == nil {
			return "", fmt.Errorf("failed to detect file type")
		}
		mType = detectedType.String()
	}

	s3Key, err := taskService.UploadIllustrationImg(mType, compressedImage)
	if err != nil {
		return "", fmt.Errorf("failed to upload illustration to S3: %w", err)
	}

	return s3Key, nil
}

// compressImage resizes and compresses the image to the specified maximum width.
// It returns the compressed image bytes or an error if the process fails.
func compressImage(imgContent []byte, maxWidth uint) ([]byte, error) {
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
