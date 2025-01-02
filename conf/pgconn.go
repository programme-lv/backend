package conf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func GetPgConnStrFromEnv() string {
	host := os.Getenv("POSTGRES_HOST")
	var pw string
	if host == "localhost" {
		pw = os.Getenv("POSTGRES_PW")
	} else {
		secretName := os.Getenv("POSTGRES_PASSWORD_SECRET_NAME")
		secretValue, err := getSecretFromAWS(secretName)
		if err != nil {
			panic(fmt.Sprintf("failed to get postgres password from AWS: %v", err))
		}
		var secret struct {
			Password string `json:"password"`
		}
		if err := json.Unmarshal([]byte(secretValue), &secret); err != nil {
			panic(fmt.Sprintf("failed to parse postgres password secret: %v", err))
		}
		pw = secret.Password
	}
	user := os.Getenv("POSTGRES_USER")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")
	ssl := os.Getenv("POSTGRES_SSLMODE")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pw, db, ssl)
}

func getSecretFromAWS(secretName string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}
	svc := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := svc.GetSecretValue(ctx, input)
	if err != nil {
		return "", err
	}
	return *result.SecretString, nil
}
