package fstask

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type MarkdownStatement struct {
	Language string

	Story   string
	Input   string
	Output  string
	Notes   string
	Scoring string
	Example string
	Talk    string // communication (interactive tasks)

	ImgSizes []MdImgSize
}

type MdImgSize struct {
	ImgPath string `toml:"img_path"`
	WidthEm int    `toml:"width_em"`
}

func ReadMarkdownStatementsFromTaskDir(dir TaskDir) ([]MarkdownStatement, error) {
	requiredSpec := Version{major: 2}
	if dir.Specification.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		return nil, fmt.Errorf(format, dir.Specification.String(), requiredSpec.String())
	}

	// read img sizes from toml
	var tomlStruct struct {
		MdImgSizes []MdImgSize `toml:"md_img_sizes"`
	}
	err := toml.Unmarshal(dir.ProblemToml, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal toml: %w", err)
	}
	mdImgSizes := tomlStruct.MdImgSizes

	var markdownStatements []MarkdownStatement

	statementsDir := filepath.Join(dir.AbsPath, "statements")
	if _, err := os.Stat(statementsDir); !os.IsNotExist(err) {
		// find all files that end with .md
		markdownFiles, err := filepath.Glob(filepath.Join(statementsDir, "*.md"))
		if err != nil {
			return nil, fmt.Errorf("error finding markdown files: %w", err)
		}

		for _, file := range markdownFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("error reading markdown file '%s': %w", file, err)
			}

			statement, err := parseMdFile(string(content))
			if err != nil {
				return nil, fmt.Errorf("error parsing markdown file '%s': %w", file, err)
			}

			statement.Language = strings.TrimSuffix(filepath.Base(file), ".md")
			statement.ImgSizes = mdImgSizes

			markdownStatements = append(markdownStatements, statement)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error accessing statements directory: %w", err)
	}

	// check if duplicate language
	seen := make(map[string]bool)
	for _, statement := range markdownStatements {
		if seen[statement.Language] {
			format := "duplicate language '%s' in md statements"
			return nil, fmt.Errorf(format, statement.Language)
		}
		seen[statement.Language] = true
	}

	return markdownStatements, nil
}

func parseMdFile(content string) (MarkdownStatement, error) {
	alphabet := "Aa,Āā,Bb,Cc,Čč,Dd,Ee,Ēē,Ff,Gg,Ģģ,Hh,Ii,Īī,Jj,Kk,Ķķ,Ll,Ļļ,Mm,Nn,Ņņ,Oo,Pp,Rr,Ss,Šš,Tt,Uu,Ūū,Vv,Zz,Žž"
	statement := MarkdownStatement{}
	re := regexp.MustCompile(fmt.Sprintf(`[%s]+\n-+\n`, alphabet))
	found := re.FindAllStringIndex(content, -1)
	for i, match := range found {
		var end int
		if i == len(found)-1 {
			end = len(content)
		} else {
			end = found[i+1][0]
		}
		section := content[match[1]+1 : end]

		header := content[match[0]:match[1]]

		// returns a function that assign the section content to statement field
		f := func(x *string) func(string) {
			return func(section string) {
				*x = section
			}
		}

		prefix := map[string]func(string){
			"Stāsts":       f(&statement.Story),
			"Ievaddati":    f(&statement.Input),
			"Izvaddati":    f(&statement.Output),
			"Piezīmes":     f(&statement.Notes),
			"Vērtēšana":    f(&statement.Scoring),
			"Piemērs":      f(&statement.Example),
			"Komunikācija": f(&statement.Talk),
			"Story":        f(&statement.Story),
			"Input":        f(&statement.Input),
			"Output":       f(&statement.Output),
			"Notes":        f(&statement.Notes),
			"Scoring":      f(&statement.Scoring),
			"Example":      f(&statement.Example),
			"Interaction":  f(&statement.Talk),
		}

		for k, v := range prefix {
			if strings.HasPrefix(header, k) {
				v(strings.TrimSpace(section))
			}
		}

	}
	return statement, nil
}

func (task *Task) LoadMarkdownStatements(dir TaskDir) error {
	markdownStatements, err := ReadMarkdownStatementsFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read markdown statements: %w", err)
	}
	task.MarkdownStatements = markdownStatements
	return nil
}
