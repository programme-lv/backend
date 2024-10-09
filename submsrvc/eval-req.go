package submsrvc

type EvalReqWithUuid struct {
	EvaluationUuid string      `json:"evaluation_uuid"`
	Request        EvalRequest `json:"request"`
	ResponseSqsUrl *string     `json:"response_sqs_url"`
}

type EvalRequest struct {
	Submission     string          `json:"submission"`
	Language       LanguageDetails `json:"language"`
	Limits         LimitsDetails   `json:"limits"`
	Tests          []Test          `json:"tests"`
	TestlibChecker string          `json:"testlib_checker"`
}

type LanguageDetails struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	CodeFilename     string  `json:"code_filename"`
	CompileCmd       *string `json:"compile_cmd"`
	CompiledFilename *string `json:"compiled_filename"`
	ExecCmd          string  `json:"exec_cmd"`
}

type LimitsDetails struct {
	CPUTimeMillis   int `json:"cpu_time_millis"`
	MemoryKibibytes int `json:"memory_kibibytes"`
}

type Test struct {
	ID            int     `json:"id"`
	InputSha256   string  `json:"input_sha256"`
	InputS3URI    string  `json:"input_s3_uri"`
	InputContent  *string `json:"input_content"`
	InputHttpURL  *string `json:"input_http_url"`
	AnswerSha256  string  `json:"answer_sha256"`
	AnswerS3URI   string  `json:"answer_s3_uri"`
	AnswerContent *string `json:"answer_content"`
	AnswerHttpURL *string `json:"answer_http_url"`
}

type Evaluation struct {
	EvaluationUUID string      `json:"evaluation_uuid"`
	Request        EvalRequest `json:"request"`
}
