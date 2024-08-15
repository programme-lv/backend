package subm

import (
	"encoding/json"
	"log"
)

func (s *submissionssrvc) processEvalResult(evalUuid string, msgType string, fields *json.RawMessage) {
	switch msgType {
	case "started_evaluation":
		var parsed struct {
			SystemInfo string `json:"system_info"`
			// StartedTime string `json:"started_time"`
		}
		err := json.Unmarshal(*fields, &parsed)
		if err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}
		log.Printf("received \"started_evaluation\" message: %+v", parsed)
		//TODO: implement
		// 1. save the system info in
		// 2. evaluation stage should change from "waiting" to "received" if it is still "waiting"
		// 3. update the submission details row with new evaluation result
		// err = s.ddbSubmTable.submTable.Update("eval_uuid", evalUuid).
		// 	Set("system_info", parsed.SystemInfo).Run(context.Background())
		// if err != nil {
		// 	log.Printf("failed to update system info: %v", err)
		// }

		// // change the hash key to be that of the submission
		// err = s.ddbSubmTable.submTable.Update("eval_uuid", evalUuid).
		// 	Set("evaluation_stage", "received").
		// 	If("evaluation_stage = ?", "waiting").
		// 	Run(context.Background())
		// if err != nil {
		// 	var cce *types.ConditionalCheckFailedException
		// 	if errors.As(err, &cce) {
		// 		// its ok, the evaluation stage was already updated
		// 	} else {
		// 		log.Printf("failed to update evaluation stage: %v", err)
		// 	}
		// }

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
