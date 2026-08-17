package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/modules/exec"
	exechttp "github.com/programme-lv/backend/modules/exec/http"
	planghttp "github.com/programme-lv/backend/modules/plang/http"
	"github.com/programme-lv/backend/modules/subm/domain"
	submhttp "github.com/programme-lv/backend/modules/subm/http"
	submpgrepo "github.com/programme-lv/backend/modules/subm/pgrepo"
	"github.com/programme-lv/backend/modules/subm/srvc"
	taskhttp "github.com/programme-lv/backend/modules/task/http"
	"github.com/programme-lv/backend/modules/task/repo"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
	usersrvc "github.com/programme-lv/backend/modules/user"
	userhttp "github.com/programme-lv/backend/modules/user/http"
	"github.com/programme-lv/backend/modules/user/mail"
	"github.com/schollz/progressbar/v3"
)

const (
	address = ":8080"
)

func main() {
	setupLogger()

	listenResults := flag.Bool("listen-sqs", true, "Whether to listen for NATS execution results")
	flag.Parse()

	jwtKey := conf.MustGetJwtKeyFromEnv()
	adminAPIKey := conf.MustGetAdminAPIKeyFromEnv()
	cookieDomain := conf.MustGetCookieDomainFromEnv()
	cookieSecure := conf.MustGetCookieSecureFromEnv()

	pgPool := conf.MustGetPgxPoolFromEnv()
	conf.MustRunPostgresMigrationsFromEnv()

	storageRoot := conf.MustGetFileStorageRootFromEnv()
	slog.Debug("file storage root", "path", storageRoot)
	apiPublicBaseURL := conf.MustGetPublicAPIBaseURLFromEnv()
	testfileSigningKey := conf.MustGetTestfileDownloadSigningKeyFromEnv()
	publicStore := conf.MustGetFileStore(storageRoot, "public")
	testfileStore := conf.MustGetFileStore(storageRoot, "testfiles")
	execStore := conf.MustGetFileStore(storageRoot, "exec")

	execCtx := context.Background()
	execCtx = ctxlog.WithLogger(execCtx, slog.Default().With("module", "exec"))
	execRepo := exec.NewFileExecRepo(execCtx, execStore)
	natsConn := conf.MustGetNatsConnFromEnv(execCtx)
	execSrvc := exec.NewExecSrvc(execCtx, execRepo, natsConn, testfileStore)
	if *listenResults {
		err := execSrvc.StartPollingResultQueue(execCtx)
		if err != nil {
			slog.Error("listen for NATS execution results", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("NATS execution result listener disabled")
	}

	// Initialize user service
	emailCfg := conf.MustGetEmailConfigFromEnv()
	var userMailer mail.Mailer = mail.NewNoopMailer()
	if emailCfg.Enabled {
		userMailer = mail.NewSMTPMailer(mail.SMTPConfig{
			Host:     emailCfg.Host,
			Port:     emailCfg.Port,
			Username: emailCfg.Username,
			Password: emailCfg.Password,
			From:     emailCfg.From,
			FromName: emailCfg.FromName,
		})
	}
	userMailer = mail.NewRateLimitedMailer(userMailer, emailCfg.GlobalHourlyLimit)
	userSrvc := usersrvc.NewUserService(pgPool, userMailer, usersrvc.EmailFlowConfig{
		WebsiteBaseURL:  emailCfg.WebsiteBaseURL,
		ResetTokenTTL:   emailCfg.ResetTokenTTL,
		VerifyTokenTTL:  emailCfg.VerifyTokenTTL,
		PerUserCooldown: emailCfg.PerUserCooldown,
	})

	// Initialize task service
	taskRepo := repo.NewTaskPgRepo(pgPool)
	taskSrvc := tasksrvc.NewTaskSrvc(
		taskRepo,
		publicStore,
		testfileStore,
		tasksrvc.WithPublicAPIBaseURL(apiPublicBaseURL),
		tasksrvc.WithTestfileDownloadSigningKey(testfileSigningKey),
	)

	// Initialize HTTP handlers
	submHttpHandler := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)
	taskHttpHandler := taskhttp.NewTaskHttpHandler(
		taskSrvc,
		taskhttp.WithFileStores(publicStore, testfileStore, testfileSigningKey),
	)
	userHttpHandler := userhttp.NewUserHttpHandler(
		userSrvc,
		jwtKey,
		userhttp.WithCookieDomain(cookieDomain),
		userhttp.WithSecureCookie(cookieSecure),
	)
	execHttpHandler := exechttp.NewExecHttpHandler(execSrvc, adminAPIKey)
	plangHttpHandler := planghttp.NewPlangHttpHandler()

	// Start HTTP server
	httpServer := newHTTPServer(
		submHttpHandler,
		taskHttpHandler,
		userHttpHandler,
		execHttpHandler,
		plangHttpHandler,
		jwtKey,
		adminAPIKey,
		cookieSecure,
		userSrvc.PasswordChangedAt,
	)

	slog.Info("starting server", "address", address)
	err := httpServer.start(address)
	slog.Info("server stopped", "error", err)
}

func setupLogger() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      conf.MustGetLogLevelFromEnv(),
			TimeFormat: time.Kitchen,
			AddSource:  true,
		}),
	))
}

