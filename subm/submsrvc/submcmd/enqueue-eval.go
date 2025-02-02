package submcmd

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	decorator "github.com/programme-lv/backend/srvccqs"
	"github.com/programme-lv/backend/subm"
)

type EnqueueEvalCmd decorator.CmdHandler[EnqueueEvalParams]

type EnqueueEvalParams struct {
	Eval     subm.Eval
	SrcCode  string
	PrLangId string
}

type EnqueueEvalCmdHandler struct {
	EnqueueExec     func(ctx context.Context, execUuid uuid.UUID, srcCode string, prLangId string, tests []execsrvc.TestFile, params execsrvc.TestingParams) error
	GetTestDownlUrl func(ctx context.Context, testFileSha256 string) (string, error)
}

func (h EnqueueEvalCmdHandler) Handle(ctx context.Context, p EnqueueEvalParams) error {
	return h.EnqueueExec(
		ctx,
		p.Eval.UUID,
		p.SrcCode,
		p.PrLangId,
		h.evalReqTests(ctx, p.Eval),
		execsrvc.TestingParams{
			CpuMs:      p.Eval.CpuLimMs,
			MemKiB:     p.Eval.MemLimKiB,
			Checker:    p.Eval.Checker,
			Interactor: p.Eval.Interactor,
		},
	)
}

func (h EnqueueEvalCmdHandler) evalReqTests(
	ctx context.Context,
	eval subm.Eval,
) []execsrvc.TestFile {
	evalReqTests := make([]execsrvc.TestFile, len(eval.Tests))
	for i, test := range eval.Tests {
		inputS3Url, err := h.GetTestDownlUrl(ctx, test.InpSha256)
		if err != nil {
			slog.Error("failed to get download URL for input", "error", err)
		}
		answerS3Url, err := h.GetTestDownlUrl(ctx, test.AnsSha256)
		if err != nil {
			slog.Error("failed to get download URL for answer", "error", err)
		}
		evalReqTests[i] = execsrvc.TestFile{
			InSha256:    &test.InpSha256,
			AnsSha256:   &test.AnsSha256,
			InDownlUrl:  &inputS3Url,
			AnsDownlUrl: &answerS3Url,
		}
	}
	return evalReqTests
}
