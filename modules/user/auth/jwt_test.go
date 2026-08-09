package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJWTAcceptsHS256(t *testing.T) {
	key := []byte("test-jwt-key")
	token, err := GenerateJWT("user", "user@example.com", uuid.New(), key, time.Hour)
	require.NoError(t, err)

	claims, err := ValidateJWT(token, key)

	require.NoError(t, err)
	assert.Equal(t, "user", claims.Username)
}

func TestValidateJWTRejectsOtherSigningMethods(t *testing.T) {
	key := []byte("test-jwt-key")
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, &JwtClaims{
		Username: "admin",
		UUID:     uuid.NewString(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tokenString, err := token.SignedString(key)
	require.NoError(t, err)

	_, err = ValidateJWT(tokenString, key)

	require.Error(t, err)
}
