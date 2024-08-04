package design

import (
	"goa.design/goa/v3/dsl"
)

// JWTAuth defines a security scheme using JWT tokens.
var JWTAuth = dsl.JWTSecurity("jwt", func() {
	dsl.Scope("users:read", "Read users")
	dsl.Scope("users:write", "Write users")
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
		dsl.MinLength(1)
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
		dsl.MinLength(8)
		dsl.Example("password123")
	})
	dsl.Required("username", "email", "firstname", "lastname", "password")
})

// LoginPayload represents the payload for the login method.
var LoginPayload = dsl.Type("LoginPayload", func() {
	dsl.Description("Payload for user login")
	dsl.Attribute("username", dsl.String, "Username of the user", func() {
		dsl.Example("johndoe")
	})
	dsl.Attribute("password", dsl.String, "Password of the user", func() {
		dsl.MinLength(8)
		dsl.Example("password123")
	})
	dsl.Required("username", "password")
})

// SecureUUIDPayload defines a payload with a JWT token and UUID.
var SecureUUIDPayload = dsl.Type("SecureUUIDPayload", func() {
	dsl.Token("token", dsl.String, "JWT token used for authentication", func() {
		dsl.Example("jwt_token")
	})
	dsl.Attribute("uuid", dsl.String, "UUID of the user", func() {
		dsl.Example("550e8400-e29b-41d4-a716-446655440000")
	})
	dsl.Required("token", "uuid")
})

var JwtClaims = dsl.Type("JwtClaims", func() {
	dsl.Attribute("username", dsl.String)
	dsl.Attribute("firstname", dsl.String)
	dsl.Attribute("lastname", dsl.String)
	dsl.Attribute("email", dsl.String)
	dsl.Attribute("uuid", dsl.String)
	dsl.Attribute("scopes", dsl.ArrayOf(dsl.String))
	dsl.Attribute("issuer", dsl.String)
	dsl.Attribute("subject", dsl.String)
	dsl.Attribute("audience", dsl.ArrayOf(dsl.String))
	dsl.Attribute("expires_at", dsl.String)
	dsl.Attribute("issued_at", dsl.String)
	dsl.Attribute("not_before", dsl.String)
})

var _ = dsl.Service("users", func() {
	dsl.Description("Service to manage users")

	dsl.Error("unauthorized", dsl.String, "Credentials are invalid")
	dsl.Error("InvalidCredentials", dsl.String, "Invalid credentials")
	dsl.Error("InvalidUserDetails", dsl.String, "Invalid user details")
	dsl.Error("NotFound", dsl.String, "User not found")
	dsl.Error("UsernameExists", dsl.String, "Username already exists")
	dsl.Error("EmailExists", dsl.String, "Email already exists")

	dsl.HTTP(func() {
		dsl.Response("InvalidCredentials", dsl.StatusUnauthorized)
		dsl.Response("InvalidUserDetails", dsl.StatusBadRequest)
		dsl.Response("NotFound", dsl.StatusNotFound)
		dsl.Response("UsernameExists", dsl.StatusConflict)
		dsl.Response("EmailExists", dsl.StatusConflict)
	})

	dsl.Method("listUsers", func() {
		dsl.Description("List all users")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("users:read")
		})
		dsl.Payload(func() {
			dsl.Token("token", dsl.String, "JWT token used for authentication", func() {
				dsl.Example("jwt_token")
			})
		})
		dsl.Result(dsl.ArrayOf(User))
		dsl.HTTP(func() {
			dsl.GET("/users")
			dsl.Param("token:Authorization")
			dsl.Response(dsl.StatusOK)
		})
	})

	dsl.Method("getUser", func() {
		dsl.Description("Get a user by UUID")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("users:read")
		})
		dsl.Payload(SecureUUIDPayload)
		dsl.Result(User)
		dsl.HTTP(func() {
			dsl.GET("/users/{uuid}")
			dsl.Param("token:Authorization")
			dsl.Response(dsl.StatusOK)
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
		dsl.Error("InvalidUserDetails")
		dsl.Error("UsernameExists", dsl.String, "Username already exists")
		dsl.Error("EmailExists", dsl.String, "Email already exists")
	})

	dsl.Method("updateUser", func() {
		dsl.Description("Update an existing user")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("users:write")
		})
		dsl.Payload(func() {
			dsl.Token("token", dsl.String, "JWT token used for authentication", func() {
				dsl.Example("jwt_token")
			})
			dsl.Attribute("uuid", dsl.String, "UUID of the user", func() {
				dsl.Example("550e8400-e29b-41d4-a716-446655440000")
			})
			dsl.Attribute("username", dsl.String, "Username of the user", func() {
				dsl.MinLength(1)
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
				dsl.MinLength(8)
				dsl.Example("password123")
			})
			dsl.Required("token", "uuid", "username", "email", "firstname", "lastname")
		})
		dsl.Result(User)
		dsl.HTTP(func() {
			dsl.PUT("/users/{uuid}")
			dsl.Param("token:Authorization")
			dsl.Response(dsl.StatusOK)
			dsl.Response(dsl.StatusBadRequest)
		})
		dsl.Error("InvalidUserDetails")
	})

	dsl.Method("deleteUser", func() {
		dsl.Description("Delete a user")
		dsl.Security(JWTAuth, func() {
			dsl.Scope("users:write")
		})
		dsl.Payload(SecureUUIDPayload)
		dsl.HTTP(func() {
			dsl.DELETE("/users/{uuid}")
			dsl.Param("token:Authorization")
			dsl.Response(dsl.StatusNoContent)
		})
		dsl.Error("NotFound")
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
			dsl.Response(dsl.StatusUnauthorized)
			dsl.Response(dsl.StatusBadRequest)
		})
		dsl.Error("InvalidCredentials")
		dsl.Error("InvalidUserDetails")
	})

	dsl.Method("queryCurrentJWT", func() {
		dsl.Description("Query current JWT")
		dsl.Security(JWTAuth, func() {
		})
		dsl.Payload(func() {
			dsl.Token("token", dsl.String, "JWT token used for authentication")
			dsl.Required("token")
		})
		dsl.Result(dsl.String, func() {
			dsl.Example("current_jwt_token")
		})
		dsl.Result(JwtClaims)
		dsl.HTTP(func() {
			dsl.GET("/auth/current/jwt")
			dsl.Response(dsl.StatusOK)
		})
	})
})
