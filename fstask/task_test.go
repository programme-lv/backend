package fstask_test

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var prjRootPath = filepath.Join(".", "..")
var testTaskPath = filepath.Join(prjRootPath, "fstask", "testdata", "kvadrputekl")

func TestReadingWritingTests(t *testing.T) {
	parsedTask, err := fstask.Read(testTaskPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	parsedTests := parsedTask.GetTestsSortedByID()
	require.Equal(t, 6, len(parsedTests))

	parsedTestNames := []string{}
	for i := 0; i < 6; i++ {
		filename := parsedTask.GetTestFilenameFromID(parsedTests[i].ID)
		parsedTestNames = append(parsedTestNames, filename)
	}
	expectedTestNames := []string{"kp01a", "kp01b", "kp01c", "kp02a", "kp02b", "kp02c"}
	assert.Equal(t, expectedTestNames, parsedTestNames)

	parsedIDs := []int{}
	for i := 0; i < 6; i++ {
		parsedIDs = append(parsedIDs, parsedTests[i].ID)
	}
	expectedIDs := []int{1, 2, 3, 4, 5, 6}
	assert.Equal(t, expectedIDs, parsedIDs)

	parsedInputs := []string{}
	for i := 0; i < 6; i++ {
		parsedInputs = append(parsedInputs, string(parsedTests[i].Input))
	}

	testPath := filepath.Join(testTaskPath, "tests")
	expectedInputs := []string{}
	for i := 0; i < 6; i++ {
		filename := parsedTask.GetTestFilenameFromID(parsedTests[i].ID)
		inPath := filepath.Join(testPath, fmt.Sprintf("%s.in", filename))

		in, err := os.ReadFile(inPath)
		require.NoErrorf(t, err, "failed to read input file: %v", err)

		expectedInputs = append(expectedInputs, string(in))
	}
	assert.Equal(t, expectedInputs, parsedInputs)

	parsedAnswers := []string{}
	for i := 0; i < 6; i++ {
		parsedAnswers = append(parsedAnswers, string(parsedTests[i].Answer))
	}
	expectedAnsers := []string{}
	for i := 0; i < 6; i++ {
		filename := parsedTask.GetTestFilenameFromID(parsedTests[i].ID)
		ansPath := filepath.Join(testPath, fmt.Sprintf("%s.out", filename))

		ans, err := os.ReadFile(ansPath)
		require.NoErrorf(t, err, "failed to read answer file: %v", err)

		expectedAnsers = append(expectedAnsers, string(ans))
	}

	assert.Equal(t, expectedAnsers, parsedAnswers)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")

	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	storedTestNames := []string{}
	tests := storedTask.GetTestsSortedByID()
	for i := 0; i < 6; i++ {
		filename := storedTask.GetTestFilenameFromID(tests[i].ID)
		storedTestNames = append(storedTestNames, filename)
	}
	assert.Equal(t, expectedTestNames, storedTestNames)

	storedIDs := []int{}
	for i := 0; i < 6; i++ {
		storedIDs = append(storedIDs, storedTask.GetTestsSortedByID()[i].ID)
	}
	assert.Equal(t, expectedIDs, storedIDs)

	storedInputs := []string{}
	for i := 0; i < 6; i++ {
		storedInputs = append(storedInputs, string(storedTask.GetTestsSortedByID()[i].Input))
	}
	assert.Equal(t, expectedInputs, storedInputs)

	storedAnswers := []string{}
	for i := 0; i < 6; i++ {
		storedAnswers = append(storedAnswers, string(storedTask.GetTestsSortedByID()[i].Answer))
	}
	assert.Equal(t, expectedAnsers, storedAnswers)

	createdTask, err := fstask.NewTask(storedTask.GetTaskName())
	require.NoErrorf(t, err, "failed to create task: %v", err)

	// set tests
	for i := 0; i < 6; i++ {
		createdTask.AddTest(parsedTests[i].Input, parsedTests[i].Answer)
		if filename := createdTask.GetTestFilenameFromID(parsedTests[i].ID); filename != "" {
			createdTask.AssignFilenameToTest(filename, parsedTests[i].ID)
		}
	}

	// compare tests
	assert.Equal(t, parsedTask.GetTestsSortedByID(), createdTask.GetTestsSortedByID())

	// shuffle test order via assigning new ids or swapping pairwise
	for i := 0; i < 10; i++ {
		a := rand.Intn(6) + 1
		b := rand.Intn(6) + 1
		createdTask.SwapTestOrder(a, b)
	}

	// store it again
	anotherOutputDir := filepath.Join(tmpDirectory, "kvadrputekl2")

	err = createdTask.Store(anotherOutputDir)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask2, err := fstask.Read(anotherOutputDir)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// compare the tests
	assert.Equal(t, createdTask.GetTestsSortedByID(), storedTask2.GetTestsSortedByID())
}

func TestReadingWritingExamples(t *testing.T) {
	parsedTask, err := fstask.Read(testTaskPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	parsedExamples := parsedTask.GetExamples()
	require.Equal(t, 2, len(parsedExamples))

	parsedExampleNames := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		parsedExampleNames = append(parsedExampleNames, *parsedExamples[i].FName)
	}
	expectedExampleNames := []string{"kp00", "kp01"}
	assert.Equal(t, expectedExampleNames, parsedExampleNames)

	parsedInputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		parsedInputs = append(parsedInputs, string(parsedExamples[i].Input))
	}

	examplePath := filepath.Join(testTaskPath, "examples")
	expectedInputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		inPath := filepath.Join(examplePath, fmt.Sprintf("%s.in", *parsedExamples[i].FName))

		in, err := os.ReadFile(inPath)
		require.NoErrorf(t, err, "failed to read input file: %v", err)

		expectedInputs = append(expectedInputs, string(in))
	}
	assert.Equal(t, expectedInputs, parsedInputs)

	parsedOutputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		parsedOutputs = append(parsedOutputs, string(parsedExamples[i].Output))
	}
	expectedOutputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		outPath := filepath.Join(examplePath, fmt.Sprintf("%s.out", *parsedExamples[i].FName))

		out, err := os.ReadFile(outPath)
		require.NoErrorf(t, err, "failed to read output file: %v", err)

		outStr := string(out)
		expectedOutputs = append(expectedOutputs, outStr)
	}
	assert.Equal(t, expectedOutputs, parsedOutputs)

	parsedNotes := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		mdNoteStr := string(parsedExamples[i].MdNote)
		parsedNotes = append(parsedNotes, mdNoteStr)
	}
	expectedNotes := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		if len(parsedExamples[i].MdNote) == 0 {
			expectedNotes = append(expectedNotes, "")
			continue
		}
		notePath := filepath.Join(examplePath, fmt.Sprintf("%s.md", *parsedExamples[i].FName))

		note, err := os.ReadFile(notePath)
		require.NoErrorf(t, err, "failed to read note file: %v", err)

		outStr := string(note)
		expectedNotes = append(expectedNotes, outStr)
	}
	require.Equal(t, expectedNotes, parsedNotes)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")

	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	storedExampleNames := []string{}
	for i := 0; i < 2; i++ {
		storedExampleNames = append(storedExampleNames, *storedTask.GetExamples()[i].FName)
	}
	assert.Equal(t, expectedExampleNames, storedExampleNames)

	storedInputs := []string{}
	for i := 0; i < 2; i++ {
		storedInputs = append(storedInputs, string(storedTask.GetExamples()[i].Input))
	}
	assert.Equal(t, expectedInputs, storedInputs)

	storedOutputs := []string{}
	for i := 0; i < 2; i++ {
		storedOutputs = append(storedOutputs, string(storedTask.GetExamples()[i].Output))
	}
	assert.Equal(t, expectedOutputs, storedOutputs)

	require.Equal(t, 2, len(storedTask.GetExamples()))
	require.Equal(t, expectedNotes[0], string(storedTask.GetExamples()[0].MdNote))
	require.Equal(t, expectedNotes[1], string(storedTask.GetExamples()[1].MdNote))

	createdTask, err := fstask.NewTask(storedTask.GetTaskName())
	if err != nil {
		t.Errorf("failed to create task: %v", err)
	}

	createdTask.AddExample([]byte(storedInputs[0]), []byte(storedOutputs[0]), []byte(expectedNotes[0]))

	// store created task
	outputDirectory2 := filepath.Join(tmpDirectory, "kvadrputekl2")
	err = createdTask.Store(outputDirectory2)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask2, err := fstask.Read(outputDirectory2)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	assert.Equal(t, storedTask2.GetExamples()[0].Input, parsedTask.GetExamples()[0].Input)
	assert.Equal(t, storedTask2.GetExamples()[0].Output, parsedTask.GetExamples()[0].Output)
}

