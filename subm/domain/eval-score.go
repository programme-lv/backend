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
	MaxCpuMs      int
	MaxMemKiB     int
	ExceededCpu   bool
	ExceededMem   bool
}

// CalculateScore calculates scoring information from an evaluation
func (e *Eval) CalculateScore() ScoreInfo {
	gotScore := 0
	maxScore := 0
	green := 0
	red := 0
	gray := 0
	yellow := 0
	purple := 0

	if e.ScoreUnit == ScoreUnitTestGroup {
		for _, testGroup := range e.Groups {
			maxScore += testGroup.Points
		}
		if e.Error == nil {
			for _, testGroup := range e.Groups {
				allUncreached := true
				allAccepted := true
				hasWrong := false
				for _, testIdx := range testGroup.TgTests {
					test := e.Tests[testIdx-1]
					if test.Reached {
						allUncreached = false
					}
					if !test.Ac {
						allAccepted = false
					}
					if test.Wa || test.Tle || test.Mle || test.Re {
						hasWrong = true
					}
				}
				if allUncreached {
					gray += testGroup.Points
				} else if allAccepted {
					green += testGroup.Points
					gotScore += testGroup.Points
				} else if hasWrong {
					red += testGroup.Points
				} else {
					yellow += testGroup.Points
				}
			}
		} else {
			purple = 100
		}
	} else if e.ScoreUnit == ScoreUnitTest {
		maxScore += len(e.Tests)
		if e.Error == nil {
			for _, test := range e.Tests {
				if test.Ac {
					green += 1
					gotScore += 1
				} else if test.Wa || test.Tle || test.Mle || test.Re {
					red += 1
				} else if test.Reached {
					yellow += 1
				} else {
					gray += 1
				}
			}
		} else {
			purple = 100
		}
	}

	// Normalize percentages to sum up to 100
	total := green + red + gray + yellow + purple
	if total > 0 {
		green = green * 100 / total
		red = red * 100 / total
		yellow = yellow * 100 / total
		purple = purple * 100 / total
		gray = 100 - green - red - yellow - purple
	}

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
		ReceivedScore: gotScore,
		PossibleScore: maxScore,
		MaxCpuMs:      maxCpuMs,
		MaxMemKiB:     maxMemKiB,
		ExceededCpu:   exceededCpu,
		ExceededMem:   exceededMem,
	}
}
