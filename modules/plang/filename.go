package plang

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ProgrLangIDFromFilename picks the default enabled language for a solution
// filename. TaskZip solutions carry only a filename, so the extension is the
// only stable hint.
func ProgrLangIDFromFilename(fname string) (string, error) {
	ext := strings.ToLower(filepath.Ext(fname))
	switch ext {
	case ".py":
		return "python3.13", nil
	case ".cpp", ".cc", ".cxx", ".c++":
		return "cpp17", nil
	case ".go":
		return "go1.25", nil
	case ".java":
		return "java25", nil
	default:
		return "", ErrInvalidProgLang.WithMsg(
			fmt.Sprintf("nevar noteikt programmēšanas valodu pēc faila paplašinājuma '%s'", ext),
		)
	}
}
