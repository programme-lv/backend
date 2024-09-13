package tasksrvc

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (ts *TaskService) queryItemsByPartitionKey(ctx context.Context, pkValue string) ([]map[string]types.AttributeValue, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:                &ts.taskTableName,
		KeyConditionExpression:   aws.String("#pk = :pkval"),
		ExpressionAttributeNames: map[string]string{"#pk": "pk"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pkval": &types.AttributeValueMemberS{Value: pkValue},
		},
		ProjectionExpression: aws.String("pk, sk"),
	}

	var items []map[string]types.AttributeValue
	paginator := dynamodb.NewQueryPaginator(ts.ddbClient, queryInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query items: %w", err)
		}
		items = append(items, page.Items...)
	}
	return items, nil
}
