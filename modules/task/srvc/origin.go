package srvc

import (
	"fmt"
	"strconv"
	"strings"
)

func applyOriginNotes(task *Task) {
	long, _, ok := lioOriginNotes(
		task.OriginOlympiad, task.OriginYear, task.OlympStage, task.OriginDivisions,
	)
	if !ok {
		return
	}
	notes := make([]OriginNote, 0, len(task.OriginNotes)+1)
	for _, note := range task.OriginNotes {
		if note.Lang != "lv" {
			notes = append(notes, note)
		}
	}
	task.OriginNotes = append(notes, OriginNote{Lang: "lv", Info: long})
}

func applyPreviewOriginNotes(preview *TaskPreview) {
	long, short, ok := lioOriginNotes(
		preview.OriginOlympiad, preview.OriginYear,
		preview.OlympStage, preview.OriginDivisions,
	)
	if ok {
		preview.OriginNote = long
		preview.OriginNoteShort = short
	}
}

func lioOriginNotes(
	olymp, rawYear, stage string, divisions []string,
) (string, string, bool) {
	year, ok := lioEditionYear(rawYear)
	stageName, stageOK := lioStageName(stage)
	if olymp != "LIO" || !ok || !stageOK {
		return "", "", false
	}
	short := fmt.Sprintf("Uzdevums no %d.\u00a0g. LIO %s kārtas", year, stageName)
	long := fmt.Sprintf(
		"Uzdevums no Latvijas %d. (%d./%d. m.g.) informātikas olimpiādes (LIO) %s kārtas",
		year-1987, year-1, year, stageName,
	)
	return long + longDivisionSuffix(divisions), short + shortDivisionSuffix(divisions), true
}

func lioEditionYear(raw string) (int, bool) {
	parts := strings.Split(raw, "/")
	value := parts[len(parts)-1]
	year, err := strconv.Atoi(value)
	if err != nil || year < 1988 {
		return 0, false
	}
	if len(parts) == 2 {
		first, firstErr := strconv.Atoi(parts[0])
		if firstErr != nil || first+1 != year {
			return 0, false
		}
	} else if len(parts) != 1 {
		return 0, false
	}
	return year, true
}

func lioStageName(stage string) (string, bool) {
	name, ok := map[string]string{
		"school": "skolas", "municipal": "novada",
		"national": "valsts", "selection": "atlases",
	}[stage]
	return name, ok
}

func shortDivisionSuffix(divisions []string) string {
	switch divisionKind(divisions) {
	case "junior":
		return "; 8.-10. klasei."
	case "senior":
		return "; 11.-12. klasei."
	case "both":
		return "; abām grupām."
	default:
		return "."
	}
}

func longDivisionSuffix(divisions []string) string {
	switch divisionKind(divisions) {
	case "junior":
		return "; jaunākajai (8.-10. klašu) grupai."
	case "senior":
		return "; vecākajai (11.-12. klašu) grupai."
	case "both":
		return "; abām grupām."
	default:
		return "."
	}
}

func divisionKind(divisions []string) string {
	junior, senior := false, false
	for _, division := range divisions {
		junior = junior || division == "junior"
		senior = senior || division == "senior"
	}
	if junior && senior && len(divisions) == 2 {
		return "both"
	}
	if junior && len(divisions) == 1 {
		return "junior"
	}
	if senior && len(divisions) == 1 {
		return "senior"
	}
	return ""
}
