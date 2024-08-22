package subm

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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
				Value: submUuid,
			},
		},
	}

	result, err := s.ddbClient.Query(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}

	type ddbSubmTestGroupResult struct {
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

		TestGroupResults map[int]ddbSubmTestGroupResult
		TestsScoringRes  *SubmScoringTestsRow

		EvalDetails     map[string]*EvalDetailsRow // maps eval uuid to EvalDetailsRow
		EvalTestResults map[string][]*EvalTestResults
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
				submMap[submUuid].TestGroupResults = make(map[int]ddbSubmTestGroupResult)
			}

			// submScoringTestgroupRow.TestGroupID()
			testGroupId := submScoringTestgroupRow.TestGroupID()
			submMap[submUuid].TestGroupResults[testGroupId] = ddbSubmTestGroupResult{
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
		} else if strings.Contains(sortKey, "#test#") {
			// to get evaluation uuid, split by #, take the second element
			hashTagParts := strings.Split(sortKey, "#")
			evalUuid := hashTagParts[1]
			// if the evaluation uuid is not in the map, create a new slice
			if _, ok := submMap[submUuid].EvalTestResults[evalUuid]; !ok {
				if submMap[submUuid].EvalTestResults == nil {
					submMap[submUuid].EvalTestResults = make(map[string][]*EvalTestResults)
				}
				submMap[submUuid].EvalTestResults[evalUuid] = make([]*EvalTestResults, 0)
			}
			// to get test id, split by #, take the last element
			testIdStr := hashTagParts[len(hashTagParts)-1]
			// convert to integer
			testId, err := strconv.Atoi(testIdStr)
			if err != nil {
				return nil, fmt.Errorf("failed to convert test id to integer: %w", err)
			}

			// parse it as EvalTestRow
			var evalTestRow EvalTestRow
			err = attributevalue.UnmarshalMap(item, &evalTestRow)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal item: %w", err)
			}

			newEntry := &EvalTestResults{
				TestId:               testId,
				Reached:              evalTestRow.Reached,
				Ignored:              evalTestRow.Ignored,
				Finished:             evalTestRow.Finished,
				InputTrimmed:         evalTestRow.InputTrimmed,
				AnswerTrimmed:        evalTestRow.AnswerTrimmed,
				Subtasks:             evalTestRow.Subtasks,
				TestGroup:            evalTestRow.TestGroup,
				SubmCpuTimeMillis:    evalTestRow.SubmCpuTimeMillis,
				SubmMemKibiBytes:     evalTestRow.SubmMemoryKibiBytes,
				SubmWallTime:         evalTestRow.SubmWallTimeMillis,
				SubmExitCode:         evalTestRow.SubmExitCode,
				SubmStdoutTrimmed:    evalTestRow.SubmStdout,
				SubmStderrTrimmed:    evalTestRow.SubmStderr,
				CheckerCpuTimeMillis: evalTestRow.CheckerCpuTimeMillis,
				CheckerMemKibiBytes:  evalTestRow.CheckerMemoryKibiBytes,
				CheckerWallTime:      evalTestRow.CheckerWallTimeMillis,
				CheckerExitCode:      evalTestRow.CheckerExitCode,
				CheckerStdoutTrimmed: evalTestRow.CheckerStdout,
				CheckerStderrTrimmed: evalTestRow.CheckerStderr,
				SubmExitSignal:       evalTestRow.SubmExitSignal,
			}

			submMap[submUuid].EvalTestResults[evalUuid] = append(submMap[submUuid].EvalTestResults[evalUuid], newEntry)
		} else if strings.Contains(sortKey, "eval#") && strings.Contains(sortKey, "#details") {
			// parse it as EvalDetailsRow
			var evalDetailsRow EvalDetailsRow
			err := attributevalue.UnmarshalMap(item, &evalDetailsRow)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal item: %w", err)
			}

			if _, ok := submMap[submUuid].EvalDetails[evalDetailsRow.EvalUuid]; !ok {
				if submMap[submUuid].EvalDetails == nil {
					submMap[submUuid].EvalDetails = make(map[string]*EvalDetailsRow)
				}
				submMap[submUuid].EvalDetails[evalDetailsRow.EvalUuid] = &evalDetailsRow
			}
		} else if strings.Contains(sortKey, "subm#scoring#tests") {
			// parse it as SubmScoringTestsRow
			var submScoringTestsRow SubmScoringTestsRow
			err := attributevalue.UnmarshalMap(item, &submScoringTestsRow)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal item: %w", err)
			}

			submMap[submUuid].TestsScoringRes = &submScoringTestsRow
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
		evalDetails, ok := v.EvalDetails[v.CurrEvalUuid]
		if !ok {
			return nil, fmt.Errorf("eval details not found for eval uuid %s", v.CurrEvalUuid)
		}
		var testResults []*EvalTestResults
		for evalUuid, evalTestResults := range v.EvalTestResults {
			if evalUuid != v.CurrEvalUuid {
				continue
			}
			for _, evalTestResult := range evalTestResults {
				if evalTestResult.SubmCpuTimeMillis != nil && evalDetails.CpuTimeLimitMillis != nil {
					res := *evalTestResult.SubmCpuTimeMillis > *evalDetails.CpuTimeLimitMillis
					evalTestResult.TimeLimitExceeded = &res
				}
				if evalTestResult.SubmMemKibiBytes != nil && evalDetails.MemLimitKibiBytes != nil {
					res := *evalTestResult.SubmMemKibiBytes > *evalDetails.MemLimitKibiBytes
					evalTestResult.MemoryLimitExceeded = &res
				}
			}
			testResults = evalTestResults
		}
		// parse rfc 3339
		createdAt := time.Time{}
		if v.CreatedAtRfc3339 != "" {
			createdAt, err = time.Parse(time.RFC3339, v.CreatedAtRfc3339)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at: %w", err)
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

		// TODO: check if user can see submission code, test inputs, answers, etc.
		subm := &FullSubmission{
			BriefSubmission: BriefSubmission{
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
			},
			SubmContent:     v.SubmContent,
			EvalTestResults: testResults,
			EvalDetails: &EvalDetails{
				EvalUuid:             evalDetails.EvalUuid,
				CreatedAt:            createdAt,
				ErrorMsg:             evalDetails.ErrorMsg,
				EvalStage:            evalDetails.EvaluationStage,
				CpuTimeLimitMillis:   evalDetails.CpuTimeLimitMillis,
				MemoryLimitKibiBytes: evalDetails.MemLimitKibiBytes,
				ProgrammingLang: ProgrammingLang{
					ID:               evalDetails.ProgrammingLang.PLangId,
					FullName:         evalDetails.ProgrammingLang.DisplayName,
					CodeFilename:     evalDetails.ProgrammingLang.SubmCodeFname,
					CompileCmd:       evalDetails.ProgrammingLang.CompileCommand,
					ExecuteCmd:       evalDetails.ProgrammingLang.ExecCommand,
					CompiledFilename: evalDetails.ProgrammingLang.CompiledFname,
				},
				SystemInformation:    evalDetails.SystemInformation,
				CompileCpuTimeMillis: evalDetails.SubmCompileCpuTimeMillis,
				CompileMemKibiBytes:  evalDetails.SubmCompileMemoryKibiBytes,
				CompileWallTime:      evalDetails.SubmCompileWallTimeMillis,
				CompileExitCode:      evalDetails.SubmCompileExitCode,
				CompileStdoutTrimmed: evalDetails.SubmCompileStdout,
				CompileStderrTrimmed: evalDetails.SubmCompileStderr,
			},
		}
		res = append(res, subm)
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
