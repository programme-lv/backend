package submsrvc

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submsrvc/submadapter"
	"github.com/programme-lv/backend/subm/submsrvc/submcmd"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
)

type SubmSrvc struct {
	SubmitSol  submcmd.SubmitSolCmd
	ReEvalSubm submcmd.ReEvalSubmCmd

	GetSubm   submquery.GetSubmQuery
	ListSubms submquery.ListSubmsQuery
	GetEval   submquery.GetEvalQuery

	SubNewSubm submquery.SubNewSubms
	SubEvalUpd submquery.SubEvalUpd
}

func NewSubmSrvc(
	userSrvc submadapter.UserSrvcFacade,
	taskSrvc submadapter.TaskSrvcFacade,
	execSrvc submadapter.ExecSrvcFacade,
	submRepo submadapter.SubmRepo,
	evalRepo submadapter.EvalRepo,
) (*SubmSrvc, error) {
	newSubmChListenerLock := sync.Mutex{}
	newSubmListeners := make(map[chan<- subm.Subm]struct{})

	submitSolCmd := submcmd.SubmitSolCmdHandler{
		DoesUserExist: func(ctx context.Context, uuid uuid.UUID) (bool, error) {
			user, err := userSrvc.GetUserByUUID(ctx, uuid)
			if err != nil {
				return false, err
			}
			return user.UUID == uuid, nil
		},
		GetTask:   taskSrvc.GetTask,
		StoreSubm: submRepo.StoreSubm,
		StoreEval: evalRepo.StoreEval,
		BcastSubmCreated: func(subm subm.Subm) {
			slog.Info("submitted solution", "subm", subm)
			newSubmChListenerLock.Lock()
			for ch := range newSubmListeners {
				ch <- subm
			}
			newSubmChListenerLock.Unlock()
		},
	}

	getSubmQuery := submquery.NewGetSubmQuery(submRepo.GetSubm)
	listSubmsQuery := submquery.NewListSubmsQuery(submRepo.ListSubms)
	getEvalQuery := submquery.NewGetEvalQuery(evalRepo.GetEval)

	subNewSubmsQuery := submquery.NewSubNewSubmsQuery(func(ctx context.Context) (<-chan subm.Subm, error) {
		ch := make(chan subm.Subm)
		newSubmChListenerLock.Lock()
		newSubmListeners[ch] = struct{}{}
		newSubmChListenerLock.Unlock()
		go func() {
			<-ctx.Done()
			newSubmChListenerLock.Lock()
			delete(newSubmListeners, ch)
			newSubmChListenerLock.Unlock()
			close(ch)
		}()
		return ch, nil

	})

	return &SubmSrvc{
		SubmitSol:  submitSolCmd,
		GetSubm:    getSubmQuery,
		ListSubms:  listSubmsQuery,
		GetEval:    getEvalQuery,
		SubNewSubm: subNewSubmsQuery,
	}, nil
}
