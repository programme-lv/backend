package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
)

// UpdatePreferences stores preferences for the specified account.
func (s *userSrvc) UpdatePreferences(ctx context.Context, userUUID uuid.UUID, hideTaskTags bool) srvcerror.E {
	var stored bool
	err := s.postgres.QueryRow(ctx, `UPDATE users SET hide_task_tags = $1 WHERE uuid = $2 RETURNING hide_task_tags`, hideTaskTags, userUUID).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		ctxlog.FromContext(ctx).Error("update user preferences", "error", err)
		return srvcerror.InternalServerError()
	}
	return nil
}
