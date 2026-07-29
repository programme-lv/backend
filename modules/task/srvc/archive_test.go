package srvc_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/gen/mocks/mocktasksrvc"
	"github.com/programme-lv/backend/modules/task/srvc"
	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestImportExportTaskZip(t *testing.T) {
	ctx := context.Background()
	repo := mocktasksrvc.NewMockTaskPgRepo(t)
	publicStore, err := filestore.NewStore(t.TempDir())
	require.NoError(t, err)
	testStore, err := filestore.NewStore(t.TempDir())
	require.NoError(t, err)
	service := srvc.NewTaskSrvc(repo, publicStore, testStore)

	data, err := os.ReadFile("testdata/lio2026cuska.zip")
	require.NoError(t, err)

	var imported srvc.Task
	repo.EXPECT().Exists(ctx, "lio2026cuska").Return(false, nil).Once()
	repo.EXPECT().CreateTask(ctx, mock.Anything).RunAndReturn(
		func(_ context.Context, task srvc.Task) error {
			imported = task
			return nil
		},
	).Once()

	id, importErr := service.ImportTaskFromZip(ctx, data, "")
	require.NoError(t, importErr)
	require.Equal(t, "lio2026cuska", id)
	require.Equal(t, "Purva čūska", imported.FullName["lv"])
	require.Equal(t, "2026", imported.OriginYear)
	require.Len(t, imported.Tests, 159)
	require.Len(t, imported.TestGroups, 53)
	require.Len(t, imported.Subtasks, 5)
	require.Len(t, imported.Solutions, 4)
	require.Len(t, imported.MdImages, 1)
	require.Contains(t, imported.MdStatements[0].Story, "veģetāra čūska")
	require.NotEmpty(t, imported.Checker)
	require.Empty(t, imported.OgFilesZipS3Key)

	repo.EXPECT().Exists(ctx, id).Return(true, nil).Once()
	repo.EXPECT().GetTask(ctx, id).Return(imported, nil).Once()
	exported, exportErr := service.ExportTaskAsZip(ctx, id)
	require.NoError(t, exportErr)
	assertIgnoredDirectoriesOmitted(t, exported)

	parsed, err := taskzipv1.Read(exported)
	require.NoError(t, err)
	require.Equal(t, id, parsed.ID)
	require.Len(t, parsed.Tests, 159)
	require.Len(t, parsed.TestGroups, 53)
	require.Len(t, parsed.Subtasks, 5)

	var reimported srvc.Task
	repo.EXPECT().Exists(ctx, "lio2026cuska-copy").Return(false, nil).Once()
	repo.EXPECT().CreateTask(ctx, mock.Anything).RunAndReturn(
		func(_ context.Context, task srvc.Task) error {
			reimported = task
			return nil
		},
	).Once()

	copyID, reimportErr := service.ImportTaskFromZip(ctx, exported, "lio2026cuska-copy")
	require.NoError(t, reimportErr)
	require.Equal(t, "lio2026cuska-copy", copyID)
	require.Len(t, reimported.Tests, len(imported.Tests))
	require.Len(t, reimported.TestGroups, len(imported.TestGroups))
	require.Len(t, reimported.Subtasks, len(imported.Subtasks))
	require.Equal(t, imported.FullName, reimported.FullName)
}

func assertIgnoredDirectoriesOmitted(t *testing.T, data []byte) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	for _, file := range reader.File {
		require.NotRegexp(t, `^(archive|testspec)/`, file.Name)
	}
}
