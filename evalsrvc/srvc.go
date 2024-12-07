/*
Evaluation service:
asd
1. picks **UUID** v7, stores an empty evaluation in-memory

2. **enqueues** evaluation request into SQS *submission queue*

3. **receives** events from the *tester* via SQS *response queue*

4. **constructs** the full evaluation from events in-memory
  - test full stdout / stderr are stored immediately to S3

5. sends each evaluation event to listener at most once
  - all evaluation related events are deleted after 5 minutes
*/
package evalsrvc

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/emirpasic/gods/v2/queues"
	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"
)

type EvalRepo interface {
	Save(eval *Evaluation) error
	Get(evalUuid uuid.UUID) (*Evaluation, error)
}

type Pair[T1 any, T2 any] struct {
	First  T1
	Second T2
}

type EvalSrvc struct {
	sqsClient *sqs.Client
	evals     []Evaluation // later these will be only the ones being tested with a timeout
	evalsLock sync.Mutex
	repo      EvalRepo

	submSqsUrl     string
	responseSqsUrl string
	extEvalSqsUrl  string

	extEvalKey string // api key for external evaluation requests

	accumulated *xsync.MapOf[uuid.UUID, Pair[*sync.Cond, queues.Queue[Pair[Msg, time.Time]]]]
}

func NewEvalSrvc() *EvalSrvc {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	sqsClient := sqs.NewFromConfig(cfg)

	submQueueUrl := os.Getenv("SUBM_SQS_QUEUE_URL")
	if submQueueUrl == "" {
		panic("SUBM_SQS_QUEUE_URL not set in .env file")
	}

	responseSQSURL := os.Getenv("RESPONSE_SQS_URL")
	if responseSQSURL == "" {
		panic("RESPONSE_SQS_URL not set in .env file")
	}

	extEvalKey := os.Getenv("EXTERNAL_EVAL_KEY")
	if extEvalKey == "" {
		panic("EXTERNAL_EVAL_KEY not set in .env file")
	}

	extEvalSqsUrl := os.Getenv("EXT_EVAL_SQS_URL")
	if extEvalSqsUrl == "" {
		panic("EXT_EVAL_SQS_URL not set in .env file")
	}

	esrvc := &EvalSrvc{
		sqsClient:      sqsClient,
		submSqsUrl:     submQueueUrl,
		evals:          []Evaluation{},
		evalsLock:      sync.Mutex{},
		repo:           nil,
		responseSqsUrl: responseSQSURL,
		extEvalKey:     extEvalKey,
		extEvalSqsUrl:  extEvalSqsUrl,
		accumulated:    xsync.NewMapOf[uuid.UUID, Pair[*sync.Cond, queues.Queue[Pair[Msg, time.Time]]]](),
	}

	return esrvc
}
