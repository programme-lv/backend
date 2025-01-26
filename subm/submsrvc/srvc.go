package submsrvc

import (
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/subm/submsrvc/submcmds"
	"github.com/programme-lv/backend/subm/submsrvc/submqueries"
)

type SubmSrvc struct {
	CreateSubm  submcmds.CreateSubmCmd
	CreateEval  submcmds.CreateEvalCmd
	AttachEval  submcmds.AttachEvalCmd
	EnqueueEval submcmds.EnqueueEvalCmd
	ReEvalSubm  submcmds.ReEvalSubmCmd

	GetSubm    submqueries.GetSubmQuery
	ListSubms  submqueries.ListSubmsQuery
	GetEval    submqueries.GetEvalQuery
	SubNewSubm submqueries.SubNewSubms
	SubEvalUpd submqueries.SubEvalUpd
}
