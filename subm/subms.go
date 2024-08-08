package subm

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
	submgen "github.com/programme-lv/backend/gen/submissions"
	"goa.design/clue/log"
)

// submissions service example implementation.
// The example methods log the requests and return zero values.
type submissionssrvc struct {
	ddbSubmTable *DynamoDbSubmTable
}

// NewSubmissions returns the submissions service implementation.
func NewSubmissions() submgen.Service {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	return &submissionssrvc{
		ddbSubmTable: NewDynamoDbSubmTable(dynamodbClient, "proglv_submissions"),
	}
}

type Validatable interface {
	IsValid() error
}

type SubmissionContent struct {
	Value string
}

func (subm *SubmissionContent) IsValid() error {
	const maxSubmissionLength = 128000 // 128 KB
	if len(subm.Value) > maxSubmissionLength {
		return submgen.InvalidSubmissionDetails(
			"maksimālais iesūtījuma garums ir 128 KB",
		)
	}
	return nil
}

func (subm *SubmissionContent) String() string {
	return subm.Value
}

// CreateSubmission implements submissions.Service.
func (s *submissionssrvc) CreateSubmission(ctx context.Context, p *submgen.CreateSubmissionPayload) (res *submgen.Submission, err error) {
	submContent := SubmissionContent{Value: p.Submission}

	for _, v := range []Validatable{&submContent} {
		err := v.IsValid()
		if err != nil {
			return nil, err
		}
	}

	// TODO: retrieved at

	uuid := uuid.New()
	createdAt := time.Now()
	row := &SubmissionRow{
		Uuid:     uuid.String(),
		UnixTime: createdAt.Unix(),
		Content:  submContent.String(),
		Version:  0,
	}

	err = s.ddbSubmTable.Save(ctx, row)
	if err != nil {
		// TODO: automatically retry with exponential backoff on version conflict
		if dynamo.IsCondCheckFailed(err) {
			log.Errorf(ctx, err, "version conflict saving user")
			return nil, submgen.InternalError("version conflict saving user")
		} else {
			log.Errorf(ctx, err, "error saving user")
			return nil, submgen.InternalError("error saving user")
		}
	}

	createdAtRfc3339 := createdAt.Format(time.RFC3339)

	res = &submgen.Submission{
		UUID:       row.Uuid,
		Submission: row.Content,
		Username:   "",
		CreatedAt:  createdAtRfc3339,
		Evaluation: nil,
		Language:   nil,
		Task:       nil,
	}

	return res, nil
}

// List all submissions
func (s *submissionssrvc) ListSubmissions(ctx context.Context) (res []*submgen.Submission, err error) {
	log.Printf(ctx, "submissions.listSubmissions")
	return
}

// Get a submission by UUID
func (s *submissionssrvc) GetSubmission(ctx context.Context, p *submgen.GetSubmissionPayload) (res *submgen.Submission, err error) {
	res = &submgen.Submission{}
	log.Printf(ctx, "submissions.getSubmission")
	return
}
