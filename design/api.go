package design

import (
	"goa.design/goa/v3/dsl"
)

// API describes the global properties of the API server.
var _ = dsl.API("programme_lv", func() {
	dsl.Title("Programme.lv backend")
	dsl.Description("Service for managing users, tasks, and submissions.")

	dsl.Server("programme_lv", func() {
		dsl.Description("Server for Programme.lv Backend")
		dsl.Services("users")
		dsl.Host("development", func() {
			dsl.Description("Development host")
			dsl.URI("http://localhost:8080")
		})
		dsl.Host("production", func() {
			dsl.Description("Production host")
			dsl.URI("https://{version}.programme.lv")
			dsl.Variable("version", dsl.String, "API version", func() {
				dsl.Default("v1")
			})
		})
	})
})
