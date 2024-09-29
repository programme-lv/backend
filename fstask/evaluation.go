package fstask

import (
	"fmt"
	"os"
	"path/filepath"
)

func (dir TaskDir) ReadTestlibChecker() (res string, err error) {
	requiredSpec := SemVer{major: 2, minor: 5}
	if dir.Spec.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		err = fmt.Errorf(format, dir.Spec.String(), requiredSpec.String())
		return
	}

	path := filepath.Join(dir.Path, "evaluation", "checker.cpp")
	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = nil
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	res = string(content)

	return
}

func (dir TaskDir) ReadTestlibInteractor() (res string, err error) {
	requiredSpec := SemVer{major: 2, minor: 5}
	if dir.Spec.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		err = fmt.Errorf(format, dir.Spec.String(), requiredSpec.String())
		return
	}

	path := filepath.Join(dir.Path, "evaluation", "interactor.cpp")
	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = nil
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	res = string(content)

	return
}

func (task *Task) LoadEvaluationCheckerAndInteractorFromDir(dir TaskDir) error {
	checker, err := dir.ReadTestlibChecker()
	if err != nil {
		return fmt.Errorf("failed to read testlib checker: %w", err)
	}
	task.TestlibChecker = checker

	interactor, err := dir.ReadTestlibInteractor()
	if err != nil {
		return fmt.Errorf("failed to read testlib interactor: %w", err)
	}
	task.TestlibInteractor = interactor
	return nil
}
