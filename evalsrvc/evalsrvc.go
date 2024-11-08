package evalsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

func (e *EvalSrvc) Enqueue(req Request) (uuid.UUID, error) {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal evaluation request: %w", err)
	}
	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate UUID: %w", err)
	}
	_, err = e.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    &e.submSqsUrl,
		MessageBody: aws.String(string(jsonReq)),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to send message to evaluation queue: %w", err)
	}

	return evalUuid, nil
}

func (e *EvalSrvc) Receive() ([]Msg, error) {
	output, err := e.sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(e.resSqsUrl),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     5,
	})
	if err != nil {
		log.Printf("failed to receive messages, %v\n", err)
		time.Sleep(1 * time.Second)
		return nil, err
	}
	msgs := make([]Msg, len(output.Messages))
	for i, msg := range output.Messages {
		msgs[i] = &EvalMsg{msg}
	}
	return msgs, nil
}

type Msg interface {
	GetId() string
}

type EvalSrvc struct {
	sqsClient *sqs.Client

	submSqsUrl string
	resSqsUrl  string
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

	return &EvalSrvc{
		sqsClient:  sqsClient,
		submSqsUrl: submQueueUrl,
		resSqsUrl:  responseSQSURL,
	}
}
