package srvc_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/gen/mocks/mocktasksrvc"
	"github.com/programme-lv/backend/modules/task/srvc"
	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPatchTaskFromZipOmitsTests(t *testing.T) {
	ctx := context.Background()
	repo := mocktasksrvc.NewMockTaskPgRepo(t)
	publicStore, _ := filestore.NewStore(t.TempDir())
	testStore, _ := filestore.NewStore(t.TempDir())
	service := srvc.NewTaskSrvc(repo, publicStore, testStore)

	data, err := os.ReadFile("testdata/lio2026cuska.zip")
	require.NoError(t, err)
	archive, err := taskzipv1.Read(data)
	require.NoError(t, err)
	existing, err := taskFromArchive(t, archive)
	require.NoError(t, err)
	existing.MdImages = nil

	repo.EXPECT().Exists(ctx, "lio2026cuska").Return(true, nil).Once()
	repo.EXPECT().GetTask(ctx, "lio2026cuska").Return(existing, nil).Once()
	manifest, manifestErr := service.GetTaskManifest(ctx, "lio2026cuska")
	require.NoError(t, manifestErr)

	archive.Statements["lv"] = []byte("# Patched story\n")
	full, err := taskzipv1.Write(archive)
	require.NoError(t, err)
	sparse := zipWithoutPrefixes(t, full, "tests/", "solutions/")

	var updated srvc.Task
	repo.EXPECT().Exists(ctx, "lio2026cuska").Return(true, nil).Once()
	repo.EXPECT().GetTask(ctx, "lio2026cuska").Return(existing, nil).Once()
	repo.EXPECT().UpdateTask(ctx, mock.Anything, false, mock.Anything).RunAndReturn(
		func(_ context.Context, task srvc.Task, replace bool, _ int) error {
			updated = task
			return nil
		},
	).Once()

	version := manifest.TestsVersion
	revision := manifest.Revision
	result, patchErr := service.PatchTaskFromZip(ctx, "lio2026cuska", sparse, srvc.PatchMeta{
		BaseContentETag:  manifest.ContentETag,
		BaseTestsVersion: &version,
		BaseRevision:     &revision,
		Sections:         []string{"statement", "examples", "readme"},
	})
	require.NoError(t, patchErr)
	require.False(t, result.TestsChanged)
	require.Equal(t, manifest.TestsVersion, result.TestsVersion)
	require.Contains(t, updated.MdStatements[0].Story, "Patched story")
	require.Equal(t, existing.Tests, updated.Tests)
	require.Equal(t, existing.Solutions, updated.Solutions)
}

func TestPatchTaskFromZipRejectsStaleBase(t *testing.T) {
	ctx := context.Background()
	repo := mocktasksrvc.NewMockTaskPgRepo(t)
	publicStore, _ := filestore.NewStore(t.TempDir())
	testStore, _ := filestore.NewStore(t.TempDir())
	service := srvc.NewTaskSrvc(repo, publicStore, testStore)

	data, err := os.ReadFile("testdata/lio2026cuska.zip")
	require.NoError(t, err)
	archive, err := taskzipv1.Read(data)
	require.NoError(t, err)
	existing, err := taskFromArchive(t, archive)
	require.NoError(t, err)

	repo.EXPECT().Exists(ctx, "lio2026cuska").Return(true, nil).Once()
	repo.EXPECT().GetTask(ctx, "lio2026cuska").Return(existing, nil).Once()

	patch, err := taskzipv1.Write(archive)
	require.NoError(t, err)
	revision := existing.Revision
	_, patchErr := service.PatchTaskFromZip(ctx, "lio2026cuska", patch, srvc.PatchMeta{
		BaseContentETag: "stale", BaseRevision: &revision, Sections: []string{"statement"},
	})
	require.ErrorIs(t, patchErr, srvc.ErrTaskConflict)
}

func TestPatchTaskFromZipBumpsTestsVersion(t *testing.T) {
	ctx := context.Background()
	repo := mocktasksrvc.NewMockTaskPgRepo(t)
	publicStore, _ := filestore.NewStore(t.TempDir())
	testStore, _ := filestore.NewStore(t.TempDir())
	service := srvc.NewTaskSrvc(repo, publicStore, testStore)

	data, err := os.ReadFile("testdata/lio2026cuska.zip")
	require.NoError(t, err)
	archive, err := taskzipv1.Read(data)
	require.NoError(t, err)
	existing, err := taskFromArchive(t, archive)
	require.NoError(t, err)
	archive.Testing.CPUMs++
	patch, err := taskzipv1.Write(archive)
	require.NoError(t, err)

	repo.EXPECT().Exists(ctx, "lio2026cuska").Return(true, nil).Once()
	repo.EXPECT().GetTask(ctx, "lio2026cuska").Return(existing, nil).Once()
	revision := existing.Revision
	_, patchErr := service.PatchTaskFromZip(ctx, "lio2026cuska", patch, srvc.PatchMeta{
		BaseRevision: &revision, Sections: []string{"statement", "tests"},
	})
	require.ErrorIs(t, patchErr, srvc.ErrTaskTestsChanged)

	repo.EXPECT().Exists(ctx, "lio2026cuska").Return(true, nil).Once()
	repo.EXPECT().GetTask(ctx, "lio2026cuska").Return(existing, nil).Once()
	repo.EXPECT().UpdateTask(ctx, mock.Anything, true, mock.Anything).Return(nil).Once()
	result, patchErr := service.PatchTaskFromZip(ctx, "lio2026cuska", patch, srvc.PatchMeta{
		BaseRevision: &revision, Sections: []string{"statement", "tests"}, AllowTests: true,
	})
	require.NoError(t, patchErr)
	require.True(t, result.TestsChanged)
	require.Equal(t, existing.TestsVersion+1, result.TestsVersion)
}

func zipWithoutPrefixes(t *testing.T, data []byte, prefixes ...string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, file := range reader.File {
		skip := false
		for _, prefix := range prefixes {
			skip = skip || strings.HasPrefix(file.Name, prefix)
		}
		if skip {
			continue
		}
		header := file.FileHeader
		writer, err := zw.CreateHeader(&header)
		require.NoError(t, err)
		rc, err := file.Open()
		require.NoError(t, err)
		_, err = io.Copy(writer, rc)
		rc.Close()
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}
