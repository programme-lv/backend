package http

import (
	"context"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"github.com/programme-lv/backend/submsrvc"
)

func (httpserver *HttpServer) listProgrammingLangs(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type listProgLangsResponse []*ProgrammingLang

	langs, err := httpserver.submSrvc.ListProgrammingLanguages(context.TODO())
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	mapProgrammingLangResponse := func(lang *submsrvc.ProgrammingLang) *ProgrammingLang {
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

	mapProgLangsResponse := func(langs []submsrvc.ProgrammingLang) listProgLangsResponse {
		response := make(listProgLangsResponse, len(langs))
		for i, lang := range langs {
			response[i] = mapProgrammingLangResponse(&lang)
		}
		return response
	}

	response := mapProgLangsResponse(langs)

	writeJsonSuccessResponse(w, response)
}
