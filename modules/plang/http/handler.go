package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/plang"
)

type PlangHttpHandler struct{}

func NewPlangHttpHandler() *PlangHttpHandler {
	return &PlangHttpHandler{}
}

func (h *PlangHttpHandler) RegisterRoutes(r *chi.Mux) {
	r.Get("/programming-languages", h.listProgrammingLangs)
	r.Get("/langs", h.listProgrammingLangs)
}

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

func (h *PlangHttpHandler) listProgrammingLangs(w http.ResponseWriter, r *http.Request) {
	type listProgLangsResponse []*ProgrammingLang

	langs, err := plang.ListProgrammingLanguages()
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	mapProgrammingLangResponse := func(lang *plang.ProgrammingLang) *ProgrammingLang {
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

	mapProgLangsResponse := func(langs []plang.ProgrammingLang) listProgLangsResponse {
		response := make(listProgLangsResponse, len(langs))
		for i, lang := range langs {
			response[i] = mapProgrammingLangResponse(&lang)
		}
		return response
	}

	response := mapProgLangsResponse(langs)

	jsonresp.Success(w, response)
}
