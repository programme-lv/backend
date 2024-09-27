package fstask

type TaskDirInfo struct {
	Path string // absolute path to the task
	Spec SemVer // specification version in problem.toml
	Info []byte // problem.toml
}