func TestReadingWritingMetadata(t *testing.T) {
	parsedTask, err := fstask.Read(testTaskPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// Set metadata using setters
	parsedTask.SetTaskName("Kvadrātveida putekļsūcējs")
	parsedTask.ProblemTags = []string{"math", "geometry"}
	parsedTask.TaskAuthors = []string{"Author1", "Author2"}
	parsedTask.OriginOlympiad = "LIO"
	parsedTask.DifficultyOneToFive = 3

	// Verify the set metadata using getters
	assert.Equal(t, "Kvadrātveida putekļsūcējs", parsedTask.GetTaskName())
	assert.Equal(t, []string{"math", "geometry"}, parsedTask.ProblemTags)
	assert.Equal(t, []string{"Author1", "Author2"}, parsedTask.TaskAuthors)
	assert.Equal(t, "LIO", parsedTask.OriginOlympiad)
	assert.Equal(t, 3, parsedTask.DifficultyOneToFive)
	require.Equal(t, map[string]string{
		"lv": "Uzdevums parādījās Latvijas 37. informātikas olimpiādes (2023./2024. gads) skolas kārtā.",
	}, parsedTask.GetOriginNotes())

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// Verify the stored metadata using getters
	assert.Equal(t, "Kvadrātveida putekļsūcējs", storedTask.GetTaskName())
	assert.Equal(t, []string{"math", "geometry"}, storedTask.ProblemTags)
	assert.Equal(t, []string{"Author1", "Author2"}, storedTask.TaskAuthors)
	assert.Equal(t, "LIO", storedTask.OriginOlympiad)
	assert.Equal(t, 3, storedTask.DifficultyOneToFive)
	require.Equal(t, map[string]string{
		"lv": "Uzdevums parādījās Latvijas 37. informātikas olimpiādes (2023./2024. gads) skolas kārtā.",
	}, storedTask.GetOriginNotes())
}

func TestReadingWritingTestGroups(t *testing.T) {
	parsedTask, err := fstask.Read(testTaskPath)
	assert.NoErrorf(t, err, "failed to read task: %v", err)

	parsedTestGroups := parsedTask.GetTestGroupIDs()
	require.Equal(t, 2, len(parsedTestGroups))

	expectedTestGroups := []int{1, 2}
	assert.Equal(t, expectedTestGroups, parsedTestGroups)

	firstParsedTestGroup := parsedTask.GetInfoOnTestGroup(1)
	assert.Equal(t, 1, firstParsedTestGroup.GroupID)
	assert.Equal(t, 3, firstParsedTestGroup.Points)
	assert.Equal(t, 1, firstParsedTestGroup.Subtask)
	assert.Equal(t, true, firstParsedTestGroup.Public)
	assert.Equal(t, []int{1, 2, 3}, firstParsedTestGroup.TestIDs)

	assert.Equal(t, "kp01a", parsedTask.GetTestFilenameFromID(1))
	assert.Equal(t, "kp01b", parsedTask.GetTestFilenameFromID(2))
	assert.Equal(t, "kp01c", parsedTask.GetTestFilenameFromID(3))

	secondParsedTestGroup := parsedTask.GetInfoOnTestGroup(2)
	assert.Equal(t, 2, secondParsedTestGroup.GroupID)
	assert.Equal(t, 8, secondParsedTestGroup.Points)
	assert.Equal(t, 2, secondParsedTestGroup.Subtask)
	assert.Equal(t, false, secondParsedTestGroup.Public)
	assert.Equal(t, []int{4, 5, 6}, secondParsedTestGroup.TestIDs)

	assert.Equal(t, "kp02a", parsedTask.GetTestFilenameFromID(4))
	assert.Equal(t, "kp02b", parsedTask.GetTestFilenameFromID(5))
	assert.Equal(t, "kp02c", parsedTask.GetTestFilenameFromID(6))

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	writtenTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	writtenTestGroups := writtenTask.GetTestGroupIDs()
	require.Equal(t, 2, len(writtenTestGroups))

	firstWrittenTestGroup := writtenTask.GetInfoOnTestGroup(1)
	assert.Equal(t, 1, firstWrittenTestGroup.GroupID)
	assert.Equal(t, 3, firstWrittenTestGroup.Points)
	assert.Equal(t, 1, firstWrittenTestGroup.Subtask)
	assert.Equal(t, true, firstWrittenTestGroup.Public)
	assert.Equal(t, []int{1, 2, 3}, firstWrittenTestGroup.TestIDs)

	assert.Equal(t, "kp01a", writtenTask.GetTestFilenameFromID(1))
	assert.Equal(t, "kp01b", writtenTask.GetTestFilenameFromID(2))
	assert.Equal(t, "kp01c", writtenTask.GetTestFilenameFromID(3))

	secondWrittenTestGroup := writtenTask.GetInfoOnTestGroup(2)
	assert.Equal(t, 2, secondWrittenTestGroup.GroupID)
	assert.Equal(t, 8, secondWrittenTestGroup.Points)
	assert.Equal(t, 2, secondWrittenTestGroup.Subtask)
	assert.Equal(t, false, secondWrittenTestGroup.Public)
	assert.Equal(t, []int{4, 5, 6}, secondWrittenTestGroup.TestIDs)

	assert.Equal(t, "kp02a", writtenTask.GetTestFilenameFromID(4))
	assert.Equal(t, "kp02b", writtenTask.GetTestFilenameFromID(5))
	assert.Equal(t, "kp02c", writtenTask.GetTestFilenameFromID(6))

	createdTask, err := fstask.NewTask(writtenTask.GetTaskName())
	require.NoErrorf(t, err, "should have failed to create task: %v", err)

	createdTask.AddTestGroup(3, true, []int{7, 8, 9}, 1)

	assert.Equal(t, 1, createdTask.GetInfoOnTestGroup(1).GroupID)
	assert.Equal(t, 3, createdTask.GetInfoOnTestGroup(1).Points)
	assert.Equal(t, 1, createdTask.GetInfoOnTestGroup(1).Subtask)
	assert.Equal(t, true, createdTask.GetInfoOnTestGroup(1).Public)
	assert.Equal(t, []int{7, 8, 9}, createdTask.GetInfoOnTestGroup(1).TestIDs)
}

func TestReadingWritingPDFStatement(t *testing.T) {
	parsedTask, err := fstask.Read(testTaskPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	expectedPdfPath := filepath.Join(testTaskPath, "statements", "pdf", "lv.pdf")
	expectedPdf, err := os.ReadFile(expectedPdfPath)
	require.NoErrorf(t, err, "failed to read PDF file: %v", err)

	actualPdf, err := parsedTask.GetPDFStatement("lv")
	require.NoErrorf(t, err, "failed to get PDF statement: %v", err)

	assert.Equal(t, expectedPdf, actualPdf)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	actualPdf2, err := storedTask.GetPDFStatement("lv")
	require.NoErrorf(t, err, "failed to get PDF statement: %v", err)
	assert.Equal(t, expectedPdf, actualPdf2)
}

func TestReadingWritingMDStatements(t *testing.T) {
	parsedTask, err := fstask.Read(testTaskPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// compare markdown story to parsed one
	mdLvDir := filepath.Join(testTaskPath, "statements", "md", "lv")
	InputMdPath := filepath.Join(mdLvDir, "input.md")
	OutputMdPath := filepath.Join(mdLvDir, "output.md")
	StoryMdPath := filepath.Join(mdLvDir, "story.md")
	ScoringMDPath := filepath.Join(mdLvDir, "scoring.md")

	inputMdBytes, err := os.ReadFile(InputMdPath)
	require.NoErrorf(t, err, "failed to read input.md file: %v", err)
	inputMd := string(inputMdBytes)

	outputMdBytes, err := os.ReadFile(OutputMdPath)
	require.NoErrorf(t, err, "failed to read output.md file: %v", err)
	outputMd := string(outputMdBytes)

	storyMdBytes, err := os.ReadFile(StoryMdPath)
	require.NoErrorf(t, err, "failed to read story.md file: %v", err)
	storyMd := string(storyMdBytes)

	scoringMdBytes, err := os.ReadFile(ScoringMDPath)
	require.NoErrorf(t, err, "failed to read scoring.md file: %v", err)
	scoringMd := string(scoringMdBytes)

	parsedMdStatements := parsedTask.GetMarkdownStatements()
	require.Equal(t, 1, len(parsedMdStatements))
	for _, mdStatement := range parsedMdStatements {
		lang := mdStatement.Language
		require.NotNil(t, lang)
		require.Equal(t, "lv", lang)

		require.Equal(t, inputMd, mdStatement.Input)
		require.Equal(t, outputMd, mdStatement.Output)
		require.Equal(t, storyMd, mdStatement.Story)
		require.Equal(t, scoringMd, *mdStatement.Scoring)
		require.Nil(t, mdStatement.Notes)
	}

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	parsedMdStatements2 := storedTask.GetMarkdownStatements()
	require.Equal(t, 1, len(parsedMdStatements2))
	for _, mdStatement := range parsedMdStatements2 {
		lang := mdStatement.Language
		require.NotNil(t, lang)
		require.Equal(t, "lv", lang)
		require.Equal(t, inputMd, mdStatement.Input)
		require.Equal(t, outputMd, mdStatement.Output)
		require.Equal(t, storyMd, mdStatement.Story)
		require.Equal(t, scoringMd, *mdStatement.Scoring)
		require.Nil(t, mdStatement.Notes)
	}
}

func TestReadingWritingIllustrationImage(t *testing.T) {
	parsedTask, err := fstask.Read(testTaskPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// read illustration image
	imgPath := filepath.Join(testTaskPath, "assets", "illustration.png")
	imgAsset2, err := os.ReadFile(imgPath)
	require.NoErrorf(t, err, "failed to read illustration image: %v", err)

	parsedImg := parsedTask.GetTaskIllustrationImage()
	require.NotNil(t, parsedImg)
	expectedImgAsset := &fstask.Asset{
		RelativePath: "illustration.png",
		Content:      imgAsset2,
	}
	require.Equal(t, len(parsedTask.GetAssets()), 3)
	require.Equal(t, expectedImgAsset.Content, parsedImg.Content)
	require.Equal(t, expectedImgAsset.RelativePath, parsedImg.RelativePath)
	require.Equal(t, expectedImgAsset, parsedImg)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	parsedImgAsset2 := storedTask.GetTaskIllustrationImage()
	require.NotNil(t, parsedImgAsset2)
	require.Equal(t, expectedImgAsset, parsedImgAsset2)
}
