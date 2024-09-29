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
func (s *SubmissionSrvc) ListSubmissions(ctx context.Context) (res []*BriefSubmission, err error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.submTableName),
		IndexName:              aws.String("gsi1_pk-gsi1_sk-index"),
		KeyConditionExpression: aws.String("gsi1_pk = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberN{Value: "1"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(200 * 10),
		// limit page size
	}

	result, err := s.ddbClient.Query(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
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

		TestGroupResults map[int]*SubmScoringTestgroupRow
		TestsScoringRes  *SubmScoringTestsRow
	}

	// note the order of the items should be that all items that provide additional information are lexicographically  > subm#details

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
				submMap[submUuid].TestGroupResults = make(map[int]*SubmScoringTestgroupRow)
			}

			// submScoringTestgroupRow.TestGroupID()
			testGroupId := submScoringTestgroupRow.TestGroupID()
			submMap[submUuid].TestGroupResults[testGroupId] = &submScoringTestgroupRow
		} else if strings.HasPrefix(sortKey, "subm#details") {
			// parse it as SubmDetailsRow
			var submDetailsRow SubmDetailsRow
			err := attributevalue.UnmarshalMap(item, &submDetailsRow)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal item: %w", err)
			}

			submMap[submUuid].foundDetailsRow = true
			submMap[submUuid].AuthorUuid = submDetailsRow.AuthorUuid
			submMap[submUuid].TaskId = submDetailsRow.TaskId
			submMap[submUuid].ProgLangId = submDetailsRow.ProgLangId
			submMap[submUuid].CurrEvalUuid = submDetailsRow.CurrentEvalUuid
			submMap[submUuid].CurrEvalStatus = submDetailsRow.CurrentEvalStatus
			submMap[submUuid].ErrorMsg = submDetailsRow.ErrorMsg
			submMap[submUuid].CreatedAtRfc3339 = submDetailsRow.CreatedAtRfc3339
		} else if strings.HasPrefix(sortKey, "subm#scoring#tests") {
			// parse it as SubmScoringTestsRow
			var submScoringTestsRow SubmScoringTestsRow
			err := attributevalue.UnmarshalMap(item, &submScoringTestsRow)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal item: %w", err)
			}

			submMap[submUuid].TestsScoringRes = &submScoringTestsRow
		}
	}

	tasks, err := s.taskSrvc.ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	mapTaskIdToName := make(map[string]string)
	for _, task := range tasks {
		mapTaskIdToName[task.ShortTaskID] = task.TaskFullName
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

	res = make([]*BriefSubmission, 0, len(submMap))
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
					TestGroupScore:   testGroupResult.TestgroupScore,
					StatementSubtask: testGroupResult.StatementSubtask,
					AcceptedTests:    testGroupResult.AcceptedTests,
					WrongTests:       testGroupResult.WrongTests,
					UntestedTests:    testGroupResult.UntestedTests,
				})
			}
		}
		var tests *TestsResult = nil
		if v.TestsScoringRes != nil {
			tests = &TestsResult{
				Accepted: v.TestsScoringRes.Accepted,
				Wrong:    v.TestsScoringRes.Wrong,
				Untested: v.TestsScoringRes.Untested,
			}
		}

		res = append(res, &BriefSubmission{
			SubmUUID:              k,
			EvalUUID:              v.CurrEvalUuid,
			Username:              mapUserUuidToUsername[v.AuthorUuid],
			CreatedAt:             v.CreatedAtRfc3339,
			EvalStatus:            v.CurrEvalStatus,
			EvalScoringTestgroups: testgroups,
			EvalScoringTests:      tests,
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

	// limit count to 30
	if len(res) > 30 {
		res = res[:30]
	}

	return res, nil
}
