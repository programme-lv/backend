package conf

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/common/s3bucket"
)

const (
	awsRegion      = "eu-central-1"
	testfileBucket = "proglv-tests"
	testingBucket  = "proglv-testing"
	repoDirName    = "backend"
)

func FindProjectRoot() string {
	re := regexp.MustCompile(`^(.*` + repoDirName + `)`)
	cwd, _ := os.Getwd()
	rootPath := re.Find([]byte(cwd))
	return string(rootPath)
}

func init() {
	rootPath := FindProjectRoot()
	err := godotenv.Load(rootPath + `/.env`)
	if err != nil {
		slog.Error("error loading .env file",
			"error", err,
			"cwd", rootPath,
		)
		os.Exit(1)
	}
}

func MustGetCdnS3Bucket() *s3bucket.S3Bucket {
	cdnBucket := os.Getenv("CDN_BUCKET")
	if cdnBucket == "" {
		slog.Error("CDN_BUCKET env var is not set")
		os.Exit(1)
	}

	s3, err := s3bucket.NewS3Bucket(awsRegion, cdnBucket)
	if err != nil {
		slog.Error("failed to create public S3 bucket", "error", err)
		os.Exit(1)
	}
	return s3
}

// utilized for testing purposes
func MustGetTestingS3Bucket() *s3bucket.S3Bucket {
	s3, err := s3bucket.NewS3Bucket(awsRegion, testingBucket)
	if err != nil {
		slog.Error("failed to create development S3 bucket", "error", err)
		os.Exit(1)
	}
	return s3
}

func MustGetTestfileS3Bucket() *s3bucket.S3Bucket {
	s3, err := s3bucket.NewS3Bucket(awsRegion, testfileBucket)
	if err != nil {
		slog.Error("failed to create test S3 bucket", "error", err)
		os.Exit(1)
	}
	return s3
}

func MustGetJwtKeyFromEnv() []byte {
	return []byte(getRequiredEnv("JWT_KEY"))
}

func MustGetCookieDomainFromEnv() string {
	return getRequiredEnv("COOKIE_DOMAIN")
}

func MustGetPgxPoolFromEnv() *pgxpool.Pool {
	pgxPool, err := GetPgxPoolFromEnv()
	if err != nil {
		slog.Error("failed to create pg pool", "error", err)
		os.Exit(1)
	}
	return pgxPool
}

func GetPgxPoolFromEnv() (*pgxpool.Pool, error) {
	connStr := getPgConnStrFromEnv()
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection is working
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return pool, nil
}

func getPgConnStrFromEnv() string {
	host := os.Getenv("POSTGRES_HOST")
	var pw string
	if host == "localhost" {
		pw = os.Getenv("POSTGRES_PW")
	} else {
		secretName := getRequiredEnv("POSTGRES_PASSWORD_SECRET_NAME")
		secretValue, err := getSecretFromAWS(secretName)
		if err != nil {
			slog.Error("failed to get postgres password from AWS", "error", err)
			os.Exit(1)
		}

		var secret struct {
			Password string `json:"password"`
		}
		if err := json.Unmarshal([]byte(secretValue), &secret); err != nil {
			slog.Error("failed to parse postgres password secret", "error", err)
			os.Exit(1)
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

	svc := secretsmanager.NewFromConfig(cfg, func(opts *secretsmanager.Options) {
		opts.Region = awsRegion
	})

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

func getRequiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		slog.Error(fmt.Sprintf("%s env var is not set", name))
		os.Exit(1)
	}
	return value
}
