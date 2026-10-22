package srvc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/programme-lv/backend/modules/exec"
	"github.com/programme-lv/backend/modules/subm/domain"
)

func TestApplyTestResultInteractivePartial(t *testing.T) {
	interactor := "interactor.cpp"
	eval := domain.Eval{
		Interactor: &interactor,
		CpuLimMs:   1000,
		MemLimKiB:  262144,
	}
	test := domain.Test{}

	applyTestResult(&test, eval,
		&exec.RunData{ExitCode: 0, CpuMs: 10, MemKiB: 1024, StdErr: "debug output"},
		&exec.RunData{ExitCode: 90},
	)

	assert.Equal(t, 40, test.Score)
	assert.False(t, test.Ac)
	assert.False(t, test.Wa)
	assert.False(t, test.Re)
}

func TestApplyTestResultInteractiveZero(t *testing.T) {
	interactor := "interactor.cpp"
	eval := domain.Eval{
		Interactor: &interactor,
		CpuLimMs:   1000,
		MemLimKiB:  262144,
	}
	test := domain.Test{}

	applyTestResult(&test, eval,
		&exec.RunData{ExitCode: 0, CpuMs: 10, MemKiB: 1024},
		&exec.RunData{ExitCode: 50},
	)

	assert.Equal(t, 0, test.Score)
	assert.True(t, test.Wa)
}

func TestApplyTestResultSimpleGroupFailure(t *testing.T) {
	eval := domain.Eval{
		CpuLimMs:  1000,
		MemLimKiB: 262144,
	}
	test := domain.Test{}

	applyTestResult(&test, eval,
		&exec.RunData{ExitCode: 0, CpuMs: 10, MemKiB: 1024},
		&exec.RunData{ExitCode: 1},
	)

	assert.Equal(t, 0, test.Score)
	assert.True(t, test.Wa)
}

func TestInteractorPercent(t *testing.T) {
	assert.Equal(t, 0, interactorPercent(nil))
	assert.Equal(t, 100, interactorPercent(&exec.RunData{ExitCode: 0}))
	assert.Equal(t, 0, interactorPercent(&exec.RunData{ExitCode: 50}))
	assert.Equal(t, 40, interactorPercent(&exec.RunData{ExitCode: 90}))
	assert.Equal(t, 100, interactorPercent(&exec.RunData{ExitCode: 150}))
	assert.Equal(t, 0, interactorPercent(&exec.RunData{ExitCode: 1}))
	assert.Equal(t, 0, interactorPercent(&exec.RunData{ExitCode: 151}))
}
