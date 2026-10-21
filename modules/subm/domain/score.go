package domain

// ScoreBarInfo contains the percentage distribution of test results
type ScoreBarInfo struct {
	Green  int // accepted tests
	Red    int // wrong/failed tests
	Gray   int // not reached tests
	Yellow int // in progress tests
	Purple int // evaluation error
}

// ScoreInfo contains all scoring related information for an evaluation
type ScoreInfo struct {
	ScoreBar      ScoreBarInfo
	ReceivedScore int
	PossibleScore int
	// Tenths preserve one decimal of precision for partial interactive tests.
	ReceivedScoreTenths int
	PossibleScoreTenths int
	MaxCpuMs            int
	MaxMemKiB           int
	ExceededCpu         bool
	ExceededMem         bool
}

// CalculateScore calculates scoring information from an evaluation
func (e *Eval) CalculateScore() ScoreInfo {
	gotCenti := 0
	maxScore := 0
	green := 0
	red := 0
	gray := 0
	yellow := 0
	purple := 0

	switch e.ScoreUnit {
	case ScoreUnitTestGroup:
		for _, testGroup := range e.Groups {
			maxScore += testGroup.Points
			if e.Error != nil {
				continue
			}
			centi := unitCenti(testGroup.Points, testGroup.TgTests, e.Tests, e.Interactor != nil)
			gotCenti += centi
			switch {
			case allUnreached(testGroup.TgTests, e.Tests):
				gray += testGroup.Points
			case centi == testGroup.Points*100:
				green += testGroup.Points
			case centi == 0:
				red += testGroup.Points
			default:
				yellow += testGroup.Points
			}
		}
	case ScoreUnitSubtask:
		for _, subtask := range e.Subtasks {
			maxScore += subtask.Points
			if e.Error != nil {
				continue
			}
			centi := unitCenti(subtask.Points, subtask.StTests, e.Tests, e.Interactor != nil)
			gotCenti += centi
			switch {
			case allUnreached(subtask.StTests, e.Tests):
				gray += subtask.Points
			case centi == subtask.Points*100:
				green += subtask.Points
			case centi == 0:
				red += subtask.Points
			default:
				yellow += subtask.Points
			}
		}
	case ScoreUnitTest:
		maxScore = len(e.Tests)
		if e.Error == nil {
			for _, test := range e.Tests {
				centi := testPercent(test)
				gotCenti += centi
				switch {
				case test.Ac:
					green++
				case centi > 0:
					yellow++
				case test.Wa || test.Tle || test.Mle || test.Re:
					red++
				case test.Reached:
					yellow++
				default:
					gray++
				}
			}
		}
	}
	if e.Error != nil {
		purple = 100
	}

	normalizeColors(&green, &red, &gray, &yellow, &purple)

	maxCpuMs := 0
	maxMemKiB := 0
	for _, test := range e.Tests {
		if test.CpuMs != nil && *test.CpuMs > maxCpuMs {
			maxCpuMs = *test.CpuMs
		}
	}
	for _, test := range e.Tests {
		if test.MemKiB != nil && *test.MemKiB > maxMemKiB {
			maxMemKiB = *test.MemKiB
		}
	}

	exceededCpu := false
	exceededMem := false
	if maxCpuMs > e.CpuLimMs {
		exceededCpu = true
	}
	if maxMemKiB > e.MemLimKiB {
		exceededMem = true
	}

	if exceededCpu {
		maxCpuMs = e.CpuLimMs
	}
	if exceededMem {
		maxMemKiB = e.MemLimKiB
	}

	return ScoreInfo{
		ScoreBar: ScoreBarInfo{
			Green:  green,
			Red:    red,
			Gray:   gray,
			Yellow: yellow,
			Purple: purple,
		},
		ReceivedScore:       (gotCenti + 50) / 100,
		PossibleScore:       maxScore,
		ReceivedScoreTenths: (gotCenti + 5) / 10,
		PossibleScoreTenths: maxScore * 10,
		MaxCpuMs:            maxCpuMs,
		MaxMemKiB:           maxMemKiB,
		ExceededCpu:         exceededCpu,
		ExceededMem:         exceededMem,
	}
}

func testPercent(test Test) int {
	if test.Ac {
		return 100
	}
	if test.Score < 0 {
		return 0
	}
	if test.Score > 100 {
		return 100
	}
	return test.Score
}

// Percent returns the test score percentage from 0 to 100.
func (test Test) Percent() int {
	return testPercent(test)
}

// UnitTenths returns the score of a subtask or group in tenths of a point.
func (e *Eval) UnitTenths(points int, testIDs []int) int {
	return (unitCenti(points, testIDs, e.Tests, e.Interactor != nil) + 5) / 10
}

// unitCenti returns the unit score in hundredths of a point. Interactive
// tests may earn a fraction of the unit; simple and checker tests are
// all-or-nothing.
func unitCenti(points int, testIDs []int, tests []Test, partial bool) int {
	if len(testIDs) == 0 {
		return 0
	}
	sum := 0
	for _, id := range testIDs {
		if id >= 1 && id <= len(tests) {
			percent := testPercent(tests[id-1])
			if !partial && percent != 100 {
				return 0
			}
			sum += percent
		}
	}
	return (points*sum + len(testIDs)/2) / len(testIDs)
}

func allUnreached(testIDs []int, tests []Test) bool {
	if len(testIDs) == 0 {
		return false
	}
	for _, id := range testIDs {
		if id >= 1 && id <= len(tests) && tests[id-1].Reached {
			return false
		}
	}
	return true
}

func normalizeColors(green *int, red *int, gray *int, yellow *int, purple *int) {
	total := *green + *red + *gray + *yellow + *purple
	newGreen := *green * 100 / total
	newRed := *red * 100 / total
	newYellow := *yellow * 100 / total
	newPurple := *purple * 100 / total
	newGray := *gray * 100 / total
	for newGreen+newRed+newYellow+newPurple+newGray < 100 {
		if *yellow > 0 {
			newYellow += 1
		} else if *purple > 0 {
			newPurple += 1
		} else if *gray > 0 {
			newGray += 1
		} else if *red > 0 {
			newRed += 1
		} else if *green > 0 {
			newGreen += 1
		} else {
			newGray += 1
		}
	}
	*green = newGreen
	*red = newRed
	*yellow = newYellow
	*purple = newPurple
	*gray = newGray
}
