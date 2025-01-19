package submhttp

import (
	"github.com/programme-lv/backend/subm/submsrvc"
)

type SubmHttpServer struct {
	submSrvc *submsrvc.SubmSrvc
}

func NewSubmHttpServer(submSrvc *submsrvc.SubmSrvc) *SubmHttpServer {
	return &SubmHttpServer{submSrvc: submSrvc}
}
