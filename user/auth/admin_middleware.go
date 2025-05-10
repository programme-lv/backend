package auth

import (
	"net/http"

	"github.com/programme-lv/backend/common/httpjson"
)

// AdminOnly is a middleware that ensures only admin users can access the route
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(CtxJwtClaimsKey).(*JwtClaims)
		if !ok || claims == nil {
			httpjson.Unauthorized(w, "failed to authenticate user")
			return
		}
		if claims.Username != "admin" {
			httpjson.Forbidden(w, "user is not an admin")
			return
		}
		next.ServeHTTP(w, r)
	})
}
