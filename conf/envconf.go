package conf

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/programme-lv/backend/common/filestore"
)

const repoDirName = "backend"

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

func MustGetCookieSecureFromEnv() bool {
	value := getRequiredEnv("COOKIE_SECURE")
	secure, err := strconv.ParseBool(value)
	if err != nil {
		slog.Error("COOKIE_SECURE must be true or false", "value", value)
		os.Exit(1)
	}
	return secure
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
	return getRequiredEnv("POSTGRES_PW")
}

func getRequiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		slog.Error(fmt.Sprintf("%s env var is not set", name))
		os.Exit(1)
	}
	return value
}

// EmailConfig holds SMTP + transactional-email rate/TTL settings.
type EmailConfig struct {
	Enabled           bool
	Host              string
	Port              int
	Username          string
	Password          string
	From              string
	FromName          string
	WebsiteBaseURL    string
	ResetTokenTTL     time.Duration
	VerifyTokenTTL    time.Duration
	PerUserCooldown   time.Duration
	GlobalHourlyLimit int
}

func MustGetEmailConfigFromEnv() EmailConfig {
	enabled := false
	if raw := os.Getenv("SMTP_ENABLED"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			slog.Error("SMTP_ENABLED must be true or false", "value", raw)
			os.Exit(1)
		}
		enabled = parsed
	}

	cfg := EmailConfig{
		Enabled:           enabled,
		Host:              os.Getenv("SMTP_HOST"),
		Username:          os.Getenv("SMTP_USERNAME"),
		Password:          os.Getenv("SMTP_PASSWORD"),
		From:              os.Getenv("SMTP_FROM"),
		FromName:          os.Getenv("SMTP_FROM_NAME"),
		ResetTokenTTL:     durationFromEnv("EMAIL_RESET_TOKEN_TTL", time.Hour),
		VerifyTokenTTL:    durationFromEnv("EMAIL_VERIFY_TOKEN_TTL", 24*time.Hour),
		PerUserCooldown:   durationFromEnv("EMAIL_PER_USER_COOLDOWN", 5*time.Minute),
		GlobalHourlyLimit: intFromEnv("EMAIL_GLOBAL_HOURLY_LIMIT", 60),
	}

	port := 587
	if raw := os.Getenv("SMTP_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			slog.Error("SMTP_PORT must be a positive integer", "value", raw)
			os.Exit(1)
		}
		port = parsed
	}
	cfg.Port = port

	if !enabled {
		return cfg
	}

	cfg.WebsiteBaseURL = strings.TrimRight(getRequiredEnv("WEBSITE_PUBLIC_BASE_URL"), "/")
	parsed, err := url.Parse(cfg.WebsiteBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		slog.Error("WEBSITE_PUBLIC_BASE_URL is invalid", "value", cfg.WebsiteBaseURL, "error", err)
		os.Exit(1)
	}

	if cfg.Host == "" {
		slog.Error("SMTP_HOST env var is not set")
		os.Exit(1)
	}
	if cfg.From == "" {
		slog.Error("SMTP_FROM env var is not set")
		os.Exit(1)
	}

	return cfg
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		slog.Error(name+" must be a positive duration (e.g. 1h, 5m)", "value", raw)
		os.Exit(1)
	}
	return parsed
}

func intFromEnv(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		slog.Error(name+" must be a positive integer", "value", raw)
		os.Exit(1)
	}
	return parsed
}
