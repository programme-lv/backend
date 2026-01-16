package exec

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/programme-lv/backend/plang"
)

func getPrLangById(id string) (PrLang, error) {
	lang, err := plang.GetProgrLangById(id)
	if err != nil {
		return PrLang{}, err
	}
	return PrLang{
		ShortId:   lang.ID,
		Display:   lang.FullName,
		CodeFname: lang.CodeFilename,
		CompCmd:   lang.CompileCmd,
		CompFname: lang.CompiledFilename,
		ExecCmd:   lang.ExecuteCmd,
	}, nil
}

func getSqsClientFromEnv() *sqs.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
	)
	if err != nil {
		panic(fmt.Errorf("unable to load SDK config, %v", err))
	}
	return sqs.NewFromConfig(cfg)
}

func getSqsClientFromEnvNoLogging() *sqs.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
		config.WithLogger(nil),
	)
	if err != nil {
		panic(fmt.Errorf("unable to load SDK config, %v", err))
	}
	return sqs.NewFromConfig(cfg)
}

func getResponseSqsUrlFromEnv() string {
	responseSQSURL := os.Getenv("SQS_RESPONSE_QUEUE_URL")
	if responseSQSURL == "" {
		panic("SQS_RESPONSE_QUEUE_URL not set in .env file")
	}
	return responseSQSURL
}

func getSubmSqsUrlFromEnv() string {
	submQueueUrl := os.Getenv("SQS_SUBM_QUEUE_URL")
	if submQueueUrl == "" {
		panic("SQS_SUBM_QUEUE_URL not set in .env file")
	}
	return submQueueUrl
}

func getExtPartnerPwFromEnv() string {
	extEvalKey := os.Getenv("EXTERNAL_EVAL_KEY")
	if extEvalKey == "" {
		panic("EXTERNAL_EVAL_KEY not set in .env file")
	}
	return extEvalKey
}

func getS3ClientFromEnv() *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
	)
	if err != nil {
		panic(fmt.Errorf("unable to load SDK config, %v", err))
	}
	return s3.NewFromConfig(cfg)
}
