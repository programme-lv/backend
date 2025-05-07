package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/common/s3bucket"
	"github.com/programme-lv/backend/exec"
	"github.com/programme-lv/backend/http"
	submhttp "github.com/programme-lv/backend/subm/http"
	submpgrepo "github.com/programme-lv/backend/subm/pgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	taskhttp "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/task/repo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user"
	userhttp "github.com/programme-lv/backend/user/http"
)

const (
	awsRegion  = "eu-central-1"
	cdnBucket  = "proglv-public"
	testBucket = "proglv-tests"
	address    = ":8080"
)

func main() {
	setupLogger()
	loadEnvVars()

	// Add flag for SQS listening
	listenSQS := flag.Bool("listen-sqs", true, "Whether to listen to result SQS queue")
	flag.Parse()

	jwtKey := getRequiredEnv("JWT_KEY")
	cookieDomain := os.Getenv("COOKIE_DOMAIN")

	pgPool, err := getPgxPoolFromEnv()
	if err != nil {
		slog.Error("failed to create pg pool", "error", err)
		os.Exit(1)
	}

	cdnS3, testS3, err := initializeS3Buckets()
	if err != nil {
		slog.Error("failed to initialize S3 buckets", "error", err)
		os.Exit(1)
	}

	execSrvc := exec.NewExecSrvc()
	if *listenSQS {
		err = execSrvc.ListenToResultSQS()
		if err != nil {
			slog.Error("failed to listen to result SQS", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("listening to execution result SQS disabled")
	}
	userSrvc := user.NewUserService(pgPool)
	taskRepo := repo.NewTaskPgRepo(pgPool)

	taskSrvc, err := srvc.NewTaskSrvc(taskRepo, cdnS3, testS3)
	if err != nil {
		slog.Error("error creating task service", "error", err)
		os.Exit(1)
	}

	// Initialize HTTP handlers
	submHttpHandler := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)
	taskHttpHandler := taskhttp.NewTaskHttpHandler(taskSrvc)
	userHttpHandler := userhttp.NewUserHttpHandler(userSrvc, []byte(jwtKey), userhttp.WithCookieDomain(cookieDomain))

	// Start HTTP server
	httpServer := http.NewHttpServer(submHttpHandler, taskHttpHandler, userHttpHandler, execSrvc, []byte(jwtKey))

	slog.Info("starting server", "address", address)
	err = httpServer.Start(address)
	slog.Info("server stopped", "error", err)
}

func setupLogger() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
			AddSource:  true,
		}),
	))
}

func loadEnvVars() {
	if err := godotenv.Load(); err != nil {
		slog.Error("error loading .env file", "error", err)
		os.Exit(1)
	}
}

func getRequiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		slog.Error(fmt.Sprintf("%s env var is not set", name))
		os.Exit(1)
	}
	return value
}

func initializeS3Buckets() (*s3bucket.S3Bucket, *s3bucket.S3Bucket, error) {
	publicS3, err := s3bucket.NewS3Bucket(awsRegion, cdnBucket)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create public S3 bucket: %w", err)
	}

	testS3, err := s3bucket.NewS3Bucket(awsRegion, testBucket)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create test S3 bucket: %w", err)
	}

	return publicS3, testS3, nil
}

func newSubmHttpHandler(userSrvc *user.UserSrvc, taskSrvc srvc.TaskSrvcClient, execSrvc *exec.ExecSrvc) *submhttp.SubmHttpHandler {
	pool, err := pgxpool.New(context.Background(), getPgConnStrFromEnv())
	if err != nil {
		slog.Error("failed to create pg pool", "error", err)
		os.Exit(1)
	}

	submPgRepo := submpgrepo.NewPgSubmRepo(pool)
	evalPgRepo := submpgrepo.NewPgEvalRepo(pool)
	submSrvc := submsrvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)

	return submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)
}

func getPgxPoolFromEnv() (*pgxpool.Pool, error) {
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
