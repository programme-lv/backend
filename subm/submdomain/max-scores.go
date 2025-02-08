package submdomain

import "github.com/google/uuid"

// max score the user has received on a subm for a specific task
type MaxScore struct {
	SubmUuid uuid.UUID
	Received int
	Possible int
}

type SubmJoinEval struct {
	Subm Subm
	Eval Eval
}

// returns a map of task short ids to the max received score the user has received on a subm for that task
func CalcMaxScores(userSubms []SubmJoinEval) map[string]MaxScore {
	maxScores := make(map[string]MaxScore)

	for _, subm := range userSubms {
		taskId := subm.Subm.TaskShortID
		scoreInfo := subm.Eval.CalculateScore()
		currentScore := MaxScore{
			SubmUuid: subm.Subm.UUID,
			Received: scoreInfo.ReceivedScore,
			Possible: scoreInfo.PossibleScore,
		}

		if existingScore, exists := maxScores[taskId]; !exists || currentScore.Received > existingScore.Received {
			maxScores[taskId] = currentScore
		}
	}

	return maxScores
}
