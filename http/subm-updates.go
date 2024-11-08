package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/programme-lv/backend/submsrvc"
)

func (httpserver *HttpServer) listenToSubmUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	listener := newSubmUpdateListener()
	go func() {
		err := httpserver.submSrvc.StreamSubmissionUpdates(r.Context(), listener)
		if err != nil {
			log.Println(err)
			return
		}
	}()

	type SubmissionStateUpdate struct {
		SubmUUID string `json:"subm_uuid"`
		EvalUUID string `json:"eval_uuid"`
		NewState string `json:"new_state"`
	}

	type TestGroupScoreUpdate struct {
		SubmUUID      string `json:"subm_uuid"`
		EvalUUID      string `json:"eval_uuid"`
		TestGroupID   int    `json:"test_group_id"`
		AcceptedTests int    `json:"accepted_tests"`
		WrongTests    int    `json:"wrong_tests"`
		UntestedTests int    `json:"untested_tests"`
	}

	type TestsScoreUpdate struct {
		SubmUuid string `json:"subm_uuid"`
		EvalUuid string `json:"eval_uuid"`
		Accepted int    `json:"accepted"`
		Wrong    int    `json:"wrong"`
		Untested int    `json:"untested"`
	}

	type SubmissionListUpdate struct {
		SubmCreated        *BriefSubmission       `json:"subm_created"`
		StateUpdate        *SubmissionStateUpdate `json:"state_update"`
		TestGroupResUpdate *TestGroupScoreUpdate  `json:"testgroup_res_update"`
		TestsScoreUpdate   *TestsScoreUpdate      `json:"tests_score_update"`
	}

	mapTestsScoreUpdate := func(update *submsrvc.TSetScoreUpd) *TestsScoreUpdate {
		if update == nil {
			return nil
		}
		return &TestsScoreUpdate{
			SubmUuid: update.SubmUuid,
			EvalUuid: update.EvalUuid,
			Accepted: update.Accepted,
			Wrong:    update.Wrong,
			Untested: update.Untested,
		}
	}

	mapStateUpdate := func(update *submsrvc.EvalStageUpd) *SubmissionStateUpdate {
		if update == nil {
			return nil
		}
		return &SubmissionStateUpdate{
			SubmUUID: update.SubmUuid,
			EvalUUID: update.EvalUuid,
			NewState: update.NewStage,
		}
	}

	mapTestgroupResUpdate := func(update *submsrvc.TGroupScoreUpd) *TestGroupScoreUpdate {
		if update == nil {
			return nil
		}
		return &TestGroupScoreUpdate{
			SubmUUID:      update.SubmUUID,
			EvalUUID:      update.EvalUUID,
			TestGroupID:   update.TestGroupID,
			AcceptedTests: update.AcceptedTests,
			WrongTests:    update.WrongTests,
			UntestedTests: update.UntestedTests,
		}
	}

	var writeMutex sync.Mutex

	// Create a helper function for thread-safe writing
	safeWrite := func(data string) {
		writeMutex.Lock()
		defer writeMutex.Unlock()
		io.WriteString(w, data)
		flusher.Flush()
	}

	keepAliveTicker := time.NewTicker(15 * time.Second)
	defer keepAliveTicker.Stop()

	done := make(chan bool)
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-keepAliveTicker.C:
				safeWrite(": keep-alive\n\n")
			}
		}
	}()

	for update := range listener.Listen() {
		message := SubmissionListUpdate{
			SubmCreated:        mapBriefSubm(update.SubmCreated),
			StateUpdate:        mapStateUpdate(update.StateUpdate),
			TestGroupResUpdate: mapTestgroupResUpdate(update.TestgroupResUpdate),
			TestsScoreUpdate:   mapTestsScoreUpdate(update.TestsResUpdate),
		}
		marshalled, err := json.Marshal(message)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println(string(marshalled))
		safeWrite("data: " + string(marshalled) + "\n\n")
	}

	done <- true
}

type submUpdateListener struct {
	updateChan chan *submsrvc.SubmissionListUpdate
	mutex      *sync.Mutex
	closed     bool
}

func (l *submUpdateListener) Send(update *submsrvc.SubmissionListUpdate) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.closed {
		return fmt.Errorf("update listener is closed")
	}
	if len(l.updateChan) == cap(l.updateChan) {
		return fmt.Errorf("update channel is full")
	}
	l.updateChan <- update
	return nil
}

func (l *submUpdateListener) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.closed = true
	close(l.updateChan)
	return nil
}

func (l *submUpdateListener) Listen() chan *submsrvc.SubmissionListUpdate {
	return l.updateChan
}

func newSubmUpdateListener() *submUpdateListener {
	return &submUpdateListener{
		updateChan: make(chan *submsrvc.SubmissionListUpdate, 10000),
		mutex:      &sync.Mutex{},
		closed:     false,
	}
}
