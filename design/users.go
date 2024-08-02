package design

import (
	"goa.design/goa/v3/dsl"
)

// JWTAuth defines a security scheme using JWT tokens.
var JWTAuth = dsl.JWTSecurity("jwt", func() {
	dsl.Scope("api:access", "API access")
})

// User represents a user.
var User = dsl.Type("User", func() {
	dsl.Description("User representation")
	dsl.Attribute("uuid", dsl.String, "Unique user UUID", func() {
		dsl.Example("550e8400-e29b-41d4-a716-446655440000")
	})
	dsl.Attribute("username", dsl.String, "Username of the user", func() {
		dsl.Example("johndoe")
	})
	dsl.Attribute("email", dsl.String, "Email of the user", func() {
		dsl.Format(dsl.FormatEmail)
		dsl.Example("johndoe@example.com")
	})
	dsl.Attribute("firstname", dsl.String, "First name of the user", func() {
		dsl.Example("John")
	})
	dsl.Attribute("lastname", dsl.String, "Last name of the user", func() {
		dsl.Example("Doe")
	})
	dsl.Required("uuid", "username", "email", "firstname", "lastname")
})

// UserPayload represents the payload for creating and updating a user.
var UserPayload = dsl.Type("UserPayload", func() {
	dsl.Description("Payload for creating and updating a user")
	dsl.Attribute("username", dsl.String, "Username of the user", func() {
		dsl.Example("johndoe")
	})
	dsl.Attribute("email", dsl.String, "Email of the user", func() {
		dsl.Format(dsl.FormatEmail)
		dsl.Example("johndoe@example.com")
	})
	dsl.Attribute("firstname", dsl.String, "First name of the user", func() {
		dsl.Example("John")
	})
	dsl.Attribute("lastname", dsl.String, "Last name of the user", func() {
		dsl.Example("Doe")
	})
	dsl.Attribute("password", dsl.String, "Password of the user", func() {
		dsl.Example("password123")
	})
	dsl.Required("username", "email", "firstname", "lastname", "password")
})

// LoginPayload represents the payload for the login method.
var LoginPayload = dsl.Type("LoginPayload", func() {
	dsl.Description("Payload for user login")
	dsl.Attribute("email", dsl.String, "Email of the user", func() {
		dsl.Format(dsl.FormatEmail)
		dsl.Example("johndoe@example.com")
	})
	dsl.Attribute("password", dsl.String, "Password of the user", func() {
		dsl.Example("password123")
	})
	dsl.Required("email", "password")
})

var _ = dsl.Service("users", func() {
	dsl.Description("Service to manage users")

	dsl.Method("listUsers", func() {
		dsl.Description("List all users")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("api:access")
		})
		dsl.Result(dsl.ArrayOf(User))
		dsl.HTTP(func() {
			dsl.GET("/users")
			dsl.Response(dsl.StatusOK)
		})
		dsl.GRPC(func() {
			dsl.Response(dsl.CodeOK)
		})
	})

	dsl.Method("getUser", func() {
		dsl.Description("Get a user by UUID")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("api:access")
		})
		dsl.Payload(func() {
			dsl.Attribute("uuid", dsl.String, "UUID of the user", func() {
				dsl.Example("550e8400-e29b-41d4-a716-446655440000")
			})
			dsl.Required("uuid")
		})
		dsl.Result(User)
		dsl.HTTP(func() {
			dsl.GET("/users/{uuid}")
			dsl.Response(dsl.StatusOK)
			dsl.Response(dsl.StatusNotFound, func() {
				dsl.Description("User not found")
			})
		})
		dsl.GRPC(func() {
			dsl.Response(dsl.CodeOK)
			dsl.Response(dsl.CodeNotFound)
		})
	})

	dsl.Method("createUser", func() {
		dsl.Description("Create a new user")
		dsl.Payload(UserPayload)
		dsl.Result(User)
		dsl.HTTP(func() {
			dsl.POST("/users")
			dsl.Response(dsl.StatusCreated)
		})
		dsl.GRPC(func() {
			dsl.Response(dsl.CodeOK)
		})
	})

	dsl.Method("updateUser", func() {
		dsl.Description("Update an existing user")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("api:access")
		})
		dsl.Payload(func() {
			dsl.Attribute("uuid", dsl.String, "UUID of the user", func() {
				dsl.Example("550e8400-e29b-41d4-a716-446655440000")
			})
			dsl.Attribute("username", dsl.String, "Username of the user", func() {
				dsl.Example("johndoe")
			})
			dsl.Attribute("email", dsl.String, "Email of the user", func() {
				dsl.Format(dsl.FormatEmail)
				dsl.Example("johndoe@example.com")
			})
			dsl.Attribute("firstname", dsl.String, "First name of the user", func() {
				dsl.Example("John")
			})
			dsl.Attribute("lastname", dsl.String, "Last name of the user", func() {
				dsl.Example("Doe")
			})
			dsl.Attribute("password", dsl.String, "Password of the user", func() {
				dsl.Example("password123")
			})
			dsl.Required("uuid", "username", "email", "firstname", "lastname")
		})
		dsl.Result(User)
		dsl.HTTP(func() {
			dsl.PUT("/users/{uuid}")
			dsl.Response(dsl.StatusOK)
			dsl.Response(dsl.StatusNotFound, func() {
				dsl.Description("User not found")
			})
		})
		dsl.GRPC(func() {
			dsl.Response(dsl.CodeOK)
			dsl.Response(dsl.CodeNotFound)
		})
	})

	dsl.Method("deleteUser", func() {
		dsl.Description("Delete a user")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("api:access")
		})
		dsl.Payload(func() {
			dsl.Attribute("uuid", dsl.String, "UUID of the user", func() {
				dsl.Example("550e8400-e29b-41d4-a716-446655440000")
			})
			dsl.Required("uuid")
		})
		dsl.HTTP(func() {
			dsl.DELETE("/users/{uuid}")
			dsl.Response(dsl.StatusNoContent)
			dsl.Response(dsl.StatusNotFound, func() {
				dsl.Description("User not found")
			})
		})
		dsl.GRPC(func() {
			dsl.Response(dsl.CodeOK)
			dsl.Response(dsl.CodeNotFound)
		})
	})

	dsl.Method("login", func() {
		dsl.Description("User login")
		dsl.Payload(LoginPayload)
		dsl.Result(dsl.String, func() {
			dsl.Example("jwt_token")
		})
		dsl.HTTP(func() {
			dsl.POST("/auth/login")
			dsl.Response(dsl.StatusOK)
			dsl.Response(dsl.StatusUnauthorized, func() {
				dsl.Description("Invalid email or password")
			})
		})
		dsl.GRPC(func() {
			dsl.Response(dsl.CodeOK)
			dsl.Response(dsl.CodeUnauthenticated)
		})
	})

	dsl.Method("queryCurrentJWT", func() {
		dsl.Description("Query current JWT")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("api:access")
		})
		dsl.Result(dsl.String, func() {
			dsl.Example("current_jwt_token")
		})
		dsl.HTTP(func() {
			dsl.GET("/auth/current/jwt")
			dsl.Response(dsl.StatusOK)
		})
		dsl.GRPC(func() {
			dsl.Response(dsl.CodeOK)
		})
	})
})
