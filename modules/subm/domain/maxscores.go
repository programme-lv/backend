package domain

import (
	"time"

	"github.com/google/uuid"
)

// max score the user has received on a subm for a specific task
type MaxScore struct {
	SubmUuid       uuid.UUID
	Received       int
	Possible       int
	ReceivedTenths int
	PossibleTenths int
	FirstTime      time.Time // first time the user got a score this high
}

type SubmJoinEvalOld struct {
	Subm Subm
	Eval Eval
}

type SubmJoinScoreInfo struct {
	SubmUuid    uuid.UUID
	TaskShortID string
	CreatedAt   time.Time
	ScoreInfo   ScoreInfo
}

// returns a map of task short ids to the max received score the user has received on a subm for that task
func CalcMaxScores(userSubms []SubmJoinScoreInfo) map[string]MaxScore {
	maxScores := make(map[string]MaxScore)

	for _, subm := range userSubms {
		taskId := subm.TaskShortID
		receivedTenths := subm.ScoreInfo.ReceivedScoreTenths
		if receivedTenths == 0 && subm.ScoreInfo.ReceivedScore > 0 {
			receivedTenths = subm.ScoreInfo.ReceivedScore * 10
		}
		possibleTenths := subm.ScoreInfo.PossibleScoreTenths
		if possibleTenths == 0 && subm.ScoreInfo.PossibleScore > 0 {
			possibleTenths = subm.ScoreInfo.PossibleScore * 10
		}
		thisScore := MaxScore{
			SubmUuid:       subm.SubmUuid,
			Received:       subm.ScoreInfo.ReceivedScore,
			Possible:       subm.ScoreInfo.PossibleScore,
			ReceivedTenths: receivedTenths,
			PossibleTenths: possibleTenths,
			FirstTime:      subm.CreatedAt,
		}

		if existingScore, exists := maxScores[taskId]; !exists {
			maxScores[taskId] = thisScore
		} else if thisScore.ReceivedTenths > existingScore.ReceivedTenths {
			maxScores[taskId] = thisScore
		} else if thisScore.ReceivedTenths == existingScore.ReceivedTenths && thisScore.FirstTime.Before(existingScore.FirstTime) {
			maxScores[taskId] = thisScore
		}
	}

	return maxScores
}
