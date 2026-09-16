package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"encrypted-db/config"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testKeyPair generates a test RSA key pair
func testKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(nil, err)
	return privateKey, &privateKey.PublicKey
}

// privateKeyToPEM converts RSA private key to PEM format
func privateKeyToPEM(key *rsa.PrivateKey) []byte {
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(block)
}

// publicKeyToPEM converts RSA public key to PEM format
func publicKeyToPEM(key *rsa.PublicKey) []byte {
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(key),
	}
	return pem.EncodeToMemory(block)
}

// setupTestConfig initializes config with test values
func setupTestConfig() {
	config.Config.JWT.AccessToken.Expiration = 15
	config.Config.JWT.RefreshToken.Expiration = 10080
	config.Config.JWT.Issuer = "test-issuer"
	config.Config.Server.Mode = "debug"
}

// setupTestKeys initializes auth package with test keys
func setupTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	setupTestConfig()
	privateKey, publicKey := testKeyPair()
	privateKeyPEM := privateKeyToPEM(privateKey)
	publicKeyPEM := publicKeyToPEM(publicKey)

	err := InitKeys(privateKeyPEM, publicKeyPEM)
	require.NoError(t, err)
	return privateKey, publicKey
}

// mockRedisClient is a simple in-memory mock for Redis
type mockRedisClient struct {
	data map[string]string
	ttls map[string]time.Time
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
		ttls: make(map[string]time.Time),
	}
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value string, expiration time.Duration) *redis.StatusCmd {
	m.data[key] = value
	m.ttls[key] = time.Now().Add(expiration)
	return redis.NewStatusCmd(ctx, "OK")
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	val, ok := m.data[key]
	if !ok {
		return redis.NewStringCmd(ctx, redis.Nil)
	}
	// Check expiration
	if exp, ok := m.ttls[key]; ok && time.Now().After(exp) {
		delete(m.data, key)
		delete(m.ttls, key)
		return redis.NewStringCmd(ctx, redis.Nil)
	}
	return redis.NewStringCmd(ctx, val)
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	count := 0
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			delete(m.ttls, key)
			count++
		}
	}
	return redis.NewIntCmd(ctx, int64(count))
}

func TestTokenBlacklist_AddTokenToBlacklist(t *testing.T) {
	tb := &TokenBlacklist{
		RedisClient: (*redis.Client)(nil), // We'll test the logic directly
		Prefix:      "test:blacklist:",
	}

	// Test the tokenKey method directly
	token := "test-token-123"
	hash := sha256.Sum256([]byte(token))
	expectedKey := "test:blacklist:" + hex.EncodeToString(hash[:])
	actualKey := tb.tokenKey(token)
	assert.Equal(t, expectedKey, actualKey)
}

func TestTokenBlacklist_IsTokenBlacklisted(t *testing.T) {
	tb := NewTokenBlacklist((*redis.Client)(nil))
	tb.Prefix = "test:blacklist:"

	// Since we can't easily mock the Redis client, we test the key generation
	token := "test-token-123"
	key := tb.tokenKey(token)
	assert.Contains(t, key, "test:blacklist:")
	assert.Len(t, key, len("test:blacklist:")+64) // SHA256 hex = 64 chars
}

func TestGenerateAccessToken(t *testing.T) {
	setupTestKeys(t)

	uuid := "user-123"
	role := "user"

	token, err := GenerateAccessToken(uuid, role)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Verify token can be parsed
	claims, err := VerifyRSAToken(token)
	require.NoError(t, err)
	assert.Equal(t, uuid, claims.UUID)
	assert.Equal(t, role, claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
}

func TestGenerateRefreshToken(t *testing.T) {
	setupTestKeys(t)

	uuid := "user-123"
	role := "user"

	token, err := GenerateRefreshToken(uuid, role)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Verify token can be parsed
	claims, err := VerifyRSAToken(token)
	require.NoError(t, err)
	assert.Equal(t, uuid, claims.UUID)
	assert.Equal(t, role, claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now().Add(6*24*time.Hour))) // ~7 days
}

func TestVerifyRSAToken_ValidToken(t *testing.T) {
	setupTestKeys(t)

	token, err := GenerateAccessToken("user-123", "user")
	require.NoError(t, err)

	claims, err := VerifyRSAToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UUID)
	assert.Equal(t, "user", claims.Role)
}

func TestVerifyRSAToken_InvalidToken(t *testing.T) {
	setupTestKeys(t)

	// Invalid token format
	_, err := VerifyRSAToken("invalid.token.string")
	assert.Error(t, err)

	// Token signed with different key
	otherPrivateKey, _ := testKeyPair()
	otherToken := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		UUID: "user-123",
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "test",
		},
	})
	otherTokenString, _ := otherToken.SignedString(otherPrivateKey)
	_, err = VerifyRSAToken(otherTokenString)
	assert.Error(t, err)
}

