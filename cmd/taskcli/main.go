package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/tasksrvc"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	dir := flag.String("dir", "", "directory path")
	flag.Parse()

	if *dir == "" {
		fmt.Println("Please provide a directory path using the -dir flag.")
		os.Exit(1)
	}

	if err := validateDirectory(*dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Printf("Error retrieving absolute path: %v\n", err)
		os.Exit(1)
	}

	task, err := fstask.Read(absPath)
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(initialModel(absPath, task))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func validateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist")
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}
	return nil
}

type phase int

const (
	phaseEnterTaskId phase = iota
	phaseConfirmUpload
	phaseUploadingIllustrationImg
	phaseCreatingTask
	phaseFinished
)

type model struct {
	phase   phase
	err     error
	dirPath string

	task    *fstask.Task
	wrapper *taskWrapper

	taskSrvc *tasksrvc.TaskService

	taskShortCodeIdInput textinput.Model
	illstrImgUplSpinner  spinner.Model

	illstrS3Key *string
}

func initialModel(dirPath string, fstask *fstask.Task) model {
	taskIdInput := textinput.New()
	taskIdInput.SetValue(filepath.Base(dirPath))
	taskIdInput.Focus()
	taskIdInput.CharLimit = 156
	taskIdInput.Width = 20

	illstrSpin := spinner.New()
	illstrSpin.Spinner = spinner.Dot
	illstrSpin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		phase:                phaseEnterTaskId,
		err:                  nil,
		dirPath:              dirPath,
		task:                 fstask,
		taskShortCodeIdInput: taskIdInput,
		illstrImgUplSpinner:  illstrSpin,
		taskSrvc:             tasksrvc.NewTaskSrvc(),
		illstrS3Key:          nil,
		wrapper:              newTaskWrapper(fstask),
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type uploadIllstrImgResult struct {
	err error
}

type createTaskResult struct {
	err error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case uploadIllstrImgResult:
		log.Print("uploadIllstrImgResult")
		// TODO: check for error
		if msg.err != nil {
			m.err = msg.err
			log.Println("Error uploading illustration image: ", msg.err)
			return m, tea.Quit
		}
		m.phase = phaseCreatingTask
		return m, func() tea.Msg {
			tests := m.task.GetTestsSortedByID()

			visInpStIds := m.task.GetVisibleInputSubtasks()
			visInpSts := make([]struct {
				Subtask int
				Inputs  []struct {
					TestId int
					Input  string
				}
			}, len(visInpStIds))
			for i, stId := range visInpStIds {
				visInpSts[i].Subtask = stId
				visInpSts[i].Inputs = make([]struct {
					TestId int
					Input  string
				}, 0)

				for _, tgroup := range m.task.GetTestGroups() {
					if tgroup.Subtask != stId {
						continue
					}
					for _, testId := range tgroup.TestIDs {
						for j := 0; j < len(tests); j++ {
							if tests[j].ID == testId {
								visInpSts[i].Inputs = append(visInpSts[i].Inputs,
									struct {
										TestId int
										Input  string
									}{
										TestId: testId,
										Input:  string(tests[j].Input),
									})
								break
							}
						}
					}
				}
			}

			err := m.taskSrvc.CreateTask(&tasksrvc.CreatePublicTaskInput{
				TaskCode:    m.taskShortCodeIdInput.Value(),
				FullName:    m.task.FullName,
				MemMBytes:   m.task.MemoryLimInMegabytes,
				CpuSecs:     m.task.CpuTimeLimInSeconds,
				Difficulty:  &m.task.DifficultyOneToFive,
				OriginOlymp: m.task.OriginOlympiad,
				IllustrKey:  m.illstrS3Key,
				VisInpSts:   visInpSts,
				TestGroups: []struct {
					GroupID int
					Points  int
					Public  bool
					Subtask int
					TestIDs []int
				}{},
				TestChsums: []struct {
					TestID  int
					InSha2  string
					AnsSha2 string
				}{},
				PdfSttments: []struct {
					LangIso639 string
					PdfSha2    string
				}{},
				MdSttments: []struct {
					LangIso639 string
					Story      string
					Input      string
					Output     string
					Score      string
				}{},
				ImgUuidMap: []struct {
					Uuid  string
					S3Key string
				}{},
				Examples: []struct {
					ExampleID int
					Input     string
					Output    string
				}{},
				OriginNotes: []struct {
					LangIso639 string
					OgInfo     string
				}{},
			})

			return createTaskResult{err: err}
		}
	case createTaskResult:
		// TODO: check for error
		if msg.err != nil {
			m.err = msg.err
			log.Println("Error creating task: ", msg.err)
			return m, tea.Quit
		}
		m.phase = phaseFinished
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.phase == phaseEnterTaskId {
				m.taskShortCodeIdInput.Blur()
				m.taskShortCodeIdInput.CursorEnd()
				m.phase = phaseConfirmUpload
				return m, cmd
			}
		case tea.KeyRunes:
			switch msg.Runes[0] {
			case 'y', 'Y':
				if m.phase == phaseConfirmUpload {
					m.phase = phaseUploadingIllustrationImg
					return m, tea.Batch(m.illstrImgUplSpinner.Tick, func() tea.Msg {
						illstrImg := m.task.GetTaskIllustrationImage()
						if illstrImg != nil {
							s3key, err := uploadIllustrationImage(illstrImg, m.taskSrvc)
							if err != nil {
								return uploadIllstrImgResult{err: err}
							}
							m.illstrS3Key = &s3key
						}
						return uploadIllstrImgResult{err: nil}

					})
				}
			case 'n', 'N':
				return m, tea.Quit
			}
		}
	case spinner.TickMsg:
		m.illstrImgUplSpinner, cmd = m.illstrImgUplSpinner.Update(msg)
		return m, cmd
	}

	m.taskShortCodeIdInput, cmd = m.taskShortCodeIdInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	b := func(format string, a ...any) string {
		blueText := lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db"))
		return blueText.Render(fmt.Sprintf(format, a...))
	}

	v := func(format string, a ...any) string {
		violetText := lipgloss.NewStyle().Foreground(lipgloss.Color("#e056fd"))
		return violetText.Render(fmt.Sprintf(format, a...))
	}

	r := func(format string, a ...any) string {
		redText := lipgloss.NewStyle().Foreground(lipgloss.Color("#e74c3c")).Width(140)
		return redText.Render(fmt.Sprintf(format, a...))
	}

	res := fmt.Sprintf(`Select action:
	[X] Upload task
Directory: %s
Task preview: %s
Please enter task's short code (id) %s
`,
		b(m.dirPath),
		renderTaskPreview(m.wrapper),
		v(m.taskShortCodeIdInput.View()),
	)

	if m.phase == phaseConfirmUpload {
		res = fmt.Sprintf("%sPress %s to confirm & upload, %s to cancel & exit", res, v("Y"), v("N"))
	}

	if m.phase >= phaseUploadingIllustrationImg {
		res += "Upload progress:"
	}

	checkMark := lipgloss.NewStyle().Foreground(lipgloss.Color("#2ecc71")).SetString("✓")

	if m.phase == phaseUploadingIllustrationImg {
		res += "\n"
		res += "\t" + m.illstrImgUplSpinner.View()
		res += "Uploading illustration image to S3"
	} else if m.phase > phaseUploadingIllustrationImg {
		res += "\n"
		res += "\t" + checkMark.Render()
		res += " Illustration image uploaded"
	}

	if m.phase == phaseCreatingTask {
		res += "\n"
		res += "\t" + m.illstrImgUplSpinner.View()
		res += "Inserting task into DynamoDB"
	} else if m.phase > phaseCreatingTask {
		res += "\n"
		res += "\t" + checkMark.Render()
		res += " Task rows inserted in DDB"
	}

	if m.phase == phaseFinished {
		res += "\n"
		res += fmt.Sprintf("Task %s created!", b(m.taskShortCodeIdInput.Value()))
	}

	if m.err != nil {
		res += "\n"
		res += r(m.err.Error())
	}
	res += "\n"
	return res
}
