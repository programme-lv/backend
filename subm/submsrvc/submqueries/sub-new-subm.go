package submqueries

import (
	decorator "github.com/programme-lv/backend/srvccqs"
	"github.com/programme-lv/backend/subm"
)

type SubNewSubms decorator.QueryHandler[struct{}, <-chan subm.Subm]
