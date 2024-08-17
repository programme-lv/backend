package subm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"golang.org/x/exp/rand"
)

func (s *SubmissionsService) processEvalResult(evalUuid string, msgType string, fields *json.RawMessage) {
	baseDelay := 100 * time.Millisecond

	var submUuid string
	var foundInMap bool
	if submUuid, foundInMap = s.evalUuidToSubmUuid[evalUuid]; !foundInMap {
		for attempt := 0; attempt < 5; attempt++ {
			// sleep for 100ms * attempt^2
			if attempt > 0 {
				time.Sleep(baseDelay * time.Duration(attempt*attempt))
			}
			// Create the query input
			input := &dynamodb.QueryInput{
				TableName:              aws.String(s.submTableName),
				IndexName:              aws.String("eval_uuid-index"),
				KeyConditionExpression: aws.String("eval_uuid = :evalUUID"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":evalUUID": &types.AttributeValueMemberS{Value: evalUuid},
				},
				ProjectionExpression: aws.String("subm_uuid"),
			}

			// Execute the query
			result, err := s.ddbClient.Query(context.TODO(), input)
			if err != nil {
				log.Printf("failed to query items: %v", err)
				continue
			}

			switch len(result.Items) {
			case 0:
				log.Printf("no submission found for eval_uuid %s", evalUuid)
				continue
			case 1:
				// Success: return the subm_uuid
				submUuid = result.Items[0]["subm_uuid"].(*types.AttributeValueMemberS).Value
			}
		}
	}
	if submUuid == "" {
		log.Printf("failed to find submission for eval_uuid %s", evalUuid)
		return
	}

	switch msgType {
	case "started_evaluation":
		var parsed struct {
			SystemInfo string `json:"system_info"`
			// StartedTime string `json:"started_time"`
		}
		err := json.Unmarshal(*fields, &parsed)
		if err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}

		// 1. SAVE THE SYSTEM INFO IN THE EVAL DETAILS ROW
		// the partition key is subm_uuid, sort key is eval#<eval_uuid>#details
		evalDetailsPk := map[string]types.AttributeValue{
			"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
			"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#details"},
		}

		updSysInfo := expression.Set(expression.Name("system_information"), expression.Value(parsed.SystemInfo))
		updSysInfoExpr, err := expression.NewBuilder().WithUpdate(updSysInfo).Build()
		if err != nil {
			log.Printf("failed to build system information update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key:                       evalDetailsPk,
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updSysInfoExpr.Update(),
			ExpressionAttributeValues: updSysInfoExpr.Values(),
			ExpressionAttributeNames:  updSysInfoExpr.Names(),
		})

		if err != nil {
			log.Printf("failed to update system info: %v", err)
			return
		}

		// 2. EVALUATION STAGE SHOULD CHANGE FROM "WAITING" TO "RECEIVED" IF IT IS STILL "WAITING"
		// 2.1 update "evaluation_stage" to "received" if it is still "waiting" for the eval details row
		// the partition key is subm_uuid, sort key is eval#<eval_uuid>#details

		updEvalStage := expression.Set(expression.Name("evaluation_stage"), expression.Value("received"))
		updEvalStageCond := expression.Name("evaluation_stage").Equal(expression.Value("waiting"))
		updEvalStageExpr, err := expression.NewBuilder().WithUpdate(updEvalStage).WithCondition(updEvalStageCond).Build()
		if err != nil {
			log.Printf("failed to build evaluation stage update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key:                       evalDetailsPk,
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updEvalStageExpr.Update(),
			ConditionExpression:       updEvalStageExpr.Condition(),
			ExpressionAttributeValues: updEvalStageExpr.Values(),
			ExpressionAttributeNames:  updEvalStageExpr.Names(),
		})

		if err != nil {
			var condFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condFailed) {
				log.Printf("failed to update eval stage because the condition failed: %v", err)
			} else {
				log.Printf("failed to update evaluation stage: %v", err)
				return
			}
		}
		// 2.2 update "current_eval_status" to "received" for the submission details row if it is still "waiting" and the current_eval_uuid is the same as eval_uuid
		updSubmCurrEvalStatus := expression.Set(expression.Name("current_eval_status"), expression.Value("received"))
		updSubmCurrEvalStatusCond := expression.Name("current_eval_status").Equal(expression.Value("waiting")).And(
			expression.Name("current_eval_uuid").Equal(expression.Value(evalUuid)))
		updSubmCurrEvalStatusExpr, err := expression.NewBuilder().WithUpdate(updSubmCurrEvalStatus).WithCondition(updSubmCurrEvalStatusCond).Build()
		if err != nil {
			log.Printf("failed to build current eval status update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key: map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "subm#details"},
			},
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updSubmCurrEvalStatusExpr.Update(),
			ConditionExpression:       updSubmCurrEvalStatusExpr.Condition(),
			ExpressionAttributeValues: updSubmCurrEvalStatusExpr.Values(),
			ExpressionAttributeNames:  updSubmCurrEvalStatusExpr.Names(),
		})

		if err != nil {
			var condFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condFailed) {
				log.Printf("failed to update current eval status because the condition failed: %v", err)
			} else {
				log.Printf("failed to update current eval status: %v", err)
				return
			}
		}

		s.updateSubmStateChan <- &SubmissionStateUpdate{
			SubmUuid: submUuid,
			EvalUuid: evalUuid,
			NewState: "received",
		}
	case "started_compilation":
		// update evaluation details row with "evaluation_stage" = "compiling" if it is either "received" or "waiting"
		updEvalStage := expression.Set(expression.Name("evaluation_stage"), expression.Value("compiling"))
		updEvalStageCond := expression.Name("evaluation_stage").In(expression.Value("received"), expression.Value("waiting"))
		updEvalStageExpr, err := expression.NewBuilder().WithUpdate(updEvalStage).WithCondition(updEvalStageCond).Build()
		if err != nil {
			log.Printf("failed to build evaluation stage update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key: map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#details"},
			},
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updEvalStageExpr.Update(),
			ConditionExpression:       updEvalStageExpr.Condition(),
			ExpressionAttributeValues: updEvalStageExpr.Values(),
			ExpressionAttributeNames:  updEvalStageExpr.Names(),
		})

		if err != nil {
			var condFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condFailed) {
				log.Printf("failed to update eval stage because the condition failed: %v", err)
			} else {
				log.Printf("failed to update evaluation stage: %v", err)
				return
			}
		}

		// update submission details row with "current_eval_status" = "compiling" if it is either "received" or "waiting" and the current_eval_uuid is the same as eval_uuid
		updSubmCurrEvalStatus := expression.Set(expression.Name("current_eval_status"), expression.Value("compiling"))
		updSubmCurrEvalStatusCond := expression.Name("current_eval_status").In(expression.Value("received"), expression.Value("waiting")).And(
			expression.Name("current_eval_uuid").Equal(expression.Value(evalUuid)))
		updSubmCurrEvalStatusExpr, err := expression.NewBuilder().WithUpdate(updSubmCurrEvalStatus).WithCondition(updSubmCurrEvalStatusCond).Build()
		if err != nil {
			log.Printf("failed to build current eval status update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key: map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "subm#details"},
			},
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updSubmCurrEvalStatusExpr.Update(),
			ConditionExpression:       updSubmCurrEvalStatusExpr.Condition(),
			ExpressionAttributeValues: updSubmCurrEvalStatusExpr.Values(),
			ExpressionAttributeNames:  updSubmCurrEvalStatusExpr.Names(),
		})

		if err != nil {
			var condFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condFailed) {
				log.Printf("failed to update current eval status because the condition failed: %v", err)
			} else {
				log.Printf("failed to update current eval status: %v", err)
				return
			}
		}

		s.updateSubmStateChan <- &SubmissionStateUpdate{
			SubmUuid: submUuid,
			EvalUuid: evalUuid,
			NewState: "compiling",
		}
	case "finished_compilation":
		var parsed struct {
			RuntimeData struct {
				Stdout          *string `json:"stdout"`
				Stderr          *string `json:"stderr"`
				ExitCode        int64   `json:"exit_code"`
				CpuTimeMillis   int64   `json:"cpu_time_millis"`
				WallTimeMillis  int64   `json:"wall_time_millis"`
				MemoryKibiBytes int64   `json:"memory_kibibytes"`
			} `json:"runtime_data"`
		}
		err := json.Unmarshal(*fields, &parsed)
		if err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}

		// update eval details row attributes "subm_comp_stdout", "subm_comp_stderr", "subm_comp_exit_code", "subm_comp_cpu_time_millis", "subm_comp_wall_time_millis", "subm_comp_memory_kibi_bytes"
		// the partition key is subm_uuid, sort key is eval#<eval_uuid>#details
		evalDetailsPk := map[string]types.AttributeValue{
			"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
			"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#details"},
		}

		updCompDetails := expression.Set(
			expression.Name("subm_comp_stdout"), expression.Value(parsed.RuntimeData.Stdout)).
			Set(expression.Name("subm_comp_stderr"), expression.Value(parsed.RuntimeData.Stderr)).
			Set(expression.Name("subm_comp_exit_code"), expression.Value(parsed.RuntimeData.ExitCode)).
			Set(expression.Name("subm_comp_cpu_time_millis"), expression.Value(parsed.RuntimeData.CpuTimeMillis)).
			Set(expression.Name("subm_comp_wall_time_millis"), expression.Value(parsed.RuntimeData.WallTimeMillis)).
			Set(expression.Name("subm_comp_memory_kibi_bytes"), expression.Value(parsed.RuntimeData.MemoryKibiBytes))

		updCompDetailsExpr, err := expression.NewBuilder().WithUpdate(updCompDetails).Build()
		if err != nil {
			log.Printf("failed to build compilation details update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key:                       evalDetailsPk,
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updCompDetailsExpr.Update(),
			ExpressionAttributeValues: updCompDetailsExpr.Values(),
			ExpressionAttributeNames:  updCompDetailsExpr.Names(),
		})

		if err != nil {
			log.Printf("failed to update compilation details: %v", err)
			return
		}
	case "started_testing":
		// in evaluation details row change evaluation stage to "testing" if it is still "compiling" or "received" or "waiting"

		updEvalStage := expression.Set(expression.Name("evaluation_stage"), expression.Value("testing"))
		updEvalStageCond := expression.Name("evaluation_stage").In(expression.Value("compiling"), expression.Value("received"), expression.Value("waiting"))
		updEvalStageExpr, err := expression.NewBuilder().WithUpdate(updEvalStage).WithCondition(updEvalStageCond).Build()
		if err != nil {
			log.Printf("failed to build evaluation stage update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key: map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#details"},
			},
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updEvalStageExpr.Update(),
			ConditionExpression:       updEvalStageExpr.Condition(),
			ExpressionAttributeValues: updEvalStageExpr.Values(),
			ExpressionAttributeNames:  updEvalStageExpr.Names(),
		})

		if err != nil {
			var condFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condFailed) {
				log.Printf("failed to update eval stage because the condition failed: %v", err)
			} else {
				log.Printf("failed to update evaluation stage: %v", err)
				return
			}
		}

		// in submission details row change current_eval_status to "testing" if it is still "compiling" or "received" or "waiting" and the current_eval_uuid is the same as eval_uuid
		updSubmCurrEvalStatus := expression.Set(expression.Name("current_eval_status"), expression.Value("testing"))
		updSubmCurrEvalStatusCond := expression.Name("current_eval_status").In(expression.Value("compiling"), expression.Value("received"), expression.Value("waiting")).And(
			expression.Name("current_eval_uuid").Equal(expression.Value(evalUuid)))
		updSubmCurrEvalStatusExpr, err := expression.NewBuilder().WithUpdate(updSubmCurrEvalStatus).WithCondition(updSubmCurrEvalStatusCond).Build()
		if err != nil {
			log.Printf("failed to build current eval status update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key: map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "subm#details"},
			},
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updSubmCurrEvalStatusExpr.Update(),
			ConditionExpression:       updSubmCurrEvalStatusExpr.Condition(),
			ExpressionAttributeValues: updSubmCurrEvalStatusExpr.Values(),
			ExpressionAttributeNames:  updSubmCurrEvalStatusExpr.Names(),
		})

		if err != nil {
			var condFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condFailed) {
				log.Printf("failed to update current eval status because the condition failed: %v", err)
			} else {
				log.Printf("failed to update current eval status: %v", err)
				return
			}
		}

		s.updateSubmStateChan <- &SubmissionStateUpdate{
			SubmUuid: submUuid,
			EvalUuid: evalUuid,
			NewState: "testing",
		}
	case "started_test":
		var parsed struct {
			TestId int64 `json:"test_id"`
		}
		err := json.Unmarshal(*fields, &parsed)
		if err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}
		// sort_key="eval#<eval_uuid>#test#0001" subm_uuid=<subm_uuid>
		testPk := map[string]types.AttributeValue{
			"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
			"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#test#" + fmt.Sprintf("%04d", parsed.TestId)},
		}
		// update evaluation test row by setting attribute "reached" to true
		updReached := expression.Set(expression.Name("reached"), expression.Value(true))
		updReachedExpr, err := expression.NewBuilder().WithUpdate(updReached).Build()
		if err != nil {
			log.Printf("failed to build reached update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key:                       testPk,
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updReachedExpr.Update(),
			ExpressionAttributeValues: updReachedExpr.Values(),
			ExpressionAttributeNames:  updReachedExpr.Names(),
		})

		if err != nil {
			log.Printf("failed to update reached: %v", err)
			return
		}
	case "ignored_test":
		var parsed struct {
			TestId int64 `json:"test_id"`
		}
		err := json.Unmarshal(*fields, &parsed)
		if err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}
		// sort_key="eval#<eval_uuid>#test#0001" subm_uuid=<subm_uuid>
		testPk := map[string]types.AttributeValue{
			"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
			"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#test#" + fmt.Sprintf("%04d", parsed.TestId)},
		}
		// update evaluation test row by setting attribute "ignored" to true
		updIgnored := expression.Set(expression.Name("ignored"), expression.Value(true))
		updIgnoredExpr, err := expression.NewBuilder().WithUpdate(updIgnored).Build()
		if err != nil {
			log.Printf("failed to build ignored update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key:                       testPk,
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updIgnoredExpr.Update(),
			ExpressionAttributeValues: updIgnoredExpr.Values(),
			ExpressionAttributeNames:  updIgnoredExpr.Names(),
		})

		if err != nil {
			log.Printf("failed to update ignored: %v", err)
			return
		}
	case "finished_test":
		var parsed struct {
			TestId     int64 `json:"test_id"`
			Submission struct {
				Stdout          *string `json:"stdout"`
				Stderr          *string `json:"stderr"`
				ExitCode        int64   `json:"exit_code"`
				CpuTimeMillis   int64   `json:"cpu_time_millis"`
				WallTimeMillis  int64   `json:"wall_time_millis"`
				MemoryKibiBytes int64   `json:"memory_kibibytes"`
			} `json:"submission"`
			Checker struct {
				Stdout          *string `json:"stdout"`
				Stderr          *string `json:"stderr"`
				ExitCode        int64   `json:"exit_code"`
				CpuTimeMillis   int64   `json:"cpu_time_millis"`
				WallTimeMillis  int64   `json:"wall_time_millis"`
				MemoryKibiBytes int64   `json:"memory_kibibytes"`
			} `json:"checker"`
		}
		err := json.Unmarshal(*fields, &parsed)
		if err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}

		// sort_key="eval#<eval_uuid>#test#0001" subm_uuid=<subm_uuid>
		testPk := map[string]types.AttributeValue{
			"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
			"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#test#" + fmt.Sprintf("%04d", parsed.TestId)},
		}

		updTest := expression.
			Set(expression.Name("subm_stdout"), expression.Value(parsed.Submission.Stdout)).
			Set(expression.Name("subm_stderr"), expression.Value(parsed.Submission.Stderr)).
			Set(expression.Name("subm_exit_code"), expression.Value(parsed.Submission.ExitCode)).
			Set(expression.Name("subm_cpu_time_millis"), expression.Value(parsed.Submission.CpuTimeMillis)).
			Set(expression.Name("subm_wall_time_millis"), expression.Value(parsed.Submission.WallTimeMillis)).
			Set(expression.Name("subm_memory_kibi_bytes"), expression.Value(parsed.Submission.MemoryKibiBytes)).
			Set(expression.Name("checker_stdout"), expression.Value(parsed.Checker.Stdout)).
			Set(expression.Name("checker_stderr"), expression.Value(parsed.Checker.Stderr)).
			Set(expression.Name("checker_exit_code"), expression.Value(parsed.Checker.ExitCode)).
			Set(expression.Name("checker_cpu_time_millis"), expression.Value(parsed.Checker.CpuTimeMillis)).
			Set(expression.Name("checker_wall_time_millis"), expression.Value(parsed.Checker.WallTimeMillis)).
			Set(expression.Name("checker_memory_kibi_bytes"), expression.Value(parsed.Checker.MemoryKibiBytes)).
			Set(expression.Name("finished"), expression.Value(true))

		updTestCond := expression.Name("finished").Equal(expression.Value(false))
		updTestExpr, err := expression.NewBuilder().WithUpdate(updTest).WithCondition(updTestCond).Build()
		if err != nil {
			log.Printf("failed to build test update expression: %v", err)
			return
		}

		_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			Key:                       testPk,
			TableName:                 aws.String(s.submTableName),
			UpdateExpression:          updTestExpr.Update(),
			ConditionExpression:       updTestExpr.Condition(),
			ExpressionAttributeValues: updTestExpr.Values(),
			ExpressionAttributeNames:  updTestExpr.Names(),
		})

		if err != nil {
			log.Printf("failed to update test: %v", err)
			return
		}

		// determine whether the test was accepted or not. change the tests, testgroup, or subtask row accordingly

		/*
			EXCERPT FROM THE OLD TESTER:
			if checkerRunData.Output.ExitCode == 0 {
				gath.FinishTestWithVerdictAccepted(int64(test.ID))
				gath.IncrementScore(1)
			} else if checkerRunData.Output.ExitCode == 1 ||
				checkerRunData.Output.ExitCode == 2 {
				gath.FinishTestWithVerdictWrongAnswer(int64(test.ID))
			} else {
				gath.FinishWithInternalServerError(fmt.Errorf("checker failed to run: %v",
					checkerRunData))
				return err
			}
		*/

		// get the test row to see subtasks and testgroups
		// pk=subm_uuid, sk=eval#<eval_uuid>#test#0001, get "subtasks" (list) and "test_group" (int pointer)
		o, err := s.ddbClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
			Key: map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#test#" + fmt.Sprintf("%04d", parsed.TestId)},
			},
			TableName:            aws.String(s.submTableName),
			ProjectionExpression: aws.String("subtasks, test_group"),
		})
		if err != nil {
			log.Printf("failed to get test row: %v", err)
			return
		}

		var testSubtasksTestGroups struct {
			Subtasks  []int `dynamodbav:"subtasks"`
			TestGroup *int  `dynamodbav:"test_group"`
		}

		err = attributevalue.UnmarshalMap(o.Item, &testSubtasksTestGroups)
		if err != nil {
			log.Printf("failed to unmarshal test row: %v", err)
			return
		}

		// if checker output has exit code non-zero, that isn't 1 or 2
		if parsed.Checker.ExitCode != 0 && parsed.Checker.ExitCode != 1 && parsed.Checker.ExitCode != 2 {
			log.Printf("checker failed to run: %v", parsed.Checker)
			// mark the evaluation as failed, submission as failed with the current eval_uuid = eval_uuid

			evalRowPk := map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#details"},
			}

			updEvalFailed := expression.Set(expression.Name("evaluation_stage"), expression.Value("error")).
				Set(expression.Name("error_msg"), expression.Value(fmt.Sprintf("checker exited with code %d for test %d", parsed.Checker.ExitCode, parsed.TestId)))
			updEvalFailedExpr, err := expression.NewBuilder().WithUpdate(updEvalFailed).Build()
			if err != nil {
				log.Printf("failed to build evaluation failed update expression: %v", err)
				return
			}

			_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
				Key:                       evalRowPk,
				TableName:                 aws.String(s.submTableName),
				UpdateExpression:          updEvalFailedExpr.Update(),
				ExpressionAttributeValues: updEvalFailedExpr.Values(),
				ExpressionAttributeNames:  updEvalFailedExpr.Names(),
			})

			if err != nil {
				log.Printf("failed to update evaluation failed: %v", err)
				return
			}

			submRowPk := map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "subm#details"},
			}

			updSubmFailed := expression.Set(expression.Name("current_eval_status"), expression.Value("error")).
				Set(expression.Name("error_msg"), expression.Value(fmt.Sprintf("checker exited with code %d for test %d", parsed.Checker.ExitCode, parsed.TestId)))
			updSubmFailedExpr, err := expression.NewBuilder().WithUpdate(updSubmFailed).Build()
			if err != nil {
				log.Printf("failed to build submission failed update expression: %v", err)
				return
			}

			_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
				Key:                       submRowPk,
				TableName:                 aws.String(s.submTableName),
				UpdateExpression:          updSubmFailedExpr.Update(),
				ExpressionAttributeValues: updSubmFailedExpr.Values(),
				ExpressionAttributeNames:  updSubmFailedExpr.Names(),
			})

			if err != nil {
				log.Printf("failed to update submission failed: %v", err)
				return
			}

			s.updateSubmStateChan <- &SubmissionStateUpdate{
				SubmUuid: submUuid,
				EvalUuid: evalUuid,
				NewState: "error",
			}

			return
		}

		// if this test has a testgroup, update testgroup row
		if testSubtasksTestGroups.TestGroup != nil {
			// find pk=<subm_uuid>, sk=eval#<eval_uuid>#scoring#testgroup#<testgroup_id>
			// read "accepted_tests", "wrong_tests", "untested_tests", "version"
			// increment "accepted_tests" by 1, decrement "untested_tests" by 1
			// increment "version" by 1, save the new version
			// on the condition that the current version is the same as the read version
			// otherwise, retry the whole process for a maximum of 30 times with random sleep between 10ms and 100ms

			testgroupPk := map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#scoring#testgroup#" + fmt.Sprintf("%02d", *testSubtasksTestGroups.TestGroup)},
			}
			for attempt := 0; attempt < 300; attempt++ { // about 15 seconds
				if attempt > 0 {
					time.Sleep(time.Duration(10+rand.Intn(91)) * time.Millisecond)
				}
				// let's read the testgroup row
				o, err := s.ddbClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
					Key:                  testgroupPk,
					TableName:            aws.String(s.submTableName),
					ProjectionExpression: aws.String("accepted_tests, wrong_tests, untested_tests, version"),
				})
				if err != nil {
					log.Printf("failed to get testgroup row: %v", err)
					continue
				}

				var testgroupRow struct {
					AcceptedTests int   `dynamodbav:"accepted_tests"`
					WrongTests    int   `dynamodbav:"wrong_tests"`
					UntestedTests int   `dynamodbav:"untested_tests"`
					Version       int64 `dynamodbav:"version"`
				}

				err = attributevalue.UnmarshalMap(o.Item, &testgroupRow)
				if err != nil {
					log.Printf("failed to unmarshal testgroup row: %v", err)
					continue
				}

				if parsed.Checker.ExitCode == 0 { // accepted
					testgroupRow.AcceptedTests++
					testgroupRow.UntestedTests--
				} else {
					testgroupRow.WrongTests++
					testgroupRow.UntestedTests--
				}

				updTestgroup := expression.
					Set(expression.Name("accepted_tests"), expression.Value(testgroupRow.AcceptedTests)).
					Set(expression.Name("untested_tests"), expression.Value(testgroupRow.UntestedTests)).
					Set(expression.Name("wrong_tests"), expression.Value(testgroupRow.WrongTests)).
					Set(expression.Name("version"), expression.Value(testgroupRow.Version+1))

				updTestgroupCond := expression.Name("version").Equal(expression.Value(testgroupRow.Version))
				updTestgroupExpr, err := expression.NewBuilder().WithUpdate(updTestgroup).WithCondition(updTestgroupCond).Build()
				if err != nil {
					log.Printf("failed to build testgroup update expression: %v", err)
					continue
				}

				_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
					Key:                       testgroupPk,
					TableName:                 aws.String(s.submTableName),
					UpdateExpression:          updTestgroupExpr.Update(),
					ConditionExpression:       updTestgroupExpr.Condition(),
					ExpressionAttributeValues: updTestgroupExpr.Values(),
					ExpressionAttributeNames:  updTestgroupExpr.Names(),
				})

				if err != nil {
					var condFailed *types.ConditionalCheckFailedException
					if errors.As(err, &condFailed) {
						log.Printf("failed to update testgroup because the condition failed: %v", err)
						continue
					} else {
						log.Printf("failed to update testgroup: %v", err)
						continue
					}
				}

				// afterwards update submission row with the new version if the current version is the smaller than testgroup row's written version
				// otherwise, stop because it is larger and the submission row was updated by another thread
				// also make sure that the current_eval_uuid is the same as eval_uuid

				// pk=subm_uuid, sk=subm#scoring#testgroup#<testgroup_id>
				submTestgroupPk := map[string]types.AttributeValue{
					"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
					"sort_key":  &types.AttributeValueMemberS{Value: "subm#scoring#testgroup#" + fmt.Sprintf("%02d", *testSubtasksTestGroups.TestGroup)},
				}

				updSubmTestgroup := expression.
					Set(expression.Name("accepted_tests"), expression.Value(testgroupRow.AcceptedTests)).
					Set(expression.Name("untested_tests"), expression.Value(testgroupRow.UntestedTests)).
					Set(expression.Name("wrong_tests"), expression.Value(testgroupRow.WrongTests)).
					Set(expression.Name("version"), expression.Value(testgroupRow.Version+1))

				updSubmTestgroupCond := expression.Name("version").LessThanEqual(expression.Value(testgroupRow.Version)).
					And(expression.Name("current_eval_uuid").Equal(expression.Value(evalUuid)))
				updSubmTestgroupExpr, err := expression.NewBuilder().WithUpdate(updSubmTestgroup).WithCondition(updSubmTestgroupCond).Build()
				if err != nil {
					log.Printf("failed to build submission testgroup update expression: %v", err)
					return
				}

				_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
					Key:                       submTestgroupPk,
					TableName:                 aws.String(s.submTableName),
					UpdateExpression:          updSubmTestgroupExpr.Update(),
					ConditionExpression:       updSubmTestgroupExpr.Condition(),
					ExpressionAttributeValues: updSubmTestgroupExpr.Values(),
					ExpressionAttributeNames:  updSubmTestgroupExpr.Names(),
				})

				if err != nil {
					var condFailed *types.ConditionalCheckFailedException
					if errors.As(err, &condFailed) {
						log.Printf("failed to update submission testgroup because the condition failed: %v", err)
					} else {
						log.Printf("failed to update submission testgroup: %v", err)
						return
					}
				} else {
					s.updateTestgroupResChan <- &TestgroupResultUpdate{
						SubmUuid:      submUuid,
						EvalUuid:      evalUuid,
						TestgroupId:   *testSubtasksTestGroups.TestGroup,
						AcceptedTests: testgroupRow.AcceptedTests,
						WrongTests:    testgroupRow.WrongTests,
						UntestedTests: testgroupRow.UntestedTests,
					}
				}

				break
			}
		} else if len(testSubtasksTestGroups.Subtasks) > 0 {
			// if this test has subtasks, update subtask rows
			// find pk=<subm_uuid>, sk=eval#<eval_uuid>#scoring#subtask#<subtask_id>
			// read "accepted_tests", "wrong_tests", "untested_tests", "version"
			// increment "accepted_tests" by 1, decrement "untested_tests" by 1
			// increment "version" by 1, save the new version
			// on the condition that the current version is the same as the read version
			// otherwise, retry the whole process for a maximum of 30 times with random sleep between 10ms and 100ms

			for _, subtaskId := range testSubtasksTestGroups.Subtasks {
				subtaskPk := map[string]types.AttributeValue{
					"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
					"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#scoring#subtask#" + fmt.Sprintf("%02d", subtaskId)},
				}
				for attempt := 0; attempt < 30; attempt++ {
					if attempt > 0 {
						time.Sleep(time.Duration(10+rand.Intn(91)) * time.Millisecond)
					}
					// let's read the subtask row
					o, err := s.ddbClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
						Key:                  subtaskPk,
						TableName:            aws.String(s.submTableName),
						ProjectionExpression: aws.String("accepted_tests, wrong_tests, untested_tests, version"),
					})
					if err != nil {
						log.Printf("failed to get subtask row: %v", err)
						continue
					}

					var subtaskRow struct {
						AcceptedTests int   `dynamodbav:"accepted_tests"`
						WrongTests    int   `dynamodbav:"wrong_tests"`
						UntestedTests int   `dynamodbav:"untested_tests"`
						Version       int64 `dynamodbav:"version"`
					}

					err = attributevalue.UnmarshalMap(o.Item, &subtaskRow)
					if err != nil {
						log.Printf("failed to unmarshal subtask row: %v", err)
					}
				}
				panic("not implemented")
			}
		} else {
			panic("not implemented")
		}

	case "finished_testing":
		// there's just nothing to do here
	case "finished_evaluation":
		var parsed struct {
			ErrorMsg *string `json:"error"`
		}
		err := json.Unmarshal(*fields, &parsed)
		if err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}

		// if ErrorMsg is not nil, set evaluation_stage to "error" and current_eval_status to "error"
		if parsed.ErrorMsg != nil {
			evalDetailsPk := map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#details"},
			}

			updEvalFailed := expression.Set(expression.Name("evaluation_stage"), expression.Value("failed")).
				Set(expression.Name("error_msg"), expression.Value(parsed.ErrorMsg))
			updEvalFailedExpr, err := expression.NewBuilder().WithUpdate(updEvalFailed).Build()
			if err != nil {
				log.Printf("failed to build evaluation failed update expression: %v", err)
				return
			}

			_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
				Key:                       evalDetailsPk,
				TableName:                 aws.String(s.submTableName),
				UpdateExpression:          updEvalFailedExpr.Update(),
				ExpressionAttributeValues: updEvalFailedExpr.Values(),
				ExpressionAttributeNames:  updEvalFailedExpr.Names(),
			})

			if err != nil {
				log.Printf("failed to update evaluation failed: %v", err)
				return
			}

			submDetailsPk := map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "subm#details"},
			}

			updSubmFailed := expression.Set(expression.Name("current_eval_status"), expression.Value("error")).
				Set(expression.Name("error_msg"), expression.Value(parsed.ErrorMsg))

			// make sure that the current_eval_uuid is the same as eval_uuid
			updSubmFailedCond := expression.Name("current_eval_uuid").Equal(expression.Value(evalUuid))
			updSubmFailedExpr, err := expression.NewBuilder().WithUpdate(updSubmFailed).WithCondition(updSubmFailedCond).Build()
			if err != nil {
				log.Printf("failed to build submission failed update expression: %v", err)
				return
			}

			_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
				Key:                       submDetailsPk,
				TableName:                 aws.String(s.submTableName),
				UpdateExpression:          updSubmFailedExpr.Update(),
				ExpressionAttributeValues: updSubmFailedExpr.Values(),
				ExpressionAttributeNames:  updSubmFailedExpr.Names(),
				ConditionExpression:       updSubmFailedExpr.Condition(),
			})

			if err != nil {
				log.Printf("failed to update submission failed: %v", err)
				return
			}

			s.updateSubmStateChan <- &SubmissionStateUpdate{
				SubmUuid: submUuid,
				EvalUuid: evalUuid,
				NewState: "error",
			}
		} else {
			// set both evaluation_stage and current_eval_status to "finished"
			evalDetailsPk := map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "eval#" + evalUuid + "#details"},
			}

			updEvalFinished := expression.Set(expression.Name("evaluation_stage"), expression.Value("finished"))
			// if not error
			updEvalFinishedCond := expression.Name("evaluation_stage").NotEqual(expression.Value("error"))
			updEvalFinishedExpr, err := expression.NewBuilder().WithUpdate(updEvalFinished).WithCondition(updEvalFinishedCond).Build()
			if err != nil {
				log.Printf("failed to build evaluation finished update expression: %v", err)
				return
			}

			_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
				Key:                       evalDetailsPk,
				TableName:                 aws.String(s.submTableName),
				UpdateExpression:          updEvalFinishedExpr.Update(),
				ExpressionAttributeValues: updEvalFinishedExpr.Values(),
				ExpressionAttributeNames:  updEvalFinishedExpr.Names(),
				ConditionExpression:       updEvalFinishedExpr.Condition(),
			})

			if err != nil {
				log.Printf("failed to update evaluation finished: %v", err)
				return
			}

			submDetailsPk := map[string]types.AttributeValue{
				"subm_uuid": &types.AttributeValueMemberS{Value: submUuid},
				"sort_key":  &types.AttributeValueMemberS{Value: "subm#details"},
			}

			updSubmFinished := expression.Set(expression.Name("current_eval_status"), expression.Value("finished"))
			// if not error
			updSubmFinishedCond := expression.Name("current_eval_status").NotEqual(expression.Value("error"))
			updSubmFinishedExpr, err := expression.NewBuilder().WithUpdate(updSubmFinished).WithCondition(updSubmFinishedCond).Build()
			if err != nil {
				log.Printf("failed to build submission finished update expression: %v", err)
				return
			}

			_, err = s.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
				Key:                       submDetailsPk,
				TableName:                 aws.String(s.submTableName),
				UpdateExpression:          updSubmFinishedExpr.Update(),
				ExpressionAttributeValues: updSubmFinishedExpr.Values(),
				ExpressionAttributeNames:  updSubmFinishedExpr.Names(),
				ConditionExpression:       updSubmFinishedExpr.Condition(),
			})

			if err != nil {
				log.Printf("failed to update submission finished: %v", err)
				return
			}

			s.updateSubmStateChan <- &SubmissionStateUpdate{
				SubmUuid: submUuid,
				EvalUuid: evalUuid,
				NewState: "finished",
			}
		}
	}

	log.Printf("processed msg type %s for subm_uuid %s, eval_uuid %s", msgType, submUuid, evalUuid)
}
