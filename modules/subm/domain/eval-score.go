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

	switch e.ScoreUnit {
	case ScoreUnitTestGroup:
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
	case ScoreUnitTest:
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
		ReceivedScore: gotScore,
		PossibleScore: maxScore,
		MaxCpuMs:      maxCpuMs,
		MaxMemKiB:     maxMemKiB,
		ExceededCpu:   exceededCpu,
		ExceededMem:   exceededMem,
	}
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
