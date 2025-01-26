package submsrvc_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
)

// component tests
// i want to test whether the solution submission works correctly

// get submission maybe returns error?
// submit solution
// await the submission created event
// get submission

type userSrvcMock struct {
	getUserByUUID func(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error)
}

func (u userSrvcMock) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error) {
	return u.getUserByUUID(ctx, uuid)
}

type taskSrvcMock struct {
	getTask func(ctx context.Context, shortId string) (tasksrvc.Task, error)
}

func (t taskSrvcMock) GetTask(ctx context.Context, shortId string) (tasksrvc.Task, error) {
	return t.getTask(ctx, shortId)
}

type execSrvcMock struct {
	enqueue func(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []execsrvc.TestFile, params execsrvc.TesterParams) (uuid.UUID, error)
	listen  func(ctx context.Context, evalUuid uuid.UUID) (<-chan execsrvc.Event, error)
}

func TestSubmitSolution(t *testing.T) {
	// test plan:
	// 1. initialize service
	// 2. attempt to get submission, expect error
	// 3. submit solution in c++
	// 4. await submission created event
	// 5. get submission, expect no error
	// 6. cmp submission to expected

	userSrvcMock := userSrvcMock{
		getUserByUUID: func(ctx context.Context, uuid uuid.UUID) (usersrvc.User, error) {
			return usersrvc.User{UUID: uuid}, nil
		},
	}

	taskSrvcMock := taskSrvcMock{
		getTask: func(ctx context.Context, shortId string) (tasksrvc.Task, error) {
			return tasksrvc.Task{}, nil
		},
	}

	srvc, err := submsrvc.NewSubmSrvc(userSrvcMock, taskSrvcMock, execSrvcMock)
}
