package srvc

import (
	"sort"
	"strconv"
	"strings"
)

// OriginCount is one distinct stored origin tuple plus how many tasks share it.
type OriginCount struct {
	Olympiad  string
	Year      string
	Stage     string
	Divisions []string
	Count     int
}

// FilterTree is the olympiad → year → stage → division catalog for the task list.
type FilterTree struct {
	Olympiads []FilterOlympiad
}

// FilterOlympiad is one origin olympiad (or the "other" bucket).
type FilterOlympiad struct {
	ID    string
	Count int
	Years []FilterYear
}

// FilterYear is one edition year under an olympiad.
type FilterYear struct {
	ID     string
	Count  int
	Stages []FilterStage
}

// FilterStage is one olympiad stage under a year.
type FilterStage struct {
	ID        string
	Count     int
	Divisions []FilterDivision
}

// FilterDivision is one age-group bucket under a stage.
type FilterDivision struct {
	ID    string
	Count int
}

type accOlympiad struct {
	count int
	years map[string]*accYear
}

type accYear struct {
	count  int
	stages map[string]*accStage
}

type accStage struct {
	count     int
	divisions map[string]int
}

var stageOrder = []string{"school", "municipal", "national", "selection"}

var divisionOrder = []string{"junior", "senior", "both"}

// BuildFilterTree nests origin counts after normalizing olympiad, year, and division.
func BuildFilterTree(rows []OriginCount) FilterTree {
	olympiads := make(map[string]*accOlympiad)
	for _, row := range rows {
		if row.Count <= 0 {
			continue
		}
		olympID := NormalizeOlympiad(row.Olympiad)
		yearID := NormalizeYear(row.Year)
		stageID := strings.TrimSpace(row.Stage)
		divID := divisionKind(row.Divisions)

		o := olympiads[olympID]
		if o == nil {
			o = &accOlympiad{years: make(map[string]*accYear)}
			olympiads[olympID] = o
		}
		o.count += row.Count
		if yearID == "" {
			continue
		}
		y := o.years[yearID]
		if y == nil {
			y = &accYear{stages: make(map[string]*accStage)}
			o.years[yearID] = y
		}
		y.count += row.Count
		if stageID == "" {
			continue
		}
		s := y.stages[stageID]
		if s == nil {
			s = &accStage{divisions: make(map[string]int)}
			y.stages[stageID] = s
		}
		s.count += row.Count
		if divID == "" {
			continue
		}
		s.divisions[divID] += row.Count
	}

	ids := make([]string, 0, len(olympiads))
	for id := range olympiads {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return olympiadLess(ids[i], ids[j])
	})

	tree := FilterTree{Olympiads: make([]FilterOlympiad, 0, len(ids))}
	for _, id := range ids {
		tree.Olympiads = append(tree.Olympiads, flattenOlympiad(id, olympiads[id]))
	}
	return tree
}

func flattenOlympiad(id string, o *accOlympiad) FilterOlympiad {
	yearIDs := make([]string, 0, len(o.years))
	for id := range o.years {
		yearIDs = append(yearIDs, id)
	}
	sort.Slice(yearIDs, func(i, j int) bool {
		return yearLess(yearIDs[i], yearIDs[j])
	})
	years := make([]FilterYear, 0, len(yearIDs))
	for _, yearID := range yearIDs {
		years = append(years, flattenYear(yearID, o.years[yearID]))
	}
	return FilterOlympiad{ID: id, Count: o.count, Years: years}
}

func flattenYear(id string, y *accYear) FilterYear {
	stageIDs := make([]string, 0, len(y.stages))
	for id := range y.stages {
		stageIDs = append(stageIDs, id)
	}
	sort.Slice(stageIDs, func(i, j int) bool {
		return stageLess(stageIDs[i], stageIDs[j])
	})
	stages := make([]FilterStage, 0, len(stageIDs))
	for _, stageID := range stageIDs {
		stages = append(stages, flattenStage(stageID, y.stages[stageID]))
	}
	return FilterYear{ID: id, Count: y.count, Stages: stages}
}

func flattenStage(id string, s *accStage) FilterStage {
	divIDs := make([]string, 0, len(s.divisions))
	for id := range s.divisions {
		divIDs = append(divIDs, id)
	}
	sort.Slice(divIDs, func(i, j int) bool {
		return divisionLess(divIDs[i], divIDs[j])
	})
	divisions := make([]FilterDivision, 0, len(divIDs))
	for _, divID := range divIDs {
		divisions = append(divisions, FilterDivision{ID: divID, Count: s.divisions[divID]})
	}
	return FilterStage{ID: id, Count: s.count, Divisions: divisions}
}

func olympiadLess(a, b string) bool {
	if a == "LIO" && b != "LIO" {
		return true
	}
	if b == "LIO" && a != "LIO" {
		return false
	}
	if a == "other" && b != "other" {
		return false
	}
	if b == "other" && a != "other" {
		return true
	}
	return a < b
}

func yearLess(a, b string) bool {
	ai, aErr := strconv.Atoi(a)
	bi, bErr := strconv.Atoi(b)
	if aErr == nil && bErr == nil {
		return ai > bi
	}
	return a > b
}

func stageLess(a, b string) bool {
	ai, aOk := stageIndex(a)
	bi, bOk := stageIndex(b)
	if aOk && bOk {
		return ai < bi
	}
	if aOk {
		return true
	}
	if bOk {
		return false
	}
	return a < b
}

func stageIndex(id string) (int, bool) {
	for i, known := range stageOrder {
		if known == id {
			return i, true
		}
	}
	return 0, false
}

func divisionLess(a, b string) bool {
	return divisionIndex(a) < divisionIndex(b)
}

func divisionIndex(id string) int {
	for i, known := range divisionOrder {
		if known == id {
			return i
		}
	}
	return len(divisionOrder)
}
