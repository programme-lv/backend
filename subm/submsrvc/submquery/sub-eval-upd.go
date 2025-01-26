package submquery

import (
	decorator "github.com/programme-lv/backend/srvccqs"
	"github.com/programme-lv/backend/subm"
)

type SubEvalUpd decorator.QueryHandler[struct{}, <-chan subm.Eval]
