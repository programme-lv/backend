package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/httpjson"
)

func (httpserver *HttpServer) testerRun(w http.ResponseWriter, r *http.Request) {
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

	tests := make([]execsrvc.TestFile, len(req.Tests))
	for i, test := range req.Tests {
		tests[i] = execsrvc.TestFile{
			InSha256:    test.InSha256,
			InDownlUrl:  test.InUrl,
			InContent:   test.InContent,
			AnsSha256:   test.AnsContent,
			AnsDownlUrl: test.AnsUrl,
			AnsContent:  test.AnsContent,
		}
	}

	execUuid := uuid.New()
	err := httpserver.execSrvc.Enqueue(
		context.Background(),
		execUuid,
		req.SrcCode,
		req.ProgLangId,
		tests,
		execsrvc.TestingParams{
			CpuMs:      req.CpuMs,
			MemKiB:     req.MemKib,
			Checker:    req.Checker,
			Interactor: req.Interactor,
		},
	)
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	type response struct {
		EvalUUID string `json:"eval_uuid"`
	}

	res := &response{
		EvalUUID: execUuid.String(),
	}

	httpjson.WriteSuccessJson(w, res)
}

func (httpserver *HttpServer) testerListen(w http.ResponseWriter, r *http.Request) {
	type request struct {
		ExecUuid string `json:"exec_uuid"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	execUuid, err := uuid.Parse(req.ExecUuid)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	ch, err := httpserver.execSrvc.Listen(context.Background(), execUuid)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	encoder := json.NewEncoder(w)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	notify := r.Context().Done()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			w.Write([]byte("data: "))
			encoder.Encode(event)
			w.Write([]byte("\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-notify:
			return
		}
	}
}
