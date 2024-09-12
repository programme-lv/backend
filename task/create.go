package task

import (
	"context"
	"crypto/sha256"
	"fmt"
	"maps"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gabriel-vasile/mimetype"
)

type CreatePublicTaskInput struct {
	TaskCode    string
	FullName    string  // full name of the task
	MemMBytes   int     // max memory usage during execution in megabytes
	CpuSecs     float64 // max execution cpu time in seconds
	Difficulty  *int    // integer from 1 to 5. 1 - very easy, 5 - very hard
	OriginOlymp string  // name of the olympiad where the task was used
	IllustrKey  *string // s3 key for bucket "proglv-public"
	VisInpSts   []struct {
		Subtask int
		Inputs  []string
	}
	TestGroups []struct {
		GroupID int
		Points  int
		Public  bool
		Subtask int
		TestIDs []int
	}
	TestChsums []struct {
		TestID  int
		InSha2  string
		AnsSha2 string
	}
	PdfSttments []struct {
		LangIso639 string
		PdfSha2    string
	}
	MdSttments []struct {
		LangIso639 string
		Story      string
		Input      string
		Output     string
		Score      string
	}
	ImgUuidMap []struct {
		Uuid  string
		S3Key string
	}
	Examples []struct {
		ExampleID int
		Input     string
		Output    string
	}
	OriginNotes []struct {
		LangIso639 string
		OgInfo     string
	}
}

type ddbDetailsRow struct {
	TaskCode    string  `dynamodbav:"task_code"`
	FullName    string  `dynamodbav:"full_name"`
	MemMbytes   int     `dynamodbav:"mem_mbytes"`
	CpuSecs     float64 `dynamodbav:"cpu_secs"`
	Difficulty  *int    `dynamodbav:"difficulty"`
	OriginOlymp string  `dynamodbav:"origin_olymp"`
	IllustrKey  *string `dynamodbav:"illustr_key"`
}

func (row ddbDetailsRow) GetKey() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: "details#"},
	}
}

func (ts *TaskService) CreateTask(in *CreatePublicTaskInput) (err error) {
	detailsRow := ddbDetailsRow{
		TaskCode:    in.TaskCode,
		FullName:    in.FullName,
		MemMbytes:   in.MemMBytes,
		CpuSecs:     in.CpuSecs,
		Difficulty:  in.Difficulty,
		OriginOlymp: in.OriginOlymp,
		IllustrKey:  in.IllustrKey,
	}

	item, err := attributevalue.MarshalMap(detailsRow)
	if err != nil {
		return fmt.Errorf("failed to marshal ddb row: %w", err)
	}

	maps.Copy(item, detailsRow.GetKey())

	_, err = ts.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &ts.taskTableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put task details: %w", err)
	}

	return nil
}

func (ts *TaskService) UploadStatementPdf(body []byte) (sha2 string, err error) {
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s/%s.pdf", "task-pdf-statements", shaHex)
	err = ts.s3PublicBucket.Upload(body, s3Key, "application/pdf")
	return shaHex, err
}

func (ts *TaskService) UploadIllustrationImg(mimeType mimetype.MIME, body []byte) (err error) {
	sha2 := ts.Sha2Hex(body)
	ext := mimeType.Extension()
	s3Key := fmt.Sprintf("%s/%s%s", "task-illustrations", sha2, ext)
	err = ts.s3PublicBucket.Upload(body, s3Key, mimeType.String())
	return err
}

func (ts *TaskService) Sha2Hex(body []byte) (sha2 string) {
	hash := sha256.Sum256(body)
	sha2 = fmt.Sprintf("%x", hash[:])
	return
}

/*
PK=task#{task_code},SK=details#
- task_code    (string)
- full_name    (string)
- mem_mbytes   (int)
- cpu_secs     (float)
- problem_tags (list)
- difficulty   (int)
- authors      (list)
- origin_olymp (string)
- illustr_key  (string)

PK=task#{task_code},SK=vis_inp_sts#{subtask}#{test_id}
- input (string)

PK=task#{task_code},SK=test_groups#{group_id}
- points   (int)
- public   (boolean)
- subtask  (int)
- test_ids (int list)

PK=task#{task_code},SK=test_chsums#{test_id}
- in_sha2  (string)
- ans_sha2 (string)

PK=task#{task_code},SK=pdf_sttments#{lang_iso639}
- pdf_sha2 (string)

PK=task#{task_code},SK=md_sttments#{lang_iso639}
- story  (string)
- input  (string)
- output (string)
- score  (string)

PK=task#{task_code},SK=img_uuid_map#{img_uuid}
- s3_key (string)

PK=task#{task_code},SK=examples#{example_id}
- input  (string)
- output (string)

PK=task#{task_code},SK=origin_notes#{lang_iso639}
- og_info (string)
*/
