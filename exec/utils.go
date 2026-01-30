package exec

import (
	"github.com/programme-lv/backend/plang"
)

func getPrLangById(id string) (PrLang, error) {
	lang, err := plang.GetProgrLangById(id)
	if err != nil {
		return PrLang{}, err
	}
	return PrLang{
		ShortId:   lang.ID,
		Display:   lang.FullName,
		CodeFname: lang.CodeFilename,
		CompCmd:   lang.CompileCmd,
		CompFname: lang.CompiledFilename,
		ExecCmd:   lang.ExecuteCmd,
	}, nil
}
