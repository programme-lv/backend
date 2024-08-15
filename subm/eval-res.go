package subm

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (s *submissionssrvc) processEvalResult(evalUuid string, msgType string, fields *json.RawMessage) {
	baseDelay := 100 * time.Millisecond

	var submUuid string
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

		// TODO: notify listeners that the submission has been received
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

		//TODO: notify listeners that the compilation has started
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
		// TODO: implement
	case "started_test":
		//TODO: implement
	case "ignored_test":
	//TODO: implement
	case "finished_test":
		//TODO: implement
	case "finished_testing":
		//TODO: implement
	case "finished_evaluation":
		// TODO: implement
	}
}
