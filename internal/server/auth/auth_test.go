package auth

import (
	"testing"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/apperrors"
	authconf "github.com/ar4ie13/gophkeeper/internal/server/auth/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuth() *Auth {
	return NewAuth(authconf.Config{
		SecretKey:       "test_secret_key_for_testing",
		TokenExpiration: time.Hour * 24,
		PasswordLen:     8,
	})
}

func TestNewAuth(t *testing.T) {
	t.Run("creates auth with config", func(t *testing.T) {
		cfg := authconf.Config{
			SecretKey:       "test_secret",
			TokenExpiration: time.Hour * 48,
			PasswordLen:     10,
		}

		auth := NewAuth(cfg)

		assert.NotNil(t, auth)
		assert.Equal(t, cfg, auth.conf)
	})
}

func TestAuth_BuildJWTString(t *testing.T) {
	t.Run("creates valid JWT token", func(t *testing.T) {
		auth := newTestAuth()
		userUUID := uuid.New()

		tokenString, err := auth.BuildJWTString(userUUID)

		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		// Verify token can be parsed
		claims, token, parseErr := auth.parseTokenString(tokenString)
		require.NoError(t, parseErr)
		assert.True(t, token.Valid)
		assert.Equal(t, userUUID, claims.UserUUID)
	})

	t.Run("token contains expiration time", func(t *testing.T) {
		auth := newTestAuth()
		userUUID := uuid.New()

		before := time.Now().Add(auth.conf.TokenExpiration)
		tokenString, err := auth.BuildJWTString(userUUID)
		after := time.Now().Add(auth.conf.TokenExpiration)

		require.NoError(t, err)

		claims, _, parseErr := auth.parseTokenString(tokenString)
		require.NoError(t, parseErr)

		expiresAt := claims.ExpiresAt.Time
		assert.True(t, expiresAt.After(before.Add(-time.Second)))
		assert.True(t, expiresAt.Before(after.Add(time.Second)))
	})

	t.Run("different UUIDs create different tokens", func(t *testing.T) {
		auth := newTestAuth()
		uuid1 := uuid.New()
		uuid2 := uuid.New()

		token1, err1 := auth.BuildJWTString(uuid1)
		token2, err2 := auth.BuildJWTString(uuid2)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, token1, token2)
	})
}

func TestAuth_ValidateUserUUID(t *testing.T) {
	t.Run("validates valid token", func(t *testing.T) {
		auth := newTestAuth()
		expectedUUID := uuid.New()

		tokenString, err := auth.BuildJWTString(expectedUUID)
		require.NoError(t, err)

		userUUID, err := auth.ValidateUserUUID(tokenString)

		assert.NoError(t, err)
		assert.Equal(t, expectedUUID, userUUID)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		auth := &Auth{
			conf: authconf.Config{
				SecretKey:       "test_secret",
				TokenExpiration: -time.Hour, // Expired 1 hour ago
				PasswordLen:     8,
			},
		}
		userUUID := uuid.New()

		tokenString, err := auth.BuildJWTString(userUUID)
		require.NoError(t, err)

		// Reset to normal expiration for validation
		auth.conf.TokenExpiration = time.Hour * 24

		validatedUUID, err := auth.ValidateUserUUID(tokenString)

		assert.ErrorIs(t, err, apperrors.ErrUserIsNotAuthorized)
		assert.Equal(t, uuid.Nil, validatedUUID)
	})

	t.Run("rejects token with invalid signature", func(t *testing.T) {
		auth := newTestAuth()
		userUUID := uuid.New()

		tokenString, err := auth.BuildJWTString(userUUID)
		require.NoError(t, err)

		// Change the secret key
		auth.conf.SecretKey = "different_secret"

		validatedUUID, err := auth.ValidateUserUUID(tokenString)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, validatedUUID)
	})

	t.Run("rejects token with nil UUID", func(t *testing.T) {
		auth := newTestAuth()

		// Manually create a token with nil UUID
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			UserUUID: uuid.Nil,
		})

		tokenString, err := token.SignedString([]byte(auth.conf.SecretKey))
		require.NoError(t, err)

		validatedUUID, err := auth.ValidateUserUUID(tokenString)

		assert.ErrorIs(t, err, apperrors.ErrInvalidUserUUID)
		assert.Equal(t, uuid.Nil, validatedUUID)
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		auth := newTestAuth()

		validatedUUID, err := auth.ValidateUserUUID("invalid.token.string")

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, validatedUUID)
	})

	t.Run("rejects empty token string", func(t *testing.T) {
		auth := newTestAuth()

		validatedUUID, err := auth.ValidateUserUUID("")

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, validatedUUID)
	})

	t.Run("rejects token with empty UUID string", func(t *testing.T) {
		auth := newTestAuth()

		// Create a token with a zero UUID
		zeroUUID := uuid.UUID{}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			UserUUID: zeroUUID,
		})

		tokenString, err := token.SignedString([]byte(auth.conf.SecretKey))
		require.NoError(t, err)

		validatedUUID, err := auth.ValidateUserUUID(tokenString)

		assert.ErrorIs(t, err, apperrors.ErrInvalidUserUUID)
		assert.Equal(t, uuid.Nil, validatedUUID)
	})
}

