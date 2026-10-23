package srvc_test

import (
	"context"
	"testing"

	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/gen/mocks/mocktasksrvc"
	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/stretchr/testify/require"
)

func editService(t *testing.T) (srvc.TaskService, *mocktasksrvc.MockTaskPgRepo) {
	t.Helper()
	repo := mocktasksrvc.NewMockTaskPgRepo(t)
	publicStore, err := filestore.NewStore(t.TempDir())
	require.NoError(t, err)
	testStore, err := filestore.NewStore(t.TempDir())
	require.NoError(t, err)
	return srvc.NewTaskSrvc(repo, publicStore, testStore), repo
}

func TestUpdateTaskLimitsValidation(t *testing.T) {
	service, _ := editService(t)
	_, err := service.UpdateTaskLimits(context.Background(), "task", 0, 512)
	require.Error(t, err)
	require.True(t, srvcerror.Is(err, "invalid_task_limits"))

	_, err = service.UpdateTaskLimits(context.Background(), "task", 4000, 2048)
	require.Error(t, err)
	require.True(t, srvcerror.Is(err, "invalid_task_limits"))
}

func TestUpdateTaskLimitsConvertsToSeconds(t *testing.T) {
	service, repo := editService(t)
	repo.On("UpdateTaskLimits", context.Background(), "task", 4.0, 640).Return(9, nil).Once()

	revision, err := service.UpdateTaskLimits(context.Background(), "task", 4000, 640)
	require.NoError(t, err)
	require.Equal(t, 9, revision)
}

func TestUpdateTaskSolutionValidatesSubtask(t *testing.T) {
	service, repo := editService(t)
	task := srvc.Task{
		ShortId: "task",
		Subtasks: []srvc.Subtask{
			{Score: 10},
			{Score: 20},
		},
		Solutions: []srvc.Solution{{Fname: "ok.cpp", Content: "int main(){}"}},
	}
	repo.EXPECT().Exists(context.Background(), "task").Return(true, nil).Once()
	repo.EXPECT().GetTask(context.Background(), "task").Return(task, nil).Once()

	_, err := service.UpdateTaskSolution(context.Background(), "task", "ok.cpp", "new", []int{3})
	require.Error(t, err)
	require.True(t, srvcerror.Is(err, "invalid_task_solution"))
}

func TestUpdateTaskSolutionDeduplicatesSubtasks(t *testing.T) {
	service, repo := editService(t)
	task := srvc.Task{
		ShortId: "task",
		Subtasks: []srvc.Subtask{
			{Score: 10},
			{Score: 20},
		},
		Solutions: []srvc.Solution{{Fname: "ok.cpp", Content: "int main(){}"}},
	}
	repo.EXPECT().Exists(context.Background(), "task").Return(true, nil).Once()
	repo.EXPECT().GetTask(context.Background(), "task").Return(task, nil).Once()
	repo.On("UpdateTaskSolution", context.Background(), "task", "ok.cpp", "new", []int{2, 1}).
		Return(4, nil).
		Once()

	revision, err := service.UpdateTaskSolution(
		context.Background(), "task", "ok.cpp", "new", []int{2, 1, 2},
	)
	require.NoError(t, err)
	require.Equal(t, 4, revision)
}

func TestGetTaskSolutionNotFound(t *testing.T) {
	service, repo := editService(t)
	task := srvc.Task{
		ShortId:   "task",
		Solutions: []srvc.Solution{{Fname: "ok.cpp", Content: "int main(){}"}},
	}
	repo.EXPECT().Exists(context.Background(), "task").Return(true, nil).Once()
	repo.EXPECT().GetTask(context.Background(), "task").Return(task, nil).Once()

	_, err := service.GetTaskSolution(context.Background(), "task", "missing.cpp")
	require.Error(t, err)
	require.True(t, srvcerror.Is(err, "task_solution_not_found"))
}
