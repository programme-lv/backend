package tasksrvc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"mime"
)

// PutTask creates a new task with its details and visualization input statuses.
func (ts *TaskService) PutTask(in *PutPublicTaskInput) (err error) {
	rows := []ddbItemStruct{}

	rows = append(rows, ddbDetailsRow{
		TaskCode:    in.TaskCode,
		FullName:    in.FullName,
		MemMbytes:   in.MemMBytes,
		CpuSecs:     in.CpuSecs,
		Difficulty:  in.Difficulty,
		OriginOlymp: in.OriginOlymp,
		IllustrKey:  in.IllustrKey,
	})

	for _, visInpSt := range in.VisInpSts {
		for _, input := range visInpSt.Inputs {
			rows = append(rows, ddbVisInpStsRow{
				TaskCode: in.TaskCode,
				Subtask:  visInpSt.Subtask,
				TestId:   input.TestID,
				Input:    input.Input,
			})
		}
	}

	return ts.PutItems(context.TODO(), rows...)
}

func (ts *TaskService) UploadStatementPdf(body []byte) (sha2 string, err error) {
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s/%s.pdf", "task-pdf-statements", shaHex)
	err = ts.s3PublicBucket.Upload(body, s3Key, "application/pdf")
	return shaHex, err
}

// UploadIllustrationImg uploads an illustration image with the given MIME type
// and content to S3. It returns the S3 key or an error if the process fails.
func (ts *TaskService) UploadIllustrationImg(mimeType string, body []byte) (s3key string, err error) {
	sha2 := ts.Sha2Hex(body)
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extennsion not found")
	}
	ext := exts[0]
	s3Key := fmt.Sprintf("%s/%s%s", "task-illustrations", sha2, ext)
	err = ts.s3PublicBucket.Upload(body, s3Key, mimeType)
	return s3Key, err
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