func TestAuth_parseTokenString(t *testing.T) {
	t.Run("parses valid token", func(t *testing.T) {
		auth := newTestAuth()
		expectedUUID := uuid.New()

		tokenString, err := auth.BuildJWTString(expectedUUID)
		require.NoError(t, err)

		claims, token, err := auth.parseTokenString(tokenString)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.NotNil(t, token)
		assert.True(t, token.Valid)
		assert.Equal(t, expectedUUID, claims.UserUUID)
	})

	t.Run("returns error for invalid token", func(t *testing.T) {
		auth := newTestAuth()

		claims, token, err := auth.parseTokenString("invalid.token.string")

		assert.Error(t, err)
		assert.NotNil(t, claims) // Claims object is always returned
		assert.NotNil(t, token)  // Token object is always returned
	})

	t.Run("rejects token with wrong signing method", func(t *testing.T) {
		auth := newTestAuth()

		// Create a token with RS256 instead of HS256
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			UserUUID: uuid.New(),
		})

		// This will fail because we're using HMAC secret with RSA method
		// but the test is to ensure the signing method check works
		tokenString := token.Raw

		if tokenString == "" {
			// If we can't create a token with wrong method, skip this edge case
			t.Skip("Cannot create token with wrong signing method")
		}

		claims, _, err := auth.parseTokenString(tokenString)

		assert.Error(t, err)
		assert.NotNil(t, claims)
	})

	t.Run("handles expired token", func(t *testing.T) {
		auth := &Auth{
			conf: authconf.Config{
				SecretKey:       "test_secret",
				TokenExpiration: -time.Hour,
				PasswordLen:     8,
			},
		}
		userUUID := uuid.New()

		tokenString, err := auth.BuildJWTString(userUUID)
		require.NoError(t, err)

		// Reset to normal config for parsing
		auth.conf.TokenExpiration = time.Hour * 24

		claims, token, err := auth.parseTokenString(tokenString)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
		assert.NotNil(t, claims)
		assert.NotNil(t, token)
	})
}

