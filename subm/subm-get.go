package subm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (s *SubmissionSrvc) GetSubmission(ctx context.Context, submUuid string) (*FullSubmission, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.submTableName),
		KeyConditionExpression: aws.String("subm_uuid = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{
				Value: "51ca9748-44a1-4d9a-9bbf-a15581f938b5",
			},
		},
	}

	result, err := s.ddbClient.Query(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}

	type ddmSubmTestGroupResult struct {
		Score    int
		Accepted int
		Wrong    int
		Untested int
		Subtask  int
	}

	type ddbSubm struct {
		SubmUuid         string
		SubmContent      string
		AuthorUuid       string
		TaskId           string
		ProgLangId       string
		CurrEvalUuid     string
		CurrEvalStatus   string
		ErrorMsg         *string
		CreatedAtRfc3339 string
		foundDetailsRow  bool

		TestGroupResults map[int]ddmSubmTestGroupResult
	}

	// note the order of the items should be that all items that provide additional information are lexicographically  > ...#details

	submMap := make(map[string]*ddbSubm) // subm_uuid -> submission
	for _, item := range result.Items {
		submUuid := item["subm_uuid"].(*types.AttributeValueMemberS).Value
		if _, ok := submMap[submUuid]; !ok {
			submMap[submUuid] = &ddbSubm{
				SubmUuid:         submUuid,
				TestGroupResults: nil,
			}
		}

		sortKey := item["sort_key"].(*types.AttributeValueMemberS).Value
		// check if "sort_key" starts with subm#scoring#testgroup#
		if strings.HasPrefix(sortKey, "subm#scoring#testgroup#") {
			// parse it as SubmScoringTestgroupRow

			var submScoringTestgroupRow SubmScoringTestgroupRow
			err := attributevalue.UnmarshalMap(item, &submScoringTestgroupRow)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal item: %w", err)
			}

			if submMap[submUuid].TestGroupResults == nil {
				submMap[submUuid].TestGroupResults = make(map[int]ddmSubmTestGroupResult)
			}

			// submScoringTestgroupRow.TestGroupID()
			testGroupId := submScoringTestgroupRow.TestGroupID()
			submMap[submUuid].TestGroupResults[testGroupId] = ddmSubmTestGroupResult{
				Score:    submScoringTestgroupRow.TestgroupScore,
				Accepted: submScoringTestgroupRow.AcceptedTests,
				Wrong:    submScoringTestgroupRow.WrongTests,
				Untested: submScoringTestgroupRow.UntestedTests,
				Subtask:  submScoringTestgroupRow.StatementSubtask,
			}
		} else if strings.HasPrefix(sortKey, "subm#details") {
			// parse it as SubmDetailsRow
			var submDetailsRow SubmDetailsRow
			err := attributevalue.UnmarshalMap(item, &submDetailsRow)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal item: %w", err)
			}

			submMap[submUuid].foundDetailsRow = true
			submMap[submUuid].SubmContent = submDetailsRow.Content
			submMap[submUuid].AuthorUuid = submDetailsRow.AuthorUuid
			submMap[submUuid].TaskId = submDetailsRow.TaskId
			submMap[submUuid].ProgLangId = submDetailsRow.ProgLangId
			submMap[submUuid].CurrEvalUuid = submDetailsRow.CurrentEvalUuid
			submMap[submUuid].CurrEvalStatus = submDetailsRow.CurrentEvalStatus
			submMap[submUuid].ErrorMsg = submDetailsRow.ErrorMsg
			submMap[submUuid].CreatedAtRfc3339 = submDetailsRow.CreatedAtRfc3339
		}
	}

	// TODO: cache task list
	tasks, err := s.taskSrvc.ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	mapTaskIdToName := make(map[string]string)
	for _, task := range tasks {
		mapTaskIdToName[task.PublishedTaskID] = task.TaskFullName
	}

	// TODO: cache submission list
	users, err := s.userSrvc.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	mapUserUuidToUsername := make(map[string]string)
	for _, user := range users {
		mapUserUuidToUsername[user.UUID] = user.Username
	}

	pLangs, err := s.ListProgrammingLanguages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch programming languages: %w", err)
	}
	mapPLangIdToDisplayName := make(map[string]string)
	mapPLangIdToMonacoId := make(map[string]string)
	for _, pLang := range pLangs {
		mapPLangIdToDisplayName[pLang.ID] = pLang.FullName
		mapPLangIdToMonacoId[pLang.ID] = pLang.MonacoId
	}

	res := make([]*FullSubmission, 0, len(submMap))
	for k, v := range submMap {
		if !v.foundDetailsRow {
			continue
		}
		var testgroups []*TestGroupResult = nil
		if v.TestGroupResults != nil {
			testgroups = make([]*TestGroupResult, 0, len(v.TestGroupResults))
			for testGroupId, testGroupResult := range v.TestGroupResults {
				testgroups = append(testgroups, &TestGroupResult{
					TestGroupID:      testGroupId,
					TestGroupScore:   testGroupResult.Score,
					StatementSubtask: testGroupResult.Subtask,
					AcceptedTests:    testGroupResult.Accepted,
					WrongTests:       testGroupResult.Wrong,
					UntestedTests:    testGroupResult.Untested,
				})
			}
		}
		res = append(res, &FullSubmission{
			BriefSubmission: BriefSubmission{
				SubmUUID:              k,
				EvalUUID:              v.CurrEvalUuid,
				Username:              mapUserUuidToUsername[v.AuthorUuid],
				CreatedAt:             v.CreatedAtRfc3339,
				EvalStatus:            v.CurrEvalStatus,
				EvalScoringTestgroups: testgroups,
				EvalScoringTests:      nil, // TODO: tests
				EvalScoringSubtasks:   nil, // TODO: subtasks
				PLangID:               v.ProgLangId,
				PLangDisplayName:      mapPLangIdToDisplayName[v.ProgLangId],
				PLangMonacoID:         mapPLangIdToMonacoId[v.ProgLangId],
				TaskName:              mapTaskIdToName[v.TaskId],
				TaskID:                v.TaskId,
			},
			SubmContent: v.SubmContent,
		},
		)
	}

	// sort by created_at descending
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt > res[j].CreatedAt
	})

	if len(res) == 0 {
		return nil, newErrSubmissionNotFound()
	}
	if len(res) > 1 {
		return nil, fmt.Errorf("found more than one submission with the same uuid")
	}

	return res[0], nil
}
