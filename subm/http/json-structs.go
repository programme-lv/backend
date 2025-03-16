package http

type DetailedSubmView struct {
	SubmUUID  string `json:"subm_uuid"`
	Content   string `json:"content,omitempty"`
	Username  string `json:"username"`
	CurrEval  *Eval  `json:"curr_eval"`
	PrLang    PrLang `json:"pr_lang"`
	TaskID    string `json:"task_id"`
	TaskName  string `json:"task_name"`
	CreatedAt string `json:"created_at"`
}

type PrLang struct {
	ShortID  string `json:"short_id"`
	Display  string `json:"display"`
	MonacoID string `json:"monaco_id"`
}

type SubmListEntry struct {
	SubmUuid   string    `json:"subm_uuid"`
	Username   string    `json:"username"`
	TaskId     string    `json:"task_id"`
	TaskName   string    `json:"task_name"`
	PrLangId   string    `json:"pr_lang_id"`
	PrLangName string    `json:"pr_lang_name"`
	ScoreInfo  ScoreInfo `json:"score_info"`
	Status     string    `json:"status"`
	CreatedAt  string    `json:"created_at"`
}

type Eval struct {
	EvalUUID  string `json:"eval_uuid"`
	SubmUUID  string `json:"subm_uuid"`
	EvalStage string `json:"eval_stage"`
	ScoreUnit string `json:"score_unit"`
	EvalError string `json:"eval_error"`
	// ErrorMsg   string      `json:"error_msg"`
	Subtasks   []Subtask   `json:"subtasks"`
	TestGroups []TestGroup `json:"test_groups"`
	Verdicts   string      `json:"verdicts"` // q,ac,wa,tle,mle,re,ig -> "QAWTMRI"
	ScoreInfo  ScoreInfo   `json:"score_info"`
}

type ScoreInfo struct {
	ScoreBar struct {
		Green  int `json:"green"`
		Red    int `json:"red"`
		Gray   int `json:"gray"`
		Yellow int `json:"yellow"`
		Purple int `json:"purple"`
	} `json:"score_bar"`
	ReceivedScore int `json:"received"`
	PossibleScore int `json:"possible"`
	MaxCpuMs      int `json:"max_cpu_ms"`  // milliseconds
	MaxMemKiB     int `json:"max_mem_kib"` // kibibytes
}

type Subtask struct {
	Points      int    `json:"points"`
	Description string `json:"description"`
	// StTests     []int  `json:"st_tests"`
	StTests [][]int `json:"st_tests"`
}

type TestGroup struct {
	Points   int   `json:"points"`
	Subtasks []int `json:"subtasks"`
	// TgTests  []int `json:"tg_tests"`
	TgTests [][]int `json:"tg_tests"`
}

type MaxScore struct {
	SubmUuid     string `json:"subm_uuid"`
	Received     int    `json:"received"`
	Possible     int    `json:"possible"`
	CreatedAt    string `json:"created_at"`
	TaskFullName string `json:"task_full_name"`
}
