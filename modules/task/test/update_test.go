//go:build integration

package test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/programme-lv/backend/modules/task/srvc"
	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
	"github.com/stretchr/testify/require"
)

func TestUpdateTaskZipInPlace(t *testing.T) {
	ctx := context.Background()
	service := newTaskSrvc(t)
	data, err := os.ReadFile(filepath.Join("..", "srvc", "testdata", "lio2026cuska.zip"))
	require.NoError(t, err)
	id, importErr := service.ImportTaskFromZip(ctx, data, "")
	require.NoError(t, importErr)

	exported, exportErr := service.ExportTaskAsZip(ctx, id)
	require.NoError(t, exportErr)
	archive, err := taskzipv1.Read(exported)
	require.NoError(t, err)
	archive.Statements["lv"] = []byte("# In-place update\n")
	updated, err := taskzipv1.Write(archive)
	require.NoError(t, err)

	result, updateErr := service.UpdateTaskFromZip(ctx, id, updated, false)
	require.NoError(t, updateErr)
	require.False(t, result.TestsChanged)
	require.Equal(t, 1, result.TestsVersion)

	task, getErr := service.GetTask(ctx, id)
	require.NoError(t, getErr)
	require.Contains(t, task.MdStatements[0].Story, "In-place update")
	require.Len(t, task.Tests, 159)
}

func TestPatchTaskZipSkipsUnchangedTests(t *testing.T) {
	ctx := context.Background()
	service := newTaskSrvc(t)
	data, err := os.ReadFile(filepath.Join("..", "srvc", "testdata", "lio2026cuska.zip"))
	require.NoError(t, err)
	id, importErr := service.ImportTaskFromZip(ctx, data, "")
	require.NoError(t, importErr)
	manifest, manifestErr := service.GetTaskManifest(ctx, id)
	require.NoError(t, manifestErr)

	exported, exportErr := service.ExportTaskAsZip(ctx, id)
	require.NoError(t, exportErr)
	archive, err := taskzipv1.Read(exported)
	require.NoError(t, err)
	archive.Statements["lv"] = []byte("# Sparse patch\n")
	full, err := taskzipv1.Write(archive)
	require.NoError(t, err)
	sparse := stripEntries(t, full, "tests/", "solutions/")

	version := manifest.TestsVersion
	revision := manifest.Revision
	result, patchErr := service.PatchTaskFromZip(ctx, id, sparse, srvc.PatchMeta{
		BaseContentETag:  manifest.ContentETag,
		BaseTestsVersion: &version,
		BaseRevision:     &revision,
		Sections:         []string{"statement", "examples", "readme"},
	})
	require.NoError(t, patchErr)
	require.False(t, result.TestsChanged)
	require.Equal(t, manifest.TestsVersion, result.TestsVersion)

	task, getErr := service.GetTask(ctx, id)
	require.NoError(t, getErr)
	require.Contains(t, task.MdStatements[0].Story, "Sparse patch")
	require.Len(t, task.Tests, 159)
	require.Len(t, task.Solutions, 4)
}

func TestPatchTaskZipRejectsStaleRevision(t *testing.T) {
	ctx := context.Background()
	service := newTaskSrvc(t)
	data, err := os.ReadFile(filepath.Join("..", "srvc", "testdata", "lio2026cuska.zip"))
	require.NoError(t, err)
	id, importErr := service.ImportTaskFromZip(ctx, data, "")
	require.NoError(t, importErr)
	task, getErr := service.GetTask(ctx, id)
	require.NoError(t, getErr)
	baseRevision := task.Revision

	updateErr := service.UpdateStatementMd(ctx, id, srvc.MarkdownStatement{
		LangIso639: "lv", Story: "concurrent edit",
	})
	require.NoError(t, updateErr)

	archive, err := taskzipv1.Read(data)
	require.NoError(t, err)
	archive.Statements["lv"] = []byte("# stale patch\n")
	patch, err := taskzipv1.Write(archive)
	require.NoError(t, err)
	_, patchErr := service.PatchTaskFromZip(ctx, id, patch, srvc.PatchMeta{
		BaseRevision: &baseRevision,
		Sections:     []string{"statement"},
	})
	require.ErrorIs(t, patchErr, srvc.ErrTaskConflict)

	after, getErr := service.GetTask(ctx, id)
	require.NoError(t, getErr)
	require.NotContains(t, after.MdStatements[0].Story, "stale patch")
}

func stripEntries(t *testing.T, data []byte, prefixes ...string) []byte {
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

func TestUpdateTaskZipGuardsTests(t *testing.T) {
	ctx := context.Background()
	service := newTaskSrvc(t)
	data, err := os.ReadFile(filepath.Join("..", "srvc", "testdata", "lio2026cuska.zip"))
	require.NoError(t, err)
	id, importErr := service.ImportTaskFromZip(ctx, data, "")
	require.NoError(t, importErr)

	archive, err := taskzipv1.Read(data)
	require.NoError(t, err)
	archive.Testing.CPUMs++
	updated, err := taskzipv1.Write(archive)
	require.NoError(t, err)

	_, updateErr := service.UpdateTaskFromZip(ctx, id, updated, false)
	require.ErrorIs(t, updateErr, srvc.ErrTaskTestsChanged)

	result, updateErr := service.UpdateTaskFromZip(ctx, id, updated, true)
	require.NoError(t, updateErr)
	require.True(t, result.TestsChanged)
	require.Equal(t, 2, result.TestsVersion)
}
