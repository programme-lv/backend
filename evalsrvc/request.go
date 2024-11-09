package evalsrvc

type Request struct {
	Code   string
	LangId string

	Tests []Test

	CpuMs  int
	MemKiB int

	Checker    *string
	Interactor *string
}

type Test struct {
	ID int

	InSha256  *string
	InUrl     *string
	InContent *string

	AnsSha256  *string
	AnsUrl     *string
	AnsContent *string
}
