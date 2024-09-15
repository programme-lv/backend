package tasksrvc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"mime"

	"github.com/klauspost/compress/zstd"
)

// PutTask creates a new task with its details and visualization input statuses.
func (ts *TaskService) PutTask(in *PutPublicTaskInput) (err error) {
	err = ts.DeleteTask(in.TaskCode)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows := []ddbItemStruct{}
	put := func(row ddbItemStruct) { rows = append(rows, row) }

	put(ddbDetailsRow{
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
			put(ddbVisInpStsRow{
				TaskCode: in.TaskCode,
				Subtask:  visInpSt.Subtask,
				TestId:   input.TestID,
				Input:    input.Input,
			})
		}
	}

	for _, group := range in.TestGroups {
		put(ddbTestGroupsRow{
			TaskCode: in.TaskCode,
			GroupId:  group.GroupID,
			Points:   group.Points,
			Public:   group.Public,
			Subtask:  group.Subtask,
			TestIds:  group.TestIDs,
		})
	}

	for _, test := range in.TestChsums {
		put(ddbTestChsumsRow{
			TaskCode: in.TaskCode,
			TestId:   test.TestID,
			InSha2:   test.InSHA2,
			AnsSha2:  test.AnsSHA2,
		})
	}

	for _, pdfSttmnt := range in.PdfSttments {
		put(ddbPdfSttmentsRow{
			TaskCode: in.TaskCode,
			LangIso:  pdfSttmnt.LangISO639,
			PdfSha2:  pdfSttmnt.PdfSHA2,
		})
	}

	for _, mdSttmnt := range in.MdSttments {
		put(ddbMdSttmentsRow{
			TaskCode: in.TaskCode,
			LangIso:  mdSttmnt.LangISO639,
			Story:    mdSttmnt.Story,
			Input:    mdSttmnt.Input,
			Output:   mdSttmnt.Output,
			Scoring:  mdSttmnt.Scoring,
			Notes:    mdSttmnt.Notes,
		})
	}

	for _, img := range in.ImgUuidMap {
		put(ddbImgUuidMapRow{
			TaskCode: in.TaskCode,
			Uuid:     img.UUID,
			S3Key:    img.S3Key,
		})
	}

	for _, example := range in.Examples {
		put(ddbExamplesRow{
			TaskCode:  in.TaskCode,
			ExampleId: example.ExampleID,
			Input:     example.Input,
			Output:    example.Output,
			MdNote:    example.MdNote,
		})
	}

	for _, origNote := range in.OriginNotes {
		put(ddbOriginNotesRow{
			TaskCode: in.TaskCode,
			LangIso:  origNote.LangISO639,
			OgInfo:   origNote.OgInfo,
		})
	}

	return ts.PutItems(context.TODO(), rows...)
}

// UploadStatementPdf uploads a PDF statement with the given content to S3.
// It returns the S3 key or an error if the process fails.
//
// S3 key format: "task-pdf-statements/<sha2>.pdf"
func (ts *TaskService) UploadStatementPdf(body []byte) (string, error) {
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s/%s.pdf", "task-pdf-statements", shaHex)
	err := ts.s3PublicBucket.Upload(body, s3Key, "application/pdf")
	return s3Key, err
}

// UploadIllustrationImg uploads an image with the given content and MIME type to S3.
// It returns the S3 key or an error if the process fails.
//
// S3 key format: "task-illustrations/<sha2>.<ext>"
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

// UploadMarkdownImage uploads an image with the given MIME type and content
// to S3. It returns the S3 key or an error if the process fails.
//
// S3 key format: "task-md-images/<sha2>.<extension>"
func (ts *TaskService) UploadMarkdownImage(mimeType string, body []byte) (s3key string, err error) {
	sha2 := ts.Sha2Hex(body)
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extennsion not found")
	}
	ext := exts[0]
	s3Key := fmt.Sprintf("%s/%s%s", "task-md-images", sha2, ext)
	err = ts.s3PublicBucket.Upload(body, s3Key, mimeType)
	return s3Key, err
}

// UploadTest uploads a test input or output to S3 after compressing it with Zstandard.
// The S3 key is the SHA256 hash of the content with a .zst extension.
func (ts *TaskService) UploadTest(body []byte) error {
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s.zst", shaHex)
	mediaType := "application/zstd"

	exists, err := ts.s3TestfileBucket.Exists(s3Key)
	if err != nil {
		return fmt.Errorf("failed to check if object exists in S3: %w", err)
	}

	if exists {
		return nil
	}

	zstdCompressed, err := compressWithZstd(body)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	err = ts.s3TestfileBucket.Upload(zstdCompressed, s3Key, mediaType)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// compressWithZstd compresses the given data using Zstandard compression.
// It returns the compressed data or an error if the compression fails.
func compressWithZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
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
