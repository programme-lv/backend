package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/exec"
	"github.com/stretchr/testify/assert"
)

type fakeExecService struct{}

func (fakeExecService) Enqueue(
	context.Context,
	uuid.UUID,
	string,
	string,
	[]exec.TestFile,
	exec.TestingParams,
) srvcerror.E {
	return nil
}

func (fakeExecService) Listen(context.Context, uuid.UUID) (<-chan exec.Event, srvcerror.E) {
	ch := make(chan exec.Event)
	close(ch)
	return ch, nil
}

func (fakeExecService) Get(_ context.Context, execUUID uuid.UUID) (exec.Execution, srvcerror.E) {
	return exec.Execution{UUID: execUUID}, nil
}

func TestExecRoutesRequireAdminAuthentication(t *testing.T) {
	handler := NewExecHttpHandler(fakeExecService{}, []byte("admin-api-key"))
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/tester/run"},
		{method: http.MethodGet, path: "/tester/run/" + uuid.NewString()},
		{method: http.MethodGet, path: "/exec/" + uuid.NewString()},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			assert.Equal(t, http.StatusUnauthorized, res.Code)
		})
	}
}

func TestExecGetAcceptsAdminAPIKey(t *testing.T) {
	handler := NewExecHttpHandler(fakeExecService{}, []byte("admin-api-key"))
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/exec/"+uuid.NewString(), nil)
	req.Header.Set("Authorization", "Bearer admin-api-key")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
}
