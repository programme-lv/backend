package srvc

// this file contains data-transfer objects for retrieving
// data from external sources such as the postgres database that we use
//
// the purpose of dtos is shifting data in expensive remote calls

import (
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/modules/subm/domain"
)

// shallow submission evaluation data-transfer object.
// excludes snapshotted subtasks, test groups and tests
// as those are 1:N relations that are a bit expensive to fetch
//
// includes, however, precalculated score info
// this object is necessary for listing tasks the user has solved
type ShallowEvalDto struct {
	UUID      uuid.UUID
	SubmUUID  uuid.UUID
	Stage     domain.EvalStage
	ScoreUnit domain.ScoreUnit

	Error *domain.EvalError

	// what was used to execute the tests
	Checker    *string
	Interactor *string
	CpuLimMs   int
	MemLimKiB  int

	ScoreInfo *domain.ScoreInfo // precalculated score info, if present
	// if score info is not present we will fetch the full domain.Eval object

	CreatedAt time.Time
}

// shallow submission (user code for solving task) data-transfer object.
// excludes the code content itself as that may be up to 100KB in size
// and there may be many submissions such as for a single user
type ShallowSubmDto struct {
	UUID         uuid.UUID
	AuthorUUID   uuid.UUID
	TaskShortID  string
	LangShortID  string
	CurrEvalUUID uuid.UUID
	CreatedAt    time.Time
}

// shallow submission dto joined with shallow evaluation dto
// this object is necessary for listing tasks the user has solved
type ShallowSubmJoinEvalDto struct {
	Subm ShallowSubmDto
	Eval ShallowEvalDto
}
