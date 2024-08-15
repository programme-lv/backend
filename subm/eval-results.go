package subm

import (
	"encoding/json"
	"log"
)

func (s *submissionssrvc) processEvalResult(evalUuid string, msgType string, fields *json.RawMessage) {
	log.Printf("processing eval result: %s, %s, %+v", evalUuid, msgType, fields)
	switch msgType {
	case "started_evaluation":
		//TODO: implement
	case "started_compilation":
		//TODO: implement
	case "finished_compilation":
		//TODO: implement
	case "started_testing":
		//TODO: implement
	case "started_test":
		//TODO: implement
	case "ignored_test":
	//TODO: implement
	case "finished_test":
		//TODO: implement
	case "finished_testing":
		//TODO: implement
	case "finished_evaluation":
		// TODO: implement
	}
}
