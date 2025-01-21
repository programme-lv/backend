package submhttp

import (
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
)

type SubmHttpServer struct {
	submSrvc *submsrvc.SubmSrvc
	taskSrvc *tasksrvc.TaskSrvc
	userSrvc *usersrvc.UserSrvc
}

func NewSubmHttpServer(
	submSrvc *submsrvc.SubmSrvc,
	taskSrvc *tasksrvc.TaskSrvc,
	userSrvc *usersrvc.UserSrvc,
) *SubmHttpServer {
	return &SubmHttpServer{
		submSrvc: submSrvc,
		taskSrvc: taskSrvc,
		userSrvc: userSrvc,
	}
}
