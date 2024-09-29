package tasksrvc

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type taskConstructor struct {
	task       *Task
	imgUuidMap []ImgUuidS3Pair
}

func newTaskConstructor() *taskConstructor {
	return &taskConstructor{
		task: &Task{},
	}
}

func (c *taskConstructor) applyDdbItem(item map[string]types.AttributeValue) error {
	skRaw, ok := item["sk"]
	if !ok {
		return fmt.Errorf("sk not found in item")
	}
	sk := skRaw.(*types.AttributeValueMemberS).Value

	if strings.HasPrefix(sk, "details#") {
		ddbr := ddbDetailsRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		c.task.ShortTaskID = ddbr.TaskCode
		c.task.TaskFullName = ddbr.FullName
		c.task.MemoryLimitMegabytes = ddbr.MemMbytes
		c.task.CPUTimeLimitSeconds = ddbr.CpuSecs
		c.task.DifficultyRating = ddbr.Difficulty
		c.task.OriginOlympiad = ddbr.OriginOlymp
		if ddbr.IllustrKey != nil {
			illstrImgUrl := fmt.Sprintf("%s%s", publicCloudfrontURLPrefix, *ddbr.IllustrKey)
			c.task.IllustrationImgURL = &illstrImgUrl
		}
	} else if strings.HasPrefix(sk, "vis_inp_sts#") {
		ddbr := ddbVisInpStsRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		if c.task.VisibleInputSubtasks == nil {
			c.task.VisibleInputSubtasks = make([]StInputs, 0)
		}

		hasThisSubtask := false
		for _, subtask := range c.task.VisibleInputSubtasks {
			if subtask.Subtask == ddbr.Subtask {
				hasThisSubtask = true
				subtask.Inputs = append(subtask.Inputs, ddbr.Input)
				break
			}
		}
		if !hasThisSubtask {
			c.task.VisibleInputSubtasks = append(c.task.VisibleInputSubtasks, StInputs{
				Subtask: ddbr.Subtask,
				Inputs:  []string{ddbr.Input},
			})
		}
	} else if strings.HasPrefix(sk, "test_groups#") {
		ddbr := ddbTestGroupsRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		if c.task.TestGroups == nil {
			c.task.TestGroups = make([]TestGroup, 0)
		}

		c.task.TestGroups = append(c.task.TestGroups, TestGroup{
			GroupID: ddbr.GroupId,
			Points:  ddbr.Points,
			Public:  ddbr.Public,
			Subtask: ddbr.Subtask,
			TestIDs: ddbr.TestIds,
		})

		// check if any tests belong to this group
		for i := 0; i < len(c.task.Tests); i++ {
			for j := 0; j < len(ddbr.TestIds); j++ {
				if c.task.Tests[i].TestID == ddbr.TestIds[j] {
					groupId := ddbr.GroupId
					c.task.Tests[i].TestGroup = &groupId
					break
				}
			}
		}
	} else if strings.HasPrefix(sk, "test_chsums#") {
		ddbr := ddbTestChsumsRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		if c.task.Tests == nil {
			c.task.Tests = make([]Test, 0)
		}

		c.task.Tests = append(c.task.Tests, Test{
			TestID:          ddbr.TestId,
			FullInputS3URI:  fmt.Sprintf("s3://proglv-tests/%s.zst", ddbr.InSha2),
			InputSha256:     ddbr.InSha2,
			FullAnswerS3URI: fmt.Sprintf("s3://proglv-tests/%s.zst", ddbr.AnsSha2),
			AnswerSha256:    ddbr.AnsSha2,
			Subtasks:        []int{},
			TestGroup:       nil,
		})

		testIdx := len(c.task.Tests) - 1

		// check if this test belong to some group
		for i := 0; i < len(c.task.TestGroups); i++ {
			tGroup := c.task.TestGroups[i]
			for j := 0; j < len(tGroup.TestIDs); j++ {
				if tGroup.TestIDs[j] == ddbr.TestId {
					c.task.Tests[testIdx].TestGroup = &tGroup.GroupID
				}
			}
		}
	} else if strings.HasPrefix(sk, "pdf_sttments#") {
		ddbr := ddbPdfSttmentsRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		pdfSttmntUrl := fmt.Sprintf("%s%s.pdf", publicCloudfrontURLPrefix, ddbr.PdfSha2)
		c.task.DefaultPdfStatementURL = &pdfSttmntUrl
	} else if strings.HasPrefix(sk, "md_sttments#") {
		ddbr := ddbMdSttmentsRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		if c.task.DefaultMdStatement == nil {
			c.task.DefaultMdStatement = &MarkdownStatement{
				LangISO639: ddbr.LangIso,
				Story:      ddbr.Story,
				Input:      ddbr.Input,
				Output:     ddbr.Output,
				Notes:      ddbr.Notes,
				Scoring:    ddbr.Scoring,
			}
		}
	} else if strings.HasPrefix(sk, "img_uuid_map#") {
		ddbr := ddbImgUuidMapRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		if c.imgUuidMap == nil {
			c.imgUuidMap = make([]ImgUuidS3Pair, 0)
		}

		c.imgUuidMap = append(c.imgUuidMap, ImgUuidS3Pair{
			UUID:  ddbr.Uuid,
			S3Key: ddbr.S3Key,
		})
	} else if strings.HasPrefix(sk, "examples#") {
		ddbr := ddbExamplesRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		if c.task.Examples == nil {
			c.task.Examples = make([]Example, 0)
		}

		c.task.Examples = append(c.task.Examples, Example{
			ExampleID: ddbr.ExampleId,
			Input:     ddbr.Input,
			Output:    ddbr.Output,
			MdNote:    ddbr.MdNote,
		})
	} else if strings.HasPrefix(sk, "origin_notes#") {
		ddbr := ddbOriginNotesRow{}
		err := attributevalue.UnmarshalMap(item, &ddbr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		if c.task.OriginNotes == nil {
			c.task.OriginNotes = make(map[string]string)
		}

		c.task.OriginNotes[ddbr.LangIso] = ddbr.OgInfo
	}

	return nil
}

func (c *taskConstructor) getTask() *Task {
	// replace uuids in markdown
	return c.task
}
