package fstask_test

import (
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingArchiveFiles(t *testing.T) {
	task, err := fstask.Read(kvadrputeklV2dot5Path)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	ensureArchiveFilesCorrespondToTestdata(t, task)

	writtenTask := writeAndReReadTask(t, task)

	ensureArchiveFilesCorrespondToTestdata(t, writtenTask)
}

func ensureArchiveFilesCorrespondToTestdata(t *testing.T, task *fstask.Task) {
	/*
		bacb6666eb89b56023f0f436beab2d2d5146578205f4cf35f1b39cd7a55b2510  riki/00_gen_params.py
		e69140058ad89ff99c4f0270d3c2570c220dbbca05885aecdb3b40ec54336896  riki/og_tests/kp.i00
		618b451f2de6fe7c969a7cf41ff27d934f5b117b609f7e41ce81219ae20affd4  task.yaml
		d54ecf2a172b059abff32412375e99dd7f5c9737b3210f9d45ac1d992d42334d  teksts/kp.typ
	*/

	expFiles := []string{
		"riki/00_gen_params.py",
		"riki/og_tests/kp.i00",
		"task.yaml",
		"teksts/kp.typ",
	}

	expChsums := []string{
		"bacb6666eb89b56023f0f436beab2d2d5146578205f4cf35f1b39cd7a55b2510",
		"e69140058ad89ff99c4f0270d3c2570c220dbbca05885aecdb3b40ec54336896",
		"618b451f2de6fe7c969a7cf41ff27d934f5b117b609f7e41ce81219ae20affd4",
		"d54ecf2a172b059abff32412375e99dd7f5c9737b3210f9d45ac1d992d42334d",
	}

	archiveFiles := task.ArchiveFiles
	require.Len(t, archiveFiles, 4)

	for i, archiveFile := range archiveFiles {
		require.Equal(t, expFiles[i], archiveFile.RelativePath)
		chsum := getSha256Sum(archiveFile.Content)
		require.Equal(t, expChsums[i], chsum)
	}
}