func newSubmHttpHandler(userSrvc usersrvc.UserService, taskSrvc tasksrvc.TaskService, execSrvc exec.CodeExecutionService) *submhttp.SubmHttpHandler {
	pgPool, err := conf.GetPgxPoolFromEnv()
	if err != nil {
		slog.Error("create pg pool", "error", err)
		os.Exit(1)
	}

	submPgRepo := submpgrepo.NewPgSubmRepo(pgPool)
	evalPgRepo := submpgrepo.NewPgEvalRepo(pgPool)
	submSrvc := srvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)

	// Check if migration is needed and run it
	runScoreMigrationIfNeeded(pgPool, submSrvc, evalPgRepo)

	return submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)
}

// runScoreMigrationIfNeeded checks if there are evaluations without score info
// and recalculates/stores them. This is a one-time migration.
func runScoreMigrationIfNeeded(pgPool *pgxpool.Pool, submSrvc srvc.SubmissionService, evalPgRepo srvc.EvalRepo) {
	ctx := context.Background()

	// Check if there are evaluations missing score info
	var count int
	err := pgPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM submissions s 
		LEFT OUTER JOIN evaluations e ON s.curr_eval_uuid = e.uuid 
		WHERE e.received_score IS NULL
	`).Scan(&count)
	if err != nil {
		slog.Error("check for missing scores", "error", err)
		return
	}

	if count == 0 {
		slog.Info("no evaluations missing score info, skipping migration")
		return
	}

	slog.Warn("found evaluations missing score info, running migration", "count", count)

	subms, err := submSrvc.ListSubms(ctx, srvc.ListSubmsParams{
		Limit:  10000000,
		Offset: 0,
		Search: "",
		Author: nil,
	})
	if err != nil {
		slog.Error("list submissions", "error", err)
		return
	}

	slog.Info("processing submissions", "total", len(subms))
	bar := progressbar.Default(int64(len(subms)), "recalculating scores")

	failed := 0
	for _, subm := range subms {
		bar.Add(1)

		if subm.CurrEvalUUID == uuid.Nil {
			reEvalErr := submSrvc.ReEvalSubm(ctx, subm.UUID)
			slog.Info("re-evaluating submission", "subm_uuid", subm.UUID)
			if reEvalErr != nil {
				slog.Error("re-evaluate submission", "error", reEvalErr, "subm_uuid", subm.UUID)
				failed++
				continue
			}

			subm, err = submSrvc.ViewSubm(ctx, subm.UUID)
			if err != nil {
				slog.Error("get submission after re-eval", "error", err, "subm_uuid", subm.UUID)
				failed++
				continue
			}

			evalFinished := false
			for {
				eval, err := submSrvc.GetEval(ctx, subm.CurrEvalUUID)
				if err != nil {
					slog.Error("get eval during re-eval wait", "error", err, "eval_uuid", subm.CurrEvalUUID)
					break
				}
				if eval.Stage == domain.EvalStageFinished {
					evalFinished = true
					break
				}
				time.Sleep(1 * time.Second)
			}
			if !evalFinished {
				failed++
				continue
			}
			slog.Info("eval finished", "eval_uuid", subm.CurrEvalUUID)
		}

		eval, err := submSrvc.GetEval(ctx, subm.CurrEvalUUID)
		if err != nil {
			slog.Error("get eval", "error", err, "eval_uuid", subm.CurrEvalUUID)
			failed++
			continue
		}

		storeEvalErr := evalPgRepo.StoreEval(ctx, eval)
		if storeEvalErr != nil {
			slog.Error("store eval", "error", storeEvalErr, "eval_uuid", eval.UUID)
			failed++
			continue
		}
	}

	if failed > 0 {
		slog.Error("score migration finished with failures", "failed", failed, "total", len(subms))
		return
	}
	slog.Info("score migration completed")
}