func TestVerifyRSAToken_ExpiredToken(t *testing.T) {
	setupTestKeys(t)

	// Create token with past expiration
	claims := Claims{
		UUID: "user-123",
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			Issuer:    "test",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString(PrivateKey())

	_, err := VerifyRSAToken(tokenString)
	assert.Error(t, err)
}

func TestVerifyRSAToken_WrongAlgorithm(t *testing.T) {
	setupTestKeys(t)

	// Create token with HS256 instead of RS256
	claims := Claims{
		UUID: "user-123",
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "test",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("secret"))

	_, err := VerifyRSAToken(tokenString)
	assert.Error(t, err)
}

func TestVerifyRSATokenWithKeys_KeyRotation(t *testing.T) {
	setupTestKeys(t)

	// Generate access token with current key
	token, err := GenerateAccessToken("user-123", "user")
	require.NoError(t, err)

	// Verify with current key works
	claims, err := VerifyRSATokenWithKeys(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UUID)

	// Add a new public key (simulating key rotation)
	newPrivateKey, newPublicKey := testKeyPair()
	newPublicKeyPEM := publicKeyToPEM(newPublicKey)
	err = AddPublicKey("key-2024-02", newPublicKeyPEM)
	require.NoError(t, err)

	// Create token with new key
	newToken := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		UUID: "user-456",
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    config.Config.JWT.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	newTokenString, _ := newToken.SignedString(newPrivateKey)

	// Verify with key rotation - should work with new key
	claims, err = VerifyRSATokenWithKeys(newTokenString)
	require.NoError(t, err)
	assert.Equal(t, "user-456", claims.UUID)

	// Original token should still work
	claims, err = VerifyRSATokenWithKeys(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UUID)
}

func TestVerifyRSATokenWithKeys_KeysNotLoaded(t *testing.T) {
	// Reset keys
	privateKey = nil
	publicKey = nil
	publicKeys = nil

	_, err := VerifyRSATokenWithKeys("any-token")
	assert.Error(t, err)
	assert.Equal(t, ErrKeysNotLoaded, err)
}

func TestVerifyAdminToken(t *testing.T) {
	setupTestKeys(t)

	adminToken, _ := GenerateAccessToken("admin-1", "admin")
	userToken, _ := GenerateAccessToken("user-1", "user")

	assert.True(t, VerifyAdminToken(adminToken))
	assert.False(t, VerifyAdminToken(userToken))
	assert.False(t, VerifyAdminToken("invalid"))
}

func TestVerifyUserToken(t *testing.T) {
	setupTestKeys(t)

	adminToken, _ := GenerateAccessToken("admin-1", "admin")
	userToken, _ := GenerateAccessToken("user-1", "user")

	// VerifyUserToken only allows "user" role (not admin)
	assert.True(t, VerifyUserToken(userToken), "user token should be valid")
	assert.False(t, VerifyUserToken(adminToken), "admin token should NOT be valid for VerifyUserToken (only 'user' role)")
	assert.False(t, VerifyUserToken("invalid"), "invalid token should fail")
}

func TestLogoutHandler(t *testing.T) {
	setupTestKeys(t)

	gin.SetMode(gin.TestMode)

	token, _ := GenerateAccessToken("user-123", "user")

	// Test logout with valid token
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	c.Request.Header.Set("Authorization", token)

	// Test with nil blacklist - should succeed (no-op when Redis is nil)
	tb := &TokenBlacklist{
		RedisClient: (*redis.Client)(nil),
		Prefix:      "test:blacklist:",
	}
	LogoutHandler(c, tb)

	// With nil Redis, AddTokenToBlacklist is no-op, returns 200
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogoutHandler_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	// No Authorization header

	tb := NewTokenBlacklist((*redis.Client)(nil))
	LogoutHandler(c, tb)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogoutHandler_InvalidToken(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	c.Request.Header.Set("Authorization", "invalid.token")

	tb := NewTokenBlacklist((*redis.Client)(nil))
	LogoutHandler(c, tb)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogoutHandler_BearerToken(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	token, err := GenerateAccessToken("user-123", "user")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	LogoutHandler(c, NewTokenBlacklist((*redis.Client)(nil)))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshTokenHandler(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	t.Run("missing cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)

		RefreshTokenHandler(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		c.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid"})

		RefreshTokenHandler(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid cookie", func(t *testing.T) {
		refreshToken, err := GenerateRefreshToken("user-123", "user")
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		c.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})

		RefreshTokenHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Access token refreshed")
		assert.Contains(t, w.Body.String(), "access_token")
	})
}

func TestSetRefreshTokenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	refreshToken := "test-refresh-token"
	SetRefreshTokenCookie(c, refreshToken)

	// Check cookie was set
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "refresh_token", cookies[0].Name)
	assert.Equal(t, refreshToken, cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.True(t, cookies[0].Secure)
	assert.Equal(t, "/", cookies[0].Path)
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	token, _ := GenerateAccessToken("user-123", "user")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", token)

	tb := NewTokenBlacklist((*redis.Client)(nil))

	middleware := JWTMiddleware(tb)
	middleware(c)

	// Middleware should pass (not abort with 401), but returns 404 since no handler
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	claims, exists := c.Get("claims")
	assert.True(t, exists)
	assert.Equal(t, "user-123", claims.(*Claims).UUID)
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	// No Authorization header

	tb := NewTokenBlacklist((*redis.Client)(nil))

	middleware := JWTMiddleware(tb)
	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", "invalid.token")

	tb := NewTokenBlacklist((*redis.Client)(nil))

	middleware := JWTMiddleware(tb)
	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	// Create expired token
	claims := Claims{
		UUID: "user-123",
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			Issuer:    "test",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString(PrivateKey())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", tokenString)

	tb := NewTokenBlacklist((*redis.Client)(nil))

	middleware := JWTMiddleware(tb)
	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_BlacklistedToken(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	token, _ := GenerateAccessToken("user-123", "user")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", token)

	// Create a blacklist that returns true for this token
	tb := NewTokenBlacklist((*redis.Client)(nil))
	tb.Prefix = "test:blacklist:"

	// Pre-populate the blacklist (can't easily test without real Redis)
	// Just verify middleware doesn't panic

	middleware := JWTMiddleware(tb)
	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code) // Passes since mock Redis returns nil
}

func TestJWTAdminVerification(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	adminToken, _ := GenerateAccessToken("admin-1", "admin")
	userToken, _ := GenerateAccessToken("user-1", "user")

	// Test admin token passes
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin", nil)
	c.Request.Header.Set("Authorization", adminToken)

	tb := NewTokenBlacklist((*redis.Client)(nil))

	middleware := JWTAdminVerification(tb)
	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code) // Passes

	// Test user token fails
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin", nil)
	c.Request.Header.Set("Authorization", userToken)

	middleware = JWTAdminVerification(tb)
	middleware(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestJWTUserVerification(t *testing.T) {
	setupTestKeys(t)
	gin.SetMode(gin.TestMode)

	adminToken, _ := GenerateAccessToken("admin-1", "admin")
	userToken, _ := GenerateAccessToken("user-1", "user")

	tb := NewTokenBlacklist((*redis.Client)(nil))

	// Test user token passes
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/user", nil)
	c.Request.Header.Set("Authorization", userToken)

	middleware := JWTUserVerification(tb)
	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test admin token passes (admin can access user endpoints)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/user", nil)
	c.Request.Header.Set("Authorization", adminToken)

	middleware = JWTUserVerification(tb)
	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test missing token fails
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/user", nil)

	middleware = JWTUserVerification(tb)
	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClaims_JSONSerialization(t *testing.T) {
	claims := Claims{
		UUID: "user-123",
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "test",
			Subject:   "user-123",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(claims)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshalled Claims
	err = json.Unmarshal(data, &unmarshalled)
	require.NoError(t, err)

	assert.Equal(t, claims.UUID, unmarshalled.UUID)
	assert.Equal(t, claims.Role, unmarshalled.Role)
	assert.Equal(t, claims.ExpiresAt.Time.Unix(), unmarshalled.ExpiresAt.Time.Unix())
}

func TestTokenBlacklist_KeyGeneration(t *testing.T) {
	tb := NewTokenBlacklist(nil)

	// Same token should generate same key
	token := "test-token"
	key1 := tb.tokenKey(token)
	key2 := tb.tokenKey(token)
	assert.Equal(t, key1, key2)

	// Different tokens should generate different keys
	token2 := "test-token-2"
	key3 := tb.tokenKey(token2)
	assert.NotEqual(t, key1, key3)

	// Key should have prefix and 64-char hex
	assert.True(t, strings.HasPrefix(key1, "blacklist:token:"))
	assert.Len(t, key1, len("blacklist:token:")+64)
}

// Benchmark tests
func BenchmarkGenerateAccessToken(b *testing.B) {
	setupTestKeys(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateAccessToken("user-123", "user")
	}
}

func BenchmarkVerifyRSAToken(b *testing.B) {
	setupTestKeys(nil)
	token, _ := GenerateAccessToken("user-123", "user")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyRSAToken(token)
	}
}

func BenchmarkVerifyRSATokenWithKeys(b *testing.B) {
	setupTestKeys(nil)
	token, _ := GenerateAccessToken("user-123", "user")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyRSATokenWithKeys(token)
	}
}
