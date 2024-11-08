package evalsrvc

type Request struct {
	Code       string   `json:"code"`
	Language   Language `json:"language"`
	Tests      []Test   `json:"tests"`
	Checker    *string  `json:"checker"`
	Interactor *string  `json:"interactor"`

	CpuMillis int `json:"cpu_millis"`
	MemoryKiB int `json:"memory_kib"`
}

type Language struct {
	LangID        string  `json:"lang_id"`
	LangName      string  `json:"lang_name"`
	CodeFname     string  `json:"code_fname"`
	CompileCmd    *string `json:"compile_cmd"`
	CompiledFname *string `json:"compiled_fname"`
	ExecCmd       string  `json:"exec_cmd"`
}

type Test struct {
	ID int `json:"id"`

	InSha256  *string `json:"in_sha256"`
	InUrl     *string `json:"in_url"`
	InContent *string `json:"in_content"`

	AnsSha256  *string `json:"ans_sha256"`
	AnsUrl     *string `json:"ans_url"`
	AnsContent *string `json:"ans_content"`
}
