package design

import (
	"goa.design/goa/v3/dsl"
)

// API describes the global properties of the API server.
var _ = dsl.API("proglv", func() {
	dsl.Title("Programme.lv backend")
	dsl.Description("Service for managing users, tasks, and submissions.")
})
