package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/apperrors"
	"github.com/ar4ie13/gophkeeper/internal/server/handlers/config"
	"github.com/ar4ie13/gophkeeper/internal/server/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuth is a mock implementation of the Auth interface
type MockAuth struct {
	mock.Mock
}

func (m *MockAuth) BuildJWTString(userUUID uuid.UUID) (string, error) {
	args := m.Called(userUUID)
	return args.String(0), args.Error(1)
}

func (m *MockAuth) ValidateUserUUID(tokenString string) (uuid.UUID, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockAuth) GenerateHashFromPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockAuth) CheckPasswordHash(password, hash string) bool {
	args := m.Called(password, hash)
	return args.Bool(0)
}

// MockService is a mock implementation of the Service interface
type MockService struct {
	mock.Mock
}

func (m *MockService) LoginUser(ctx context.Context, login string) (models.User, error) {
	args := m.Called(ctx, login)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockService) CreateUser(ctx context.Context, user models.User) (uuid.UUID, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockService) StoreSecret(ctx context.Context, secretType string, secret any) error {
	args := m.Called(ctx, secretType, secret)
	return args.Error(0)
}

func (m *MockService) DecryptSecret(ctx context.Context, secretType string, userUUID uuid.UUID, name string) (any, error) {
	args := m.Called(ctx, secretType, userUUID, name)
	return args.Get(0), args.Error(1)
}

func (m *MockService) UpdateSecret(ctx context.Context, secretType string, secret any) error {
	args := m.Called(ctx, secretType, secret)
	return args.Error(0)
}

func (m *MockService) DeleteSecret(ctx context.Context, secretType string, userUUID uuid.UUID, name string) error {
	args := m.Called(ctx, secretType, userUUID, name)
	return args.Error(0)
}

func (m *MockService) GetAllUserTexts(ctx context.Context, user uuid.UUID) ([]models.Text, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Text), args.Error(1)
}

func (m *MockService) GetAllUserCredentials(ctx context.Context, user uuid.UUID) ([]models.Credential, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Credential), args.Error(1)
}

func (m *MockService) GetAllUserCards(ctx context.Context, user uuid.UUID) ([]models.Card, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Card), args.Error(1)
}

func (m *MockService) GetAllUserFiles(ctx context.Context, user uuid.UUID) ([]models.File, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.File), args.Error(1)
}

func setupTest() (*Handlers, *MockAuth, *MockService) {
	gin.SetMode(gin.TestMode)

	mockAuth := new(MockAuth)
	mockService := new(MockService)

	cfg := config.ServerConfig{
		ServerAddr:  ":8080",
		TLSCertPath: "cert.pem",
		TLSKeyPath:  "key.pem",
	}

	handler := NewHandler(cfg, zerolog.Nop(), mockAuth, mockService)

	return handler, mockAuth, mockService
}

func TestNewHandler(t *testing.T) {
	cfg := config.ServerConfig{}
	logger := zerolog.Nop()
	mockAuth := new(MockAuth)
	mockService := new(MockService)

	handler := NewHandler(cfg, logger, mockAuth, mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, cfg, handler.cfg)
	assert.Equal(t, mockAuth, handler.auth)
	assert.Equal(t, mockService, handler.srv)
}

