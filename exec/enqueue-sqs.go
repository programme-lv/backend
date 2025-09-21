package exec

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	testerapi "github.com/programme-lv/tester/api"
)

// Enqueues code for evaluation into AWS SQS:
// 1. converts tests to tester format
// 2. marshall request to json
// 3. enqueues request to sqs queue
func enqueue(
	uuid uuid.UUID,
	code string,
	lang PrLang,
	tests []TestFile,
	params TestingParams,
	client *sqs.Client,
	submQ string,
) error {
	testsTester := make([]testerapi.Test, len(tests))
	for i, test := range tests {
		testsTester[i] = testerapi.Test{
			In: testerapi.File{
				Sha256:  test.InSha256,
				Url:     test.InDownlUrl,
				Content: test.InContent,
			},
			Ans: testerapi.File{
				Sha256:  test.AnsSha256,
				Url:     test.AnsDownlUrl,
				Content: test.AnsContent,
			},
		}
	}

	jsonReq, err := json.Marshal(testerapi.ExecReq{
		Uuid: uuid.String(),
		Code: code,
		Lang: testerapi.PrLang{
			LangName:      lang.Display,
			CodeFname:     lang.CodeFname,
			CompileCmd:    lang.CompCmd,
			CompiledFname: lang.CompFname,
			ExecCmd:       lang.ExecCmd,
		},
		Tests:      testsTester,
		Checker:    params.Checker,
		Interactor: params.Interactor,
		CpuMs:      int32(params.CpuMs),
		RamKiB:     int32(params.MemKiB),
	})
	if err != nil {
		format := "failed to marshal eval request: %w"
		errMsg := fmt.Errorf(format, err)
		return errMsg
	}

	zstdEncoder, err := zstd.NewWriter(nil)
	if err != nil {
		return fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer zstdEncoder.Close()

	compressed := zstdEncoder.EncodeAll(jsonReq, make([]byte, 0, len(jsonReq)))
	encoded := base64.StdEncoding.EncodeToString(compressed)

	_, err = client.SendMessage(context.TODO(),
		&sqs.SendMessageInput{
			QueueUrl:    aws.String(submQ),
			MessageBody: aws.String(encoded),
		})
	if err != nil {
		format := "failed to send message to eval queue: %w"
		errMsg := fmt.Errorf(format, err)
		return errMsg
	}

	return nil
}
