package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
)

func (s *UserSrvc) IsAdmin(ctx context.Context, userUuid uuid.UUID) (bool, error) {
	l := ctxlog.FromContext(ctx)
	user, err := s.GetUserByUUID(ctx, userUuid)
	if err != nil {
		l.Error("failed to get user by uuid", "error", err)
		return false, srvcerror.ErrInternalServerError()
	}

	return user.Username == "admin", nil
}
