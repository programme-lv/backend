package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/subm/submhttp"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	http1 "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/task/pgrepo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/task/tracing"
	"github.com/programme-lv/backend/usersrvc"
)

func main() {
	w := os.Stderr

	// set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))

	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		slog.Error("JWT_KEY is not set")
		os.Exit(1)
	}

	execSrvc := execsrvc.NewExecSrvc()
	userSrvc := usersrvc.NewUserService()

	pg, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}

	repo := pgrepo.NewTaskPgRepo(pg)

	// Create a traced task service
	var taskSrvc srvc.TaskSrvcClient
	var tracerProvider *tracing.TracerProvider

	// Check if Jaeger tracing is enabled
	enableTracing := os.Getenv("ENABLE_JAEGER_TRACING") == "true"

	if enableTracing {
		// Initialize with tracing
		slog.Info("Attempting to initialize Jaeger tracing (this will block with a timeout)")

		// Get the timeout from environment or use default
		timeoutStr := os.Getenv("JAEGER_CONNECTION_TIMEOUT")
		timeout := 10 * time.Second // Default timeout
		if timeoutStr != "" {
			if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
				timeout = parsedTimeout
				slog.Info("Using custom Jaeger connection timeout", "timeout", timeout)
			} else {
				slog.Warn("Invalid JAEGER_CONNECTION_TIMEOUT format, using default", "default", timeout)
			}
		}

		// Create a context with timeout for the entire initialization process
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Create a channel to receive the result
		resultCh := make(chan struct {
			svc srvc.TaskSrvcClient
			tp  *tracing.TracerProvider
			err error
		}, 1)

		// Run the initialization in a goroutine
		go func() {
			svc, tp, err := tracing.NewTracedTaskService(repo)
			resultCh <- struct {
				svc srvc.TaskSrvcClient
				tp  *tracing.TracerProvider
				err error
			}{svc, tp, err}
		}()

		// Wait for the result or timeout
		select {
		case result := <-resultCh:
			if result.err != nil {
				slog.Error("Failed to create traced task service", "error", result.err)
				slog.Error("Exiting because Jaeger tracing is required but could not be initialized")
				os.Exit(1)
			} else {
				taskSrvc = result.svc
				tracerProvider = result.tp
				// Ensure tracer is shut down on exit
				defer tracing.ShutdownTracing(context.Background(), tracerProvider)
				slog.Info("Jaeger tracing enabled for task service")
			}
		case <-ctx.Done():
			slog.Error("Timeout waiting for Jaeger tracing initialization", "timeout", timeout)
			slog.Error("Exiting because Jaeger tracing is required but could not be initialized")
			os.Exit(1)
		}
	} else {
		// Initialize without tracing
		taskSrvc, err = srvc.NewTaskSrvc(repo)
		if err != nil {
			log.Fatalf("error creating task service: %v", err)
		}
		slog.Info("Jaeger tracing disabled for task service")
	}

	submHttpHandler := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)
	taskHttpHandler := http1.NewTaskHttpHandler(taskSrvc)

	// Add tracing middleware to HTTP handlers if enabled
	if enableTracing && tracerProvider != nil {
		// Create tracing middleware for HTTP handlers
		tracingMiddleware := tracing.NewTracingMiddleware("task-http")
		taskHttpHandler.UseMiddleware(tracingMiddleware.Middleware)
		slog.Info("Jaeger tracing middleware added to task HTTP handlers")
	}

	httpServer := http.NewHttpServer(submHttpHandler, taskHttpHandler, userSrvc, execSrvc, []byte(jwtKey))

	address := ":8080"
	slog.Info("starting server", "address", address)
	err = httpServer.Start(":8080")
	slog.Info("server stopped", "error", err)
}

func newSubmHttpHandler(userSrvc *usersrvc.UserSrvc, taskSrvc srvc.TaskSrvcClient, execSrvc *execsrvc.ExecSrvc) *submhttp.SubmHttpHandler {
	pool, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}

	submPgRepo := submpgrepo.NewPgSubmRepo(pool)
	evalPgRepo := submpgrepo.NewPgEvalRepo(pool)
	submSrvc := submsrvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)
	if err != nil {
		log.Fatalf("error creating submission service: %v", err)
	}

	submHttpServer := submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)

	return submHttpServer
}