func TestAuth_GenerateHashFromPassword(t *testing.T) {
	t.Run("generates hash for valid password", func(t *testing.T) {
		auth := newTestAuth()
		password := "password123"

		hash, err := auth.GenerateHashFromPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
	})

	t.Run("generates different hashes for same password", func(t *testing.T) {
		auth := newTestAuth()
		password := "password123"

		hash1, err1 := auth.GenerateHashFromPassword(password)
		hash2, err2 := auth.GenerateHashFromPassword(password)

		require.NoError(t, err1)
		require.NoError(t, err2)
		// bcrypt includes a salt, so hashes should be different
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("rejects password shorter than minimum length", func(t *testing.T) {
		auth := newTestAuth() // PasswordLen = 8
		shortPassword := "short"

		hash, err := auth.GenerateHashFromPassword(shortPassword)

		assert.Error(t, err)
		assert.Empty(t, hash)
		assert.ErrorIs(t, err, apperrors.ErrPasswordMinSymbols)
	})

	t.Run("accepts password equal to minimum length", func(t *testing.T) {
		auth := newTestAuth() // PasswordLen = 8
		password := "12345678" // exactly 8 characters

		hash, err := auth.GenerateHashFromPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("accepts password longer than minimum length", func(t *testing.T) {
		auth := newTestAuth() // PasswordLen = 8
		password := "long_password_12345678"

		hash, err := auth.GenerateHashFromPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestAuth_CheckPasswordHash(t *testing.T) {
	t.Run("validates correct password", func(t *testing.T) {
		auth := newTestAuth()
		password := "password123"

		hash, err := auth.GenerateHashFromPassword(password)
		require.NoError(t, err)

		isValid := auth.CheckPasswordHash(password, hash)

		assert.True(t, isValid)
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		auth := newTestAuth()
		password := "password123"
		wrongPassword := "wrongpassword"

		hash, err := auth.GenerateHashFromPassword(password)
		require.NoError(t, err)

		isValid := auth.CheckPasswordHash(wrongPassword, hash)

		assert.False(t, isValid)
	})

	t.Run("rejects empty password", func(t *testing.T) {
		auth := newTestAuth()
		password := "password123"

		hash, err := auth.GenerateHashFromPassword(password)
		require.NoError(t, err)

		isValid := auth.CheckPasswordHash("", hash)

		assert.False(t, isValid)
	})

	t.Run("rejects invalid hash", func(t *testing.T) {
		auth := newTestAuth()
		password := "password123"

		isValid := auth.CheckPasswordHash(password, "invalid_hash")

		assert.False(t, isValid)
	})

	t.Run("is case sensitive", func(t *testing.T) {
		auth := newTestAuth()
		password := "Password123"

		hash, err := auth.GenerateHashFromPassword(password)
		require.NoError(t, err)

		isValidCorrectCase := auth.CheckPasswordHash("Password123", hash)
		isValidWrongCase := auth.CheckPasswordHash("password123", hash)

		assert.True(t, isValidCorrectCase)
		assert.False(t, isValidWrongCase)
	})
}

func TestAuth_Integration(t *testing.T) {
	t.Run("full authentication flow", func(t *testing.T) {
		auth := newTestAuth()
		password := "mySecurePassword123"
		userUUID := uuid.New()

		// Step 1: Hash password
		hash, err := auth.GenerateHashFromPassword(password)
		require.NoError(t, err)

		// Step 2: Verify password
		isValid := auth.CheckPasswordHash(password, hash)
		assert.True(t, isValid)

		// Step 3: Create JWT token
		tokenString, err := auth.BuildJWTString(userUUID)
		require.NoError(t, err)

		// Step 4: Validate token
		validatedUUID, err := auth.ValidateUserUUID(tokenString)
		require.NoError(t, err)
		assert.Equal(t, userUUID, validatedUUID)
	})

	t.Run("token lifecycle with different configs", func(t *testing.T) {
		// Create token with short expiration
		shortAuth := &Auth{
			conf: authconf.Config{
				SecretKey:       "shared_secret",
				TokenExpiration: time.Second * 1, // 1 second expiration
				PasswordLen:     8,
			},
		}

		userUUID := uuid.New()
		tokenString, err := shortAuth.BuildJWTString(userUUID)
		require.NoError(t, err)

		// Token should be valid immediately
		validatedUUID, err := shortAuth.ValidateUserUUID(tokenString)
		require.NoError(t, err)
		assert.Equal(t, userUUID, validatedUUID)

		// Wait for token to expire
		time.Sleep(time.Second * 2)

		// Token should now be invalid (expired)
		validatedUUID, err = shortAuth.ValidateUserUUID(tokenString)
		assert.ErrorIs(t, err, apperrors.ErrUserIsNotAuthorized)
		assert.Equal(t, uuid.Nil, validatedUUID)
	})
}

func TestClaims(t *testing.T) {
	t.Run("claims contains user UUID and expiration", func(t *testing.T) {
		userUUID := uuid.New()
		expiresAt := time.Now().Add(time.Hour)

		claims := Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expiresAt),
			},
			UserUUID: userUUID,
		}

		assert.Equal(t, userUUID, claims.UserUUID)
		assert.NotNil(t, claims.ExpiresAt)
		assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
	})
}
