package conf

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/logging"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/programme-lv/backend/common/filestore"
)

const (
	awsRegion   = "eu-central-1"
	repoDirName = "backend"
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
		slog.Info("no .env file loaded; using process environment",
			"error", err,
			"cwd", rootPath,
		)
	}
}

type slogLogger struct {
	logger *slog.Logger
}

func newSlogLogger(logger *slog.Logger) *slogLogger {
	return &slogLogger{
		logger: logger,
	}
}

func (l slogLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	switch classification {
	case logging.Warn: // up the severity beause warning should not happen
		l.logger.Error(format, v...)
	case logging.Debug: // same goes for debug messages
		l.logger.Info(format, v...)
	default:
		// should never happen
		panic(fmt.Sprintf("unknown classification: %s", classification))
	}
}

func MustGetSqsClientFromEnv(logger *slog.Logger) *sqs.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
		config.WithLogger(newSlogLogger(logger)),
	)
	if err != nil {
		panic(fmt.Errorf("unable to load SDK config, %v", err))
	}
	return sqs.NewFromConfig(cfg)
}

func MustGetNatsConnFromEnv(ctx context.Context) *nats.Conn {
	conn, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		panic(fmt.Errorf("unable to connect to nats: %v", err))
	}
	return conn
}

func MustGetJwtKeyFromEnv() []byte {
	return []byte(getRequiredEnv("JWT_KEY"))
}

func MustGetAdminAPIKeyFromEnv() []byte {
	return []byte(getRequiredEnv("ADMIN_API_KEY"))
}

func MustGetFileStorageRootFromEnv() string {
	root := getRequiredEnv("FILE_STORAGE_ROOT")
	info, err := os.Stat(root)
	if err != nil {
		slog.Error("stat FILE_STORAGE_ROOT", "error", err, "path", root)
		os.Exit(1)
	}
	if !info.IsDir() {
		slog.Error("FILE_STORAGE_ROOT is not a directory", "path", root)
		os.Exit(1)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		slog.Error("resolve FILE_STORAGE_ROOT", "error", err, "path", root)
		os.Exit(1)
	}
	return absRoot
}

func MustGetPublicAPIBaseURLFromEnv() string {
	baseURL := getRequiredEnv("API_PUBLIC_BASE_URL")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		slog.Error("API_PUBLIC_BASE_URL is invalid", "value", baseURL, "error", err)
		os.Exit(1)
	}
	return filestore.NormalizeBaseURL(baseURL)
}

func MustGetTestfileDownloadSigningKeyFromEnv() []byte {
	return []byte(getRequiredEnv("TESTFILE_DOWNLOAD_SIGNING_KEY"))
}

func MustGetFileStore(root string, subdir string) *filestore.Store {
	store, err := filestore.NewStore(filepath.Join(root, subdir))
	if err != nil {
		slog.Error("create file store", "error", err, "subdir", subdir)
		os.Exit(1)
	}
	return store
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

func MustRunPostgresMigrationsFromEnv() {
	migrationsPath := os.Getenv("POSTGRES_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "postgres/migrate"
	}

	m, err := migrate.New("file://"+migrationsPath, getPgMigrationURLFromEnv())
	if err != nil {
		slog.Error("create postgres migrator", "error", err, "path", migrationsPath)
		os.Exit(1)
	}
	defer m.Close()

	err = m.Up()
	switch err {
	case nil:
		slog.Info("postgres migrations applied")
	case migrate.ErrNoChange:
		slog.Info("postgres schema already up to date")
	default:
		slog.Error("run postgres migrations", "error", err)
		os.Exit(1)
	}
}

func getPgConnStrFromEnv() string {
	host := os.Getenv("POSTGRES_HOST")
	user := os.Getenv("POSTGRES_USER")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")
	ssl := os.Getenv("POSTGRES_SSLMODE")
	pw := getPostgresPasswordFromEnv()

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pw, db, ssl)
}

func getPgMigrationURLFromEnv() string {
	host := os.Getenv("POSTGRES_HOST")
	user := os.Getenv("POSTGRES_USER")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")
	ssl := os.Getenv("POSTGRES_SSLMODE")
	pw := getPostgresPasswordFromEnv()

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, pw),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   db,
	}
	query := u.Query()
	query.Set("sslmode", ssl)
	u.RawQuery = query.Encode()

	return u.String()
}

func getPostgresPasswordFromEnv() string {
	pw := os.Getenv("POSTGRES_PW")
	if pw == "" {
		secretName := getRequiredEnv("POSTGRES_PW_SECRET_NAME")
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

	return pw
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
