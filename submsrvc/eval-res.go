package submsrvc

import (
	"encoding/json"
	"log"
)

func (s *SubmissionSrvc) processEvalResult(evalUuid string, msgType string, fields *json.RawMessage) {
	_, err := s.getSubmUuidByEvalUuid(evalUuid)
	if err != nil {
		log.Printf("failed to get subm_uuid by eval_uuid: %v", err)
		return
	}
	panic("not implemented")
}

func (s *SubmissionSrvc) getSubmUuidByEvalUuid(evalUuid string) (string, error) {
	panic("not implemented")
}
