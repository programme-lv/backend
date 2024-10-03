package fstask

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

type Origin struct {
	Olympiad     string
	AcademicYear string
	Stage        string
	Institution  string
	Authors      []string
	Notes        map[string]string
}

func ReadOriginFromTask(dir TaskDir) (res Origin, err error) {
	obj := struct {
		Origin struct {
			Olympiad     string            `toml:"olympiad"`
			AcademicYear string            `toml:"academic_year"`
			Stage        string            `toml:"stage"`
			Institution  string            `toml:"institution"`
			Authors      []string          `toml:"authors"`
			Notes        map[string]string `toml:"notes,omitempty"`
		} `toml:"origin"`
	}{}

	err = toml.Unmarshal(dir.ProblemToml, &obj)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal the origin: %w", err)
		return
	}
	res = Origin{
		Olympiad:     obj.Origin.Olympiad,
		AcademicYear: obj.Origin.AcademicYear,
		Stage:        obj.Origin.Stage,
		Institution:  obj.Origin.Institution,
		Authors:      obj.Origin.Authors,
		Notes:        obj.Origin.Notes,
	}
	return
}

func (task *Task) LoadOriginInformation(dir TaskDir) error {
	origin, err := ReadOriginFromTask(dir)
	if err != nil {
		return fmt.Errorf("failed to read origin: %w", err)
	}
	task.OriginOlympiad = origin.Olympiad
	task.AcademicYear = origin.AcademicYear
	task.OlympiadStage = origin.Stage
	task.OriginInstitution = origin.Institution
	task.TaskAuthors = origin.Authors
	task.OriginNotes = origin.Notes
	return nil
}
