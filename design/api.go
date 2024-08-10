package design

import (
	"goa.design/goa/v3/dsl"
	cors "goa.design/plugins/v3/cors/dsl"
)

// API describes the global properties of the API server.
var _ = dsl.API("proglv", func() {
	dsl.Title("Programme.lv backend")
	dsl.Description("Service for managing users, tasks, and submissions.")

	cors.Origin("http://localhost:3000", func() {
		cors.Methods("GET", "POST", "PUT", "DELETE", "OPTIONS")
		cors.Headers("*")
		cors.Expose("*")
		cors.MaxAge(600)
		cors.Credentials()
	})
	cors.Origin("https://programme.lv", func() {
		cors.Methods("GET", "POST", "PUT", "DELETE", "OPTIONS")
		cors.Headers("*")
		cors.Expose("*")
		cors.MaxAge(600)
		cors.Credentials()
	})
	cors.Origin("https://www.programme.lv", func() {
		cors.Methods("GET", "POST", "PUT", "DELETE", "OPTIONS")
		cors.Headers("*")
		cors.Expose("*")
		cors.MaxAge(600)
		cors.Credentials()
	})
})

// JWTAuth defines a security scheme using JWT tokens.
var JWTAuth = dsl.JWTSecurity("jwt", func() {
	dsl.Scope("users:read", "Read users")
	dsl.Scope("users:write", "Write users")
})
