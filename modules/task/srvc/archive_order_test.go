package srvc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapToTaskZipSortsTestIDs(t *testing.T) {
	task := Task{
		ShortId:         "order-test",
		FullName:        map[string]string{"lv": "Order test"},
		CpuTimeLimSecs:  1,
		MemLimMegabytes: 256,
		TestGroups: []TestGroup{
			{Points: 1, TestIDs: []int{3, 2, 1}},
		},
		Subtasks: []Subtask{
			{Score: 1, TestIDs: []int{3, 1, 2}},
		},
	}

	archive, err := mapToTaskZip(task)
	require.NoError(t, err)
	require.Len(t, archive.Subtasks, 1)
	require.Equal(t, "01-01", archive.Subtasks[0].Groups)
}
