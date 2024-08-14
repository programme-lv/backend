package subm

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/guregu/dynamo/v2"
	"goa.design/clue/log"
)

type SubmissionDetailsRow struct {
	SubmUuid string `dynamo:"subm_uuid,hash"` // partition key
	SortKey  string `dynamo:"sort_key,range"` // submission

	Content string `dynamo:"subm_content"` // submission task solution code

	AuthorUuid string `dynamo:"author_uuid"`
	TaskUuid   string `dynamo:"task_uuid"`
	ProgLangId string `dynamo:"prog_lang_id"`

	EvalResult *SubmDetailsRowEvaluation `dynamo:"evaluation_result"`

	CreatedAtRfc3339 string `dynamo:"created_at_rfc3339_utc"`
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
	SortKey  string `dynamo:"sort_key,range"` // evaluation#<eval_uuid>

	EvaluationStage string `dynamo:"evaluation_stage"` // "waiting", "received", "compiling", "testing", "finished"

	TestlibCheckerCode string `dynamo:"testlib_checker_code"` // the code of the testlib checker

	SystemInformation *string      `dynamo:"system_information"` // details about the system that ran the evaluation
	SubmCompileData   *RuntimeData `dynamo:"subm_compile_data"`  // compilation runtime data for author's submission

	ProgrammingLang SubmEvalDetailsProgrammingLang `dynamo:"programming_lang"`

	CreatedAtRfc3339 string `dynamo:"created_at_rfc3339_utc"`
	Version          int64  `dynamo:"version"` // For optimistic locking
}

type SubmEvalDetailsProgrammingLang struct {
	PLangId        string  `dynamo:"p_lang_id"`
	DisplayName    string  `dynamo:"display_name"`
	SubmCodeFname  string  `dynamo:"subm_code_fname"`
	CompileCommand *string `dynamo:"compile_command"`
	CompiledFname  *string `dynamo:"compiled_fname"`
	ExecCommand    string  `dynamo:"exec_command"`
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

	Reached  bool `dynamo:"reached"`
	Ignored  bool `dynamo:"ignored"`  // if doesn't count towards the score
	Finished bool `dynamo:"finished"` // if the test is evaluated / tested

	InputTrimmed  *string `dynamo:"input_trimmed"`  // trimmed input for display
	AnswerTrimmed *string `dynamo:"answer_trimmed"` // trimmed answer for display

	SubmTestRuntimeData *RuntimeData `dynamo:"subm_test_runtime_data"`
	CheckerRuntimeData  *RuntimeData `dynamo:"checker_runtime_data"`

	Subtasks  []int `dynamo:"subtasks"`   // subtasks that the test is part of
	TestGroup *int  `dynamo:"test_group"` // test group that the test is part of
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
	err := put.Run(ctx)
	return err
}

func (ddb *DynamoDbSubmTableV2) SaveSubmissionEvaluationDetails(ctx context.Context, eval *SubmissionEvaluationDetailsRow) error {
	// Increment the version number for optimistic locking
	eval.Version++

	put := ddb.submTable.Put(eval).If("attribute_not_exists(version) OR version = ?", eval.Version-1)
	return put.Run(ctx)
}

func (ddb *DynamoDbSubmTableV2) BatchSaveEvaluationTestRows(ctx context.Context, tests []*SubmissionEvaluationTestRow) error {
	for i := range (len(tests) + 24) / 25 {
		batch := make([]interface{}, 0)
		for j := range 25 {
			if i*25+j >= len(tests) {
				break
			}
			batch = append(batch, *tests[i*25+j])
		}
		_, err := ddb.submTable.Batch().Write().Put(batch...).Run(ctx)
		if err != nil {
			// check for The level of configured provisioned throughput for the table was exceeded. Consider increasing your provisioning level with the UpdateTable API.
			//types.ProvisionedThroughputExceededException
			var pte *types.ProvisionedThroughputExceededException
			if errors.As(err, &pte) {
				// backoff and retry
				log.Printf(ctx, "ProvisionedThroughputExceededException: %v", err)
				time.Sleep(1 * time.Second)
				i -= 1
				continue
			}

			return err
		}
	}
	return nil
}
