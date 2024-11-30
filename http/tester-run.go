package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
)

type TesterRunResponse struct {
	EvalUUID string `json:"eval_uuid"`
}

func (httpserver *HttpServer) testerRun(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	type test struct {
		InSha256   *string `json:"in_sha256"`
		InUrl      *string `json:"in_url"`
		InContent  *string `json:"in_content"`
		AnsSha256  *string `json:"ans_sha256"`
		AnsUrl     *string `json:"ans_url"`
		AnsContent *string `json:"ans_content"`
	}

	type request struct {
		ApiKey     string  `json:"api_key"`
		SrcCode    string  `json:"src_code"`
		ProgLangId string  `json:"lang_id"`
		CpuMs      int     `json:"cpu_ms"`
		MemKib     int     `json:"mem_kib"`
		Tests      []test  `json:"tests"`
		Checker    *string `json:"checker"`
		Interactor *string `json:"interactor"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	tests := make([]evalsrvc.Test, len(req.Tests))
	for i, test := range req.Tests {
		tests[i] = evalsrvc.Test{
			ID:         i + 1,
			InSha256:   test.InSha256,
			InUrl:      test.InUrl,
			InContent:  test.InContent,
			AnsSha256:  test.AnsContent,
			AnsUrl:     test.AnsUrl,
			AnsContent: test.AnsContent,
		}
	}

	uuid, err := httpserver.evalSrvc.EnqueueExternal(req.ApiKey, evalsrvc.NewEvalParams{
		Code:       req.SrcCode,
		LangId:     req.ProgLangId,
		Tests:      tests,
		CpuMs:      req.CpuMs,
		MemKiB:     req.MemKib,
		Checker:    req.Checker,
		Interactor: req.Interactor,
	})
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	res := &TesterRunResponse{
		EvalUUID: uuid.String(),
	}

	writeJsonSuccessResponse(w, res)
}

func (httpserver *HttpServer) testerRunLongPoll(w http.ResponseWriter, r *http.Request) {
	logger := httplog.LogEntry(r.Context())

	evalUuidStr := chi.URLParam(r, "evalUuid")
	evalUuid, err := uuid.Parse(evalUuidStr)
	if err != nil {
		writeJsonErrorResponse(w,
			"Nederīgs UUID",
			http.StatusBadRequest,
			"invalid_eval_uuid")
		return
	}

	received, err := httpserver.evalSrvc.ReceiveFor(evalUuid)
	if err != nil {
		handleJsonSrvcError(logger, w, err)
		return
	}

	data := make([]interface{}, len(received))
	for i, msg := range received {
		marshalled, _ := json.Marshal(msg.Data)
		values := make(map[string]interface{})
		json.Unmarshal(marshalled, &values)
		values["msg_type"] = msg.Data.Type()
		delete(values, "sys_info")
		data[i] = values
	}

	writeJsonSuccessResponse(w, data)
}
