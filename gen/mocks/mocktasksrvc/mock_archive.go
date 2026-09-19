package mocktasksrvc

import (
	"context"

	srvc "github.com/programme-lv/backend/modules/task/srvc"
)

func (_m *MockTaskPgRepo) AddArchiveFile(ctx context.Context, taskId string, file srvc.ArchiveFile) error {
	ret := _m.Called(ctx, taskId, file)
	return ret.Error(0)
}

func (_m *MockTaskPgRepo) ClearArchiveObjectKey(ctx context.Context, taskId string) error {
	ret := _m.Called(ctx, taskId)
	return ret.Error(0)
}

func (_m *MockTaskPgRepo) DeleteArchiveFile(ctx context.Context, taskId, relPath string) error {
	ret := _m.Called(ctx, taskId, relPath)
	return ret.Error(0)
}

func (_m *MockTaskPgRepo) ListArchiveFiles(ctx context.Context, taskId string) ([]srvc.ArchiveFile, error) {
	ret := _m.Called(ctx, taskId)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]srvc.ArchiveFile), ret.Error(1)
}

func (_m *MockTaskPgRepo) ListLegacyArchiveZips(ctx context.Context) ([]srvc.LegacyArchiveZip, error) {
	ret := _m.Called(ctx)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]srvc.LegacyArchiveZip), ret.Error(1)
}
