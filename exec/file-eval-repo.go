package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/filestore"
)

type FileEvalRepo struct {
	logger *slog.Logger
	store  *filestore.Store
}

func NewFileExecRepo(ctx context.Context, store *filestore.Store) *FileEvalRepo {
	return &FileEvalRepo{
		logger: ctxlog.FromContext(ctx),
		store:  store,
	}
}

func (r *FileEvalRepo) Save(ctx context.Context, eval *Execution) error {
	data, err := json.Marshal(eval)
	if err != nil {
		return fmt.Errorf("marshal evaluation: %w", err)
	}

	key := fmt.Sprintf("%s.json", eval.UUID.String())
	r.logger.Info("saving eval to file store", "key", key)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if _, err := r.store.Upload(data, key, "application/json"); err != nil {
		r.logger.Error("store evaluation in file store", "error", err)
		return fmt.Errorf("store evaluation in file store: %w", err)
	}
	return nil
}

func (r *FileEvalRepo) Get(ctx context.Context, evalUuid uuid.UUID) (*Execution, error) {
	key := fmt.Sprintf("%s.json", evalUuid.String())
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := r.store.Download(key)
	if err != nil {
		return nil, fmt.Errorf("get evaluation from file store: %w", err)
	}

	var eval Execution
	if err := json.Unmarshal(data, &eval); err != nil {
		return nil, fmt.Errorf("unmarshal evaluation: %w", err)
	}

	return &eval, nil
}

var _ ExecRepo = &FileEvalRepo{}
