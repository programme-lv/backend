package http

import (
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/planglist"
)

// ProgrammingLang represents a programming language.
type ProgrammingLang struct {
	ID               string  `json:"id"`
	FullName         string  `json:"fullName"`
	CodeFilename     string  `json:"codeFilename"`
	CompileCmd       *string `json:"compileCmd"`
	ExecuteCmd       string  `json:"executeCmd"`
	EnvVersionCmd    string  `json:"envVersionCmd"`
	HelloWorldCode   string  `json:"helloWorldCode"`
	MonacoID         string  `json:"monacoId"`
	CompiledFilename *string `json:"compiledFilename"`
	Enabled          bool    `json:"enabled"`
}

func (httpserver *HttpServer) listProgrammingLangs(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type listProgLangsResponse []*ProgrammingLang

	langs, err := planglist.ListProgrammingLanguages()
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	mapProgrammingLangResponse := func(lang *planglist.ProgrammingLang) *ProgrammingLang {
		return &ProgrammingLang{
			ID:               lang.ID,
			FullName:         lang.FullName,
			CodeFilename:     lang.CodeFilename,
			CompileCmd:       lang.CompileCmd,
			ExecuteCmd:       lang.ExecuteCmd,
			EnvVersionCmd:    lang.EnvVersionCmd,
			HelloWorldCode:   lang.HelloWorldCode,
			MonacoID:         lang.MonacoId,
			CompiledFilename: lang.CompiledFilename,
			Enabled:          lang.Enabled,
		}
	}

	mapProgLangsResponse := func(langs []planglist.ProgrammingLang) listProgLangsResponse {
		response := make(listProgLangsResponse, len(langs))
		for i, lang := range langs {
			response[i] = mapProgrammingLangResponse(&lang)
		}
		return response
	}

	response := mapProgLangsResponse(langs)

	writeJsonSuccessResponse(w, response)
}
