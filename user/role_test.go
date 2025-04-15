package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRoleHttp(t *testing.T) {
	h := setupUserHttpHandler(t)

	// Test 1: Guest role (no login)
	t.Run("Guest Role", func(t *testing.T) {
		role := getRole(t, h, "")
		assert.Equal(t, "guest", role)
	})

	// Test 2: Regular user role
	t.Run("User Role", func(t *testing.T) {
		token := registerAndLogin(t, h, "testuser")
		role := getRole(t, h, token)
		assert.Equal(t, "user", role)
	})

	// Test 3: Admin role
	t.Run("Admin Role", func(t *testing.T) {
		token := registerAndLogin(t, h, "admin")
		role := getRole(t, h, token)
		assert.Equal(t, "admin", role)
	})
}
