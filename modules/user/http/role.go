package http

import (
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/user/auth"
)

// GetRole returns the role of the currently logged-in user
func (httpserver *UserHttpHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	// Get JWT claims from context, added by the JWT middleware
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)

	type RoleResponse struct {
		Role string `json:"role"`
	}

	// If no claims (not logged in) or claims retrieval failed, user is a guest
	if !ok || claims == nil {
		jsonresp.Success(w, RoleResponse{Role: "guest"})
		return
	}

	if claims.Username == "admin" {
		jsonresp.Success(w, RoleResponse{Role: "admin"})
		return
	}

	// Return the role from JWT claims
	jsonresp.Success(w, RoleResponse{Role: "user"})
}