func TestHandlers_getStatusCode(t *testing.T) {
	handler, _, _ := setupTest()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "user already exists",
			err:            apperrors.ErrUserAlreadyExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "password min symbols",
			err:            apperrors.ErrPasswordMinSymbols,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found",
			err:            apperrors.ErrNotFound,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "unknown error",
			err:            errors.New("unknown error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := handler.getStatusCode(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestHandlers_userRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, mockAuth, mockService := setupTest()

		userUUID := uuid.New()
		registerReq := registerRequest{
			Login:    "testuser",
			Password: "password123",
		}

		mockAuth.On("GenerateHashFromPassword", "password123").Return("hashed_password", nil)
		mockService.On("CreateUser", mock.Anything, mock.MatchedBy(func(u models.User) bool {
			return u.Login == "testuser" && u.PasswordHash == "hashed_password"
		})).Return(userUUID, nil)
		mockAuth.On("BuildJWTString", userUUID).Return("jwt_token", nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body, _ := json.Marshal(registerReq)
		c.Request = httptest.NewRequest("POST", "/api/user/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userRegister(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockAuth.AssertExpectations(t)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		handler, _, _ := setupTest()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest("POST", "/api/user/register", bytes.NewBuffer([]byte("invalid")))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userRegister(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("hash generation error", func(t *testing.T) {
		handler, mockAuth, _ := setupTest()

		registerReq := registerRequest{
			Login:    "testuser",
			Password: "password123",
		}

		mockAuth.On("GenerateHashFromPassword", "password123").Return("", errors.New("hash error"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body, _ := json.Marshal(registerReq)
		c.Request = httptest.NewRequest("POST", "/api/user/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userRegister(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("user already exists", func(t *testing.T) {
		handler, mockAuth, mockService := setupTest()

		registerReq := registerRequest{
			Login:    "testuser",
			Password: "password123",
		}

		mockAuth.On("GenerateHashFromPassword", "password123").Return("hashed_password", nil)
		mockService.On("CreateUser", mock.Anything, mock.Anything).Return(nil, apperrors.ErrUserAlreadyExists)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body, _ := json.Marshal(registerReq)
		c.Request = httptest.NewRequest("POST", "/api/user/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userRegister(c)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockAuth.AssertExpectations(t)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_userLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, mockAuth, mockService := setupTest()

		userUUID := uuid.New()
		loginReq := loginRequest{
			Login:    "testuser",
			Password: "password123",
		}

		user := models.User{
			UUID:         userUUID,
			Login:        "testuser",
			PasswordHash: "hashed_password",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		mockService.On("LoginUser", mock.Anything, "testuser").Return(user, nil)
		mockAuth.On("CheckPasswordHash", "password123", "hashed_password").Return(true)
		mockAuth.On("BuildJWTString", userUUID).Return("jwt_token", nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body, _ := json.Marshal(loginReq)
		c.Request = httptest.NewRequest("POST", "/api/user/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userLogin(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockAuth.AssertExpectations(t)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		handler, _, _ := setupTest()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest("POST", "/api/user/login", bytes.NewBuffer([]byte("invalid")))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userLogin(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		handler, _, mockService := setupTest()

		loginReq := loginRequest{
			Login:    "testuser",
			Password: "password123",
		}

		mockService.On("LoginUser", mock.Anything, "testuser").Return(models.User{}, apperrors.ErrUserNotFound)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body, _ := json.Marshal(loginReq)
		c.Request = httptest.NewRequest("POST", "/api/user/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userLogin(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		handler, mockAuth, mockService := setupTest()

		userUUID := uuid.New()
		loginReq := loginRequest{
			Login:    "testuser",
			Password: "wrongpassword",
		}

		user := models.User{
			UUID:         userUUID,
			Login:        "testuser",
			PasswordHash: "hashed_password",
		}

		mockService.On("LoginUser", mock.Anything, "testuser").Return(user, nil)
		mockAuth.On("CheckPasswordHash", "wrongpassword", "hashed_password").Return(false)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body, _ := json.Marshal(loginReq)
		c.Request = httptest.NewRequest("POST", "/api/user/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.userLogin(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockAuth.AssertExpectations(t)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_getUserUUIDFromRequest(t *testing.T) {
	handler, _, _ := setupTest()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		expectedUUID := uuid.New()
		c.Set("user_uuid", expectedUUID.String())

		userUUID, err := handler.getUserUUIDFromRequest(c)
		assert.NoError(t, err)
		assert.Equal(t, expectedUUID, userUUID)
	})

	t.Run("uuid not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userUUID, err := handler.getUserUUIDFromRequest(c)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userUUID)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set("user_uuid", "invalid-uuid")

		userUUID, err := handler.getUserUUIDFromRequest(c)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userUUID)
	})
}

func TestHandlers_ping(t *testing.T) {
	handler, _, _ := setupTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.ping(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlers_storeText(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		textReq := text{
			Name:        "test_text",
			Description: "description",
			Text:        "secret text",
		}

		mockService.On("StoreSecret", mock.Anything, "text", mock.MatchedBy(func(t models.Text) bool {
			return t.Name == "test_text" && t.UserUUID == userUUID
		})).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(textReq)
		c.Request = httptest.NewRequest("POST", "/api/user/text/store", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeText(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid user uuid", func(t *testing.T) {
		handler, _, _ := setupTest()

		textReq := text{
			Name: "test_text",
			Text: "secret text",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body, _ := json.Marshal(textReq)
		c.Request = httptest.NewRequest("POST", "/api/user/text/store", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeText(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		handler, _, _ := setupTest()

		userUUID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		c.Request = httptest.NewRequest("POST", "/api/user/text/store", bytes.NewBuffer([]byte("invalid")))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeText(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlers_getText(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedText := models.Text{
			Name:        "test_text",
			Description: "description",
			UserText:    []byte("secret text"),
			UserUUID:    userUUID,
		}

		mockService.On("DecryptSecret", mock.Anything, "text", userUUID, "test_text").Return(expectedText, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())
		c.Params = gin.Params{{Key: "name", Value: "test_text"}}

		c.Request = httptest.NewRequest("GET", "/api/user/text/get/test_text", nil)

		handler.getText(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response text
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test_text", response.Name)
		assert.Equal(t, "secret text", response.Text)

		mockService.AssertExpectations(t)
	})

	t.Run("missing name", func(t *testing.T) {
		handler, _, _ := setupTest()

		userUUID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())
		c.Params = gin.Params{{Key: "name", Value: ""}}

		c.Request = httptest.NewRequest("GET", "/api/user/text/get/", nil)

		handler.getText(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlers_deleteSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		deleteReq := deleteRequest{
			Name: "test_secret",
			Type: "text",
		}

		mockService.On("DeleteSecret", mock.Anything, "text", userUUID, "test_secret").Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(deleteReq)
		c.Request = httptest.NewRequest("DELETE", "/api/user/secret/delete", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.deleteSecret(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		handler, _, _ := setupTest()

		userUUID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		c.Request = httptest.NewRequest("DELETE", "/api/user/secret/delete", bytes.NewBuffer([]byte("invalid")))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.deleteSecret(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlers_getAllUserTexts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedTexts := []models.Text{
			{Name: "text1", Description: "desc1", UserUUID: userUUID},
			{Name: "text2", Description: "desc2", UserUUID: userUUID},
		}

		mockService.On("GetAllUserTexts", mock.Anything, userUUID).Return(expectedTexts, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		c.Request = httptest.NewRequest("GET", "/api/user/text/list", nil)

		handler.getAllUserTexts(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []allSecrets
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "text1", response[0].Name)

		mockService.AssertExpectations(t)
	})

	t.Run("no texts found", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()

		mockService.On("GetAllUserTexts", mock.Anything, userUUID).Return(nil, apperrors.ErrNotFound)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		c.Request = httptest.NewRequest("GET", "/api/user/text/list", nil)

		handler.getAllUserTexts(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_updateText(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		textReq := text{
			Name:        "test_text",
			Description: "updated description",
			Text:        "updated text",
		}

		mockService.On("UpdateSecret", mock.Anything, "text", mock.MatchedBy(func(t models.Text) bool {
			return t.Name == "test_text" && t.UserUUID == userUUID
		})).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(textReq)
		c.Request = httptest.NewRequest("PATCH", "/api/user/text/patch", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.updateText(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_authMiddleware(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, mockAuth, _ := setupTest()

		userUUID := uuid.New()

		mockAuth.On("ValidateUserUUID", "valid_token").Return(userUUID, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "user_uuid",
			Value: "valid_token",
		})

		middleware := handler.authMiddleware()
		middleware(c)

		assert.False(t, c.IsAborted())
		userUUIDStr, exists := c.Get("user_uuid")
		assert.True(t, exists)
		assert.Equal(t, userUUID.String(), userUUIDStr)

		mockAuth.AssertExpectations(t)
	})

	t.Run("missing cookie", func(t *testing.T) {
		handler, _, _ := setupTest()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware := handler.authMiddleware()
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		handler, mockAuth, _ := setupTest()

		mockAuth.On("ValidateUserUUID", "invalid_token").Return(nil, errors.New("invalid token"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "user_uuid",
			Value: "invalid_token",
		})

		middleware := handler.authMiddleware()
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		mockAuth.AssertExpectations(t)
	})
}

func TestHandlers_storeCredential(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		credReq := credential{
			Name:        "test_cred",
			Description: "description",
			Login:       "mylogin",
			Password:    "mypassword",
		}

		mockService.On("StoreSecret", mock.Anything, "credential", mock.MatchedBy(func(c models.Credential) bool {
			return c.Name == "test_cred" && c.UserUUID == userUUID
		})).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(credReq)
		c.Request = httptest.NewRequest("POST", "/api/user/credential/store", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeCredential(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_getCredential(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedCred := models.Credential{
			Name:        "test_cred",
			Description: "description",
			Login:       []byte("mylogin"),
			Password:    []byte("mypassword"),
			UserUUID:    userUUID,
		}

		mockService.On("DecryptSecret", mock.Anything, "credential", userUUID, "test_cred").Return(expectedCred, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())
		c.Params = gin.Params{{Key: "name", Value: "test_cred"}}

		c.Request = httptest.NewRequest("GET", "/api/user/credential/get/test_cred", nil)

		handler.getCredential(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_storeCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		cardReq := card{
			Name:        "test_card",
			Description: "description",
			CardNumber:  "1234567890123456",
			Owner:       "John Doe",
			ExpiresAt:   "12/25",
			CVC:         "123",
		}

		mockService.On("StoreSecret", mock.Anything, "card", mock.MatchedBy(func(c models.Card) bool {
			return c.Name == "test_card" && c.UserUUID == userUUID
		})).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(cardReq)
		c.Request = httptest.NewRequest("POST", "/api/user/card/store", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeCard(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_getCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedCard := models.Card{
			Name:        "test_card",
			Description: "description",
			CardNumber:  []byte("1234567890123456"),
			Owner:       []byte("John Doe"),
			ExpiresAt:   []byte("12/25"),
			CVC:         []byte("123"),
			UserUUID:    userUUID,
		}

		mockService.On("DecryptSecret", mock.Anything, "card", userUUID, "test_card").Return(expectedCard, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())
		c.Params = gin.Params{{Key: "name", Value: "test_card"}}

		c.Request = httptest.NewRequest("GET", "/api/user/card/get/test_card", nil)

		handler.getCard(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_storeFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		// Create a small base64-encoded file
		fileContent := "SGVsbG8gV29ybGQh" // "Hello World!" in base64

		fileReq := file{
			Name:        "test_file",
			Description: "description",
			UserFile:    fileContent,
		}

		mockService.On("StoreSecret", mock.Anything, "file", mock.MatchedBy(func(f models.File) bool {
			return f.Name == "test_file" && f.UserUUID == userUUID
		})).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(fileReq)
		c.Request = httptest.NewRequest("POST", "/api/user/file/store", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeFile(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("file too large", func(t *testing.T) {
		handler, _, _ := setupTest()

		userUUID := uuid.New()
		// Create a base64 string that exceeds the limit
		largeContent := make([]byte, maxBase64Len+100)
		for i := range largeContent {
			largeContent[i] = 'A'
		}

		fileReq := file{
			Name:     "test_file",
			UserFile: string(largeContent),
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(fileReq)
		c.Request = httptest.NewRequest("POST", "/api/user/file/store", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeFile(c)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("invalid base64", func(t *testing.T) {
		handler, _, _ := setupTest()

		userUUID := uuid.New()

		fileReq := file{
			Name:     "test_file",
			UserFile: "invalid-base64!!!",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(fileReq)
		c.Request = httptest.NewRequest("POST", "/api/user/file/store", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.storeFile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlers_getFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedFile := models.File{
			Name:        "test_file",
			Description: "description",
			UserFile:    []byte("Hello World!"),
			UserUUID:    userUUID,
		}

		mockService.On("DecryptSecret", mock.Anything, "file", userUUID, "test_file").Return(expectedFile, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())
		c.Params = gin.Params{{Key: "name", Value: "test_file"}}

		c.Request = httptest.NewRequest("GET", "/api/user/file/get/test_file", nil)

		handler.getFile(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_getAllUserCredentials(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedCreds := []models.Credential{
			{Name: "cred1", Description: "desc1", UserUUID: userUUID},
			{Name: "cred2", Description: "desc2", UserUUID: userUUID},
		}

		mockService.On("GetAllUserCredentials", mock.Anything, userUUID).Return(expectedCreds, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		c.Request = httptest.NewRequest("GET", "/api/user/credential/list", nil)

		handler.getAllUserCredentials(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_getAllUserCards(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedCards := []models.Card{
			{Name: "card1", Description: "desc1", UserUUID: userUUID},
			{Name: "card2", Description: "desc2", UserUUID: userUUID},
		}

		mockService.On("GetAllUserCards", mock.Anything, userUUID).Return(expectedCards, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		c.Request = httptest.NewRequest("GET", "/api/user/card/list", nil)

		handler.getAllUserCards(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_getAllUserFiles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		expectedFiles := []models.File{
			{Name: "file1", Description: "desc1", UserUUID: userUUID},
			{Name: "file2", Description: "desc2", UserUUID: userUUID},
		}

		mockService.On("GetAllUserFiles", mock.Anything, userUUID).Return(expectedFiles, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		c.Request = httptest.NewRequest("GET", "/api/user/file/list", nil)

		handler.getAllUserFiles(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_updateCredential(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		credReq := credential{
			Name:        "test_cred",
			Description: "updated description",
			Login:       "updated_login",
			Password:    "updated_password",
		}

		mockService.On("UpdateSecret", mock.Anything, "credential", mock.MatchedBy(func(c models.Credential) bool {
			return c.Name == "test_cred" && c.UserUUID == userUUID
		})).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(credReq)
		c.Request = httptest.NewRequest("PATCH", "/api/user/credential/patch", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.updateCredential(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandlers_updateCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _, mockService := setupTest()

		userUUID := uuid.New()
		cardReq := card{
			Name:        "test_card",
			Description: "updated description",
			CardNumber:  "9876543210987654",
			Owner:       "Jane Doe",
			ExpiresAt:   "06/28",
			CVC:         "456",
		}

		mockService.On("UpdateSecret", mock.Anything, "card", mock.MatchedBy(func(c models.Card) bool {
			return c.Name == "test_card" && c.UserUUID == userUUID
		})).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_uuid", userUUID.String())

		body, _ := json.Marshal(cardReq)
		c.Request = httptest.NewRequest("PATCH", "/api/user/card/patch", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.updateCard(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}
