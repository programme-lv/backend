package subm

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/guregu/dynamo/v2"
)

type SubmissionDetailsRow struct {
	Uuid    string `dynamo:"subm_uuid,hash"` // partition key
	SortKey string `dynamo:"sort_key,range"` // details#<padded_unix_timestamp>

	Content string `dynamo:"content"` // submission task solution code

	AuthorUuid string `dynamo:"author_uuid"`
	TaskUuid   string `dynamo:"task_uuid"`
	ProgLangId string `dynamo:"prog_lang_id"`

	BriefEvaluation *SubmDetailsRowEvaluation `dynamo:"evaluation"`

	CreatedAtRfc3339 string `dynamo:"created_at_rfc3339"`
	Version          int64  `dynamo:"version"` // For optimistic locking
}

type SubmDetailsRowEvaluation struct {
	EvalUuid string       `dynamo:"eval_uuid"`
	Status   string       `dynamo:"status"`
	Scores   []ScoreGroup `dynamo:"scores"`
}

// ScoreGroup is either subtask or test group
// used to display visually the score partition of a submission in realtime
type ScoreGroup struct {
	Received int  `dynamo:"received"`
	Possible int  `dynamo:"possible"`
	Finished bool `dynamo:"finished"`
}

type SubmissionEvaluationDetailsRow struct {
	SubmUuid string `dynamo:"subm_uuid,hash"` // partition key
	SortKey  string `dynamo:"sort_key,range"` // evaluation#<eval_uuid>#details

	EvaluationStage string `dynamo:"evaluation_stage"` // "waiting", "received", "compiling", "testing", "finished"

	TestlibCheckerCode string `dynamo:"testlib_checker_code"` // the code of the testlib checker

	SystemInformation *string      `dynamo:"system_information"` // details about the system that ran the evaluation
	SubmCompileData   *RuntimeData `dynamo:"subm_compile_data"`  // compilation runtime data for author's submission

	CreatedAtRfc3339 string `dynamo:"created_at_rfc3339"`
	Version          int64  `dynamo:"version"` // For optimistic locking
}

func (s *SubmissionEvaluationDetailsRow) EvaluationUuid() string {
	// TODO: read second part of the sort key after splitting by "#"
	panic("not implemented")
}

type RuntimeData struct {
	Stdout   *string `dynamo:"stdout"` // might be trimmed
	Stderr   *string `dynamo:"stderr"` // might be trimmed
	ExitCode int64   `dynamo:"exit_code"`

	CpuTimeMillis   int64 `dynamo:"cpu_time_millis"`
	WallTimeMillis  int64 `dynamo:"wall_time_millis"`
	MemoryKibiBytes int64 `dynamo:"memory_kibi_bytes"`
}

type SubmissionEvaluationTestRow struct {
	SubmUuid string `dynamo:"subm_uuid,hash"` // partition key
	SortKey  string `dynamo:"sort_key,range"` // evaluation#<eval_uuid>#test#<padded_test_id>

	FullInputS3Uri  string `dynamo:"full_input_s3_uri"`
	FullAnswerS3Uri string `dynamo:"full_answer_s3_uri"`

	Ignored  bool `dynamo:"ignored"`
	Started  bool `dynamo:"started"`
	Finished bool `dynamo:"finished"`

	SubmTestRuntimeData *RuntimeData `dynamo:"subm_test_runtime_data"`
	CheckerRuntimeData  *RuntimeData `dynamo:"checker_runtime_data"`

	Subtasks  []int `dynamo:"subtasks"`   // subtasks that the test is part of
	TestGroup int   `dynamo:"test_group"` // test group that the test is part of
}

func (s *SubmissionEvaluationTestRow) TestId() int {
	// TODO: read second part of the sort key after splitting by "#"
	panic("not implemented")
}

type DynamoDbSubmTableV2 struct {
	ddbClient *dynamodb.Client
	tableName string
	submTable *dynamo.Table
}

func NewDynamoDbSubmTableV2(ddbClient *dynamodb.Client, tableName string) *DynamoDbSubmTableV2 {
	ddb := &DynamoDbSubmTableV2{
		ddbClient: ddbClient,
		tableName: tableName,
	}
	db := dynamo.NewFromIface(ddb.ddbClient)
	table := db.Table(ddb.tableName)
	ddb.submTable = &table

	return ddb
}

func (ddb *DynamoDbSubmTableV2) SaveSubmissionDetails(ctx context.Context, subm *SubmissionDetailsRow) error {
	// Increment the version number for optimistic locking
	subm.Version++

	put := ddb.submTable.Put(subm).If("attribute_not_exists(version) OR version = ?", subm.Version-1)
	return put.Run(ctx)
}

func (ddb *DynamoDbSubmTableV2) SaveSubmissionEvaluationDetails(ctx context.Context, eval *SubmissionEvaluationDetailsRow) error {
	// Increment the version number for optimistic locking
	eval.Version++

	put := ddb.submTable.Put(eval).If("attribute_not_exists(version) OR version = ?", eval.Version-1)
	return put.Run(ctx)
}
