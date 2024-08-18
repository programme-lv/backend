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
	"goa.design/clue/log"
)

// List all submissions
func (s *SubmissionsService) ListSubmissions(ctx context.Context) (res []*Submission, err error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.submTableName),
		IndexName:              aws.String("gsi1_pk-gsi1_sk-index"),
		KeyConditionExpression: aws.String("gsi1_pk = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberN{
				Value: "1",
			},
		},
		ScanIndexForward: aws.Bool(false),
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

		TestGroupResults map[int]ddmSubmTestGroupResult
	}

	// TODO: the last item might not have all of its data because of the paginated query

	submMap := make(map[string]*ddbSubm) // subm_uuid -> submission
	log.Printf(ctx, "found %d items", len(result.Items))
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

	tasks, err := s.taskSrvc.ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	mapTaskIdToName := make(map[string]string)
	for _, task := range tasks {
		mapTaskIdToName[task.PublishedTaskID] = task.TaskFullName
	}

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

	res = make([]*Submission, 0, len(submMap))
	for k, v := range submMap {
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
		res = append(res, &Submission{
			SubmUUID:              k,
			Submission:            v.SubmContent, // TODO: reconsider retrieving submission content in submission list
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
		})
	}

	// sort by created_at descending
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt > res[j].CreatedAt
	})

	return res, nil
}
