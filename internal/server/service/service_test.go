package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/apperrors"
	"github.com/ar4ie13/gophkeeper/internal/server/models"
	"github.com/ar4ie13/gophkeeper/internal/server/service/config"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateUser(ctx context.Context, user models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	args := m.Called(ctx, login)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockRepository) StoreSecret(ctx context.Context, secretType string, secret any) error {
	args := m.Called(ctx, secretType, secret)
	return args.Error(0)
}

func (m *MockRepository) GetSecret(ctx context.Context, userUUID uuid.UUID, secretType string, secretName string) (any, error) {
	args := m.Called(ctx, userUUID, secretType, secretName)
	return args.Get(0), args.Error(1)
}

func (m *MockRepository) UpdateSecret(ctx context.Context, secretType string, secret any) error {
	args := m.Called(ctx, secretType, secret)
	return args.Error(0)
}

func (m *MockRepository) DeleteSecret(ctx context.Context, userUUID uuid.UUID, secretType string, secretName string) error {
	args := m.Called(ctx, userUUID, secretType, secretName)
	return args.Error(0)
}

func (m *MockRepository) GetUserKey(ctx context.Context, user uuid.UUID) ([]byte, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockRepository) GetAllUserTexts(ctx context.Context, user uuid.UUID) ([]models.Text, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Text), args.Error(1)
}

func (m *MockRepository) GetAllUserCredentials(ctx context.Context, user uuid.UUID) ([]models.Credential, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Credential), args.Error(1)
}

func (m *MockRepository) GetAllUserCards(ctx context.Context, user uuid.UUID) ([]models.Card, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Card), args.Error(1)
}

func (m *MockRepository) GetAllUserFiles(ctx context.Context, user uuid.UUID) ([]models.File, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.File), args.Error(1)
}

func newTestService(repo Repository) *Service {
	return NewService(
		repo,
		zerolog.Nop(),
		config.ServiceConf{
			MasterKey: "12345678901234567890123456789012", // 32-byte key for AES-256
		},
	)
}

func TestService_checkLoginString(t *testing.T) {
	mockRepo := new(MockRepository)
	service := newTestService(mockRepo)

	tests := []struct {
		name     string
		login    string
		expected bool
	}{
		{"valid lowercase", "testuser", true},
		{"valid uppercase", "TESTUSER", true},
		{"valid mixed case", "TestUser", true},
		{"valid with numbers", "test123", true},
		{"invalid with special chars", "test@user", false},
		{"invalid with spaces", "test user", false},
		{"invalid with underscore", "test_user", false},
		{"invalid with dash", "test-user", false},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.checkLoginString(tt.login)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_LoginUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		encryptedKey, err := service.encryptKey([]byte("user_encryption_key_1234567890ab"))
		require.NoError(t, err)

		expectedUser := models.User{
			UUID:         userUUID,
			Login:        "testuser",
			PasswordHash: "hashed_password",
			EncryptedKey: encryptedKey,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		mockRepo.On("GetUserByLogin", ctx, "testuser").Return(expectedUser, nil)

		user, err := service.LoginUser(ctx, "testuser")
		assert.NoError(t, err)
		assert.Equal(t, userUUID, user.UUID)
		assert.Equal(t, "testuser", user.Login)
		// The encrypted key should be decrypted
		assert.NotEqual(t, encryptedKey, user.EncryptedKey)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid login string", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		user, err := service.LoginUser(ctx, "test@user")
		assert.ErrorIs(t, err, apperrors.ErrInvalidLoginString)
		assert.Equal(t, models.User{}, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		mockRepo.On("GetUserByLogin", ctx, "testuser").Return(models.User{}, apperrors.ErrUserNotFound)

		user, err := service.LoginUser(ctx, "testuser")
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
		assert.Equal(t, models.User{}, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_CreateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		user := models.User{
			Login:        "newuser",
			PasswordHash: "hashed_password",
		}

		mockRepo.On("CreateUser", ctx, mock.MatchedBy(func(u models.User) bool {
			return u.Login == "newuser" && u.UUID != uuid.Nil && len(u.EncryptedKey) > 0
		})).Return(nil)

		userUUID, err := service.CreateUser(ctx, user)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, userUUID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid login string", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		user := models.User{
			Login:        "new@user",
			PasswordHash: "hashed_password",
		}

		userUUID, err := service.CreateUser(ctx, user)
		assert.ErrorIs(t, err, apperrors.ErrInvalidLoginString)
		assert.Equal(t, uuid.Nil, userUUID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		user := models.User{
			Login:        "newuser",
			PasswordHash: "hashed_password",
		}

		mockRepo.On("CreateUser", ctx, mock.Anything).Return(apperrors.ErrUserAlreadyExists)

		userUUID, err := service.CreateUser(ctx, user)
		assert.ErrorIs(t, err, apperrors.ErrUserAlreadyExists)
		assert.Equal(t, uuid.Nil, userUUID)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_StoreSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("store text success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		text := models.Text{
			UserUUID:    userUUID,
			UserText:    []byte("my secret text"),
			Name:        "test_text",
			Description: "test description",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("StoreSecret", ctx, "text", mock.MatchedBy(func(t models.Text) bool {
			return t.Name == "test_text" && len(t.UserText) > 0
		})).Return(nil)

		err := service.StoreSecret(ctx, "text", text)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("store credential success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		credential := models.Credential{
			UserUUID:    userUUID,
			Login:       []byte("mylogin"),
			Password:    []byte("mypassword"),
			Name:        "test_cred",
			Description: "test description",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("StoreSecret", ctx, "credential", mock.MatchedBy(func(c models.Credential) bool {
			return c.Name == "test_cred" && len(c.Login) > 0 && len(c.Password) > 0
		})).Return(nil)

		err := service.StoreSecret(ctx, "credential", credential)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("store card success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		card := models.Card{
			UserUUID:    userUUID,
			CardNumber:  []byte("1234567890123456"),
			Owner:       []byte("John Doe"),
			ExpiresAt:   []byte("12/25"),
			CVC:         []byte("123"),
			Name:        "test_card",
			Description: "test description",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("StoreSecret", ctx, "card", mock.MatchedBy(func(c models.Card) bool {
			return c.Name == "test_card" && len(c.CardNumber) > 0
		})).Return(nil)

		err := service.StoreSecret(ctx, "card", card)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("store file success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		file := models.File{
			UserUUID:    userUUID,
			UserFile:    []byte("file content"),
			Name:        "test_file",
			Description: "test description",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("StoreSecret", ctx, "file", mock.MatchedBy(func(f models.File) bool {
			return f.Name == "test_file" && len(f.UserFile) > 0
		})).Return(nil)

		err := service.StoreSecret(ctx, "file", file)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid secret type", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		err := service.StoreSecret(ctx, "invalid", "data")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong type assertion", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		err := service.StoreSecret(ctx, "text", "wrong_type")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})

	t.Run("text too long", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		longText := strings.Repeat("a", 1001)
		text := models.Text{
			UserUUID: userUUID,
			UserText: []byte(longText),
			Name:     "test_text",
		}

		err := service.StoreSecret(ctx, "text", text)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "more than 1000 characters")
		mockRepo.AssertExpectations(t)
	})

	t.Run("missing name", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		text := models.Text{
			UserUUID: userUUID,
			UserText: []byte("text"),
			Name:     "",
		}

		err := service.StoreSecret(ctx, "text", text)
		assert.ErrorIs(t, err, apperrors.ErrNameRequired)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_DecryptSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("decrypt text success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		originalText := "my secret text"
		encryptedText, _ := service.encryptString(originalText, userKey)

		storedText := models.Text{
			UserUUID: userUUID,
			UserText: encryptedText,
			Name:     "test_text",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("GetSecret", ctx, userUUID, "text", "test_text").Return(storedText, nil)

		result, err := service.DecryptSecret(ctx, "text", userUUID, "test_text")
		assert.NoError(t, err)
		text, ok := result.(models.Text)
		assert.True(t, ok)
		assert.Equal(t, originalText, string(text.UserText))
		mockRepo.AssertExpectations(t)
	})

	t.Run("decrypt credential success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		originalLogin := "mylogin"
		originalPassword := "mypassword"
		encryptedLogin, _ := service.encryptString(originalLogin, userKey)
		encryptedPassword, _ := service.encryptString(originalPassword, userKey)

		storedCred := models.Credential{
			UserUUID: userUUID,
			Login:    encryptedLogin,
			Password: encryptedPassword,
			Name:     "test_cred",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("GetSecret", ctx, userUUID, "credential", "test_cred").Return(storedCred, nil)

		result, err := service.DecryptSecret(ctx, "credential", userUUID, "test_cred")
		assert.NoError(t, err)
		cred, ok := result.(models.Credential)
		assert.True(t, ok)
		assert.Equal(t, originalLogin, string(cred.Login))
		assert.Equal(t, originalPassword, string(cred.Password))
		mockRepo.AssertExpectations(t)
	})

	t.Run("decrypt card success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		originalCardNumber := "1234567890123456"
		originalOwner := "John Doe"
		originalExpires := "12/25"
		originalCVC := "123"

		encryptedCardNumber, _ := service.encryptString(originalCardNumber, userKey)
		encryptedOwner, _ := service.encryptString(originalOwner, userKey)
		encryptedExpires, _ := service.encryptString(originalExpires, userKey)
		encryptedCVC, _ := service.encryptString(originalCVC, userKey)

		storedCard := models.Card{
			UserUUID:   userUUID,
			CardNumber: encryptedCardNumber,
			Owner:      encryptedOwner,
			ExpiresAt:  encryptedExpires,
			CVC:        encryptedCVC,
			Name:       "test_card",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("GetSecret", ctx, userUUID, "card", "test_card").Return(storedCard, nil)

		result, err := service.DecryptSecret(ctx, "card", userUUID, "test_card")
		assert.NoError(t, err)
		card, ok := result.(models.Card)
		assert.True(t, ok)
		assert.Equal(t, originalCardNumber, string(card.CardNumber))
		assert.Equal(t, originalOwner, string(card.Owner))
		assert.Equal(t, originalExpires, string(card.ExpiresAt))
		assert.Equal(t, originalCVC, string(card.CVC))
		mockRepo.AssertExpectations(t)
	})

	t.Run("decrypt file success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		originalFile := "file content here"
		encryptedFile, _ := service.encryptString(originalFile, userKey)

		storedFile := models.File{
			UserUUID: userUUID,
			UserFile: encryptedFile,
			Name:     "test_file",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("GetSecret", ctx, userUUID, "file", "test_file").Return(storedFile, nil)

		result, err := service.DecryptSecret(ctx, "file", userUUID, "test_file")
		assert.NoError(t, err)
		file, ok := result.(models.File)
		assert.True(t, ok)
		assert.Equal(t, originalFile, string(file.UserFile))
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid secret type", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		result, err := service.DecryptSecret(ctx, "invalid", userUUID, "test")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("secret not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("GetSecret", ctx, userUUID, "text", "nonexistent").Return(models.Text{}, apperrors.ErrNotFound)

		result, err := service.DecryptSecret(ctx, "text", userUUID, "nonexistent")
		assert.ErrorIs(t, err, apperrors.ErrNotFound)
		assert.Equal(t, models.Text{}, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_UpdateSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("update text success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		text := models.Text{
			UserUUID:    userUUID,
			UserText:    []byte("updated text"),
			Name:        "test_text",
			Description: "updated description",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("UpdateSecret", ctx, "text", mock.MatchedBy(func(t models.Text) bool {
			return t.Name == "test_text" && len(t.UserText) > 0
		})).Return(nil)

		err := service.UpdateSecret(ctx, "text", text)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update credential success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		userKey, _ := service.generateUserKey()
		encryptedUserKey, _ := service.encryptKey(userKey)

		credential := models.Credential{
			UserUUID:    userUUID,
			Login:       []byte("updated_login"),
			Password:    []byte("updated_password"),
			Name:        "test_cred",
			Description: "updated description",
		}

		mockRepo.On("GetUserKey", ctx, userUUID).Return(encryptedUserKey, nil)
		mockRepo.On("UpdateSecret", ctx, "credential", mock.MatchedBy(func(c models.Credential) bool {
			return c.Name == "test_cred" && len(c.Login) > 0 && len(c.Password) > 0
		})).Return(nil)

		err := service.UpdateSecret(ctx, "credential", credential)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid secret type", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		err := service.UpdateSecret(ctx, "invalid", "data")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong type assertion", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		err := service.UpdateSecret(ctx, "text", "wrong_type")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_DeleteSecret(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		secretType string
		shouldErr  bool
	}{
		{"delete text", "text", false},
		{"delete credential", "credential", false},
		{"delete card", "card", false},
		{"delete file", "file", false},
		{"invalid type", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := newTestService(mockRepo)

			userUUID := uuid.New()

			if !tt.shouldErr {
				mockRepo.On("DeleteSecret", ctx, userUUID, tt.secretType, "test_name").Return(nil)
			}

			err := service.DeleteSecret(ctx, tt.secretType, userUUID, "test_name")
			if tt.shouldErr {
				assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetAllUserTexts(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		expectedTexts := []models.Text{
			{Name: "text1", UserUUID: userUUID},
			{Name: "text2", UserUUID: userUUID},
		}

		mockRepo.On("GetAllUserTexts", ctx, userUUID).Return(expectedTexts, nil)

		texts, err := service.GetAllUserTexts(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, texts, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil UUID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		texts, err := service.GetAllUserTexts(ctx, uuid.Nil)
		assert.ErrorIs(t, err, apperrors.ErrInvalidUserUUID)
		assert.Nil(t, texts)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		mockRepo.On("GetAllUserTexts", ctx, userUUID).Return(nil, apperrors.ErrNotFound)

		texts, err := service.GetAllUserTexts(ctx, userUUID)
		assert.ErrorIs(t, err, apperrors.ErrNotFound)
		assert.Nil(t, texts)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetAllUserCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		expectedCreds := []models.Credential{
			{Name: "cred1", UserUUID: userUUID},
			{Name: "cred2", UserUUID: userUUID},
		}

		mockRepo.On("GetAllUserCredentials", ctx, userUUID).Return(expectedCreds, nil)

		creds, err := service.GetAllUserCredentials(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, creds, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil UUID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		creds, err := service.GetAllUserCredentials(ctx, uuid.Nil)
		assert.ErrorIs(t, err, apperrors.ErrInvalidUserUUID)
		assert.Nil(t, creds)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetAllUserCards(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		expectedCards := []models.Card{
			{Name: "card1", UserUUID: userUUID},
			{Name: "card2", UserUUID: userUUID},
		}

		mockRepo.On("GetAllUserCards", ctx, userUUID).Return(expectedCards, nil)

		cards, err := service.GetAllUserCards(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, cards, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil UUID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		cards, err := service.GetAllUserCards(ctx, uuid.Nil)
		assert.ErrorIs(t, err, apperrors.ErrInvalidUserUUID)
		assert.Nil(t, cards)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetAllUserFiles(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()
		expectedFiles := []models.File{
			{Name: "file1", UserUUID: userUUID},
			{Name: "file2", UserUUID: userUUID},
		}

		mockRepo.On("GetAllUserFiles", ctx, userUUID).Return(expectedFiles, nil)

		files, err := service.GetAllUserFiles(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, files, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil UUID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		files, err := service.GetAllUserFiles(ctx, uuid.Nil)
		assert.ErrorIs(t, err, apperrors.ErrInvalidUserUUID)
		assert.Nil(t, files)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_encryptDecryptString(t *testing.T) {
	mockRepo := new(MockRepository)
	service := newTestService(mockRepo)

	t.Run("encrypt and decrypt", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := service.generateUserKey()
		require.NoError(t, err)

		originalText := "Hello, World!"
		encrypted, err := service.encryptString(originalText, key)
		assert.NoError(t, err)
		assert.NotEqual(t, originalText, string(encrypted))

		decrypted, err := service.decryptString(encrypted, key)
		assert.NoError(t, err)
		assert.Equal(t, originalText, decrypted)
	})

	t.Run("decrypt with wrong key", func(t *testing.T) {
		key1 := make([]byte, 32)
		key2 := make([]byte, 32)
		key2[0] = 1 // Make it different

		originalText := "Hello, World!"
		encrypted, err := service.encryptString(originalText, key1)
		assert.NoError(t, err)

		_, err = service.decryptString(encrypted, key2)
		assert.Error(t, err)
	})
}

func TestService_encryptDecryptKey(t *testing.T) {
	mockRepo := new(MockRepository)
	service := newTestService(mockRepo)

	t.Run("encrypt and decrypt user key", func(t *testing.T) {
		userKey, err := service.generateUserKey()
		require.NoError(t, err)

		encryptedKey, err := service.encryptKey(userKey)
		assert.NoError(t, err)
		assert.NotEqual(t, userKey, encryptedKey)

		decryptedKey, err := service.decryptKey(encryptedKey)
		assert.NoError(t, err)
		assert.Equal(t, userKey, decryptedKey)
	})

	t.Run("decrypt invalid key", func(t *testing.T) {
		invalidKey := []byte("too_short")
		_, err := service.decryptKey(invalidKey)
		assert.Error(t, err)
	})
}

func TestService_generateUserKey(t *testing.T) {
	mockRepo := new(MockRepository)
	service := newTestService(mockRepo)

	t.Run("generates 32-byte key", func(t *testing.T) {
		key, err := service.generateUserKey()
		assert.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("generates unique keys", func(t *testing.T) {
		key1, err := service.generateUserKey()
		assert.NoError(t, err)

		key2, err := service.generateUserKey()
		assert.NoError(t, err)

		assert.NotEqual(t, key1, key2)
	})
}

func TestService_encryptCredential_validation(t *testing.T) {
	ctx := context.Background()

	t.Run("missing name", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		credential := models.Credential{
			UserUUID: userUUID,
			Login:    []byte("login"),
			Password: []byte("password"),
			Name:     "",
		}

		_, err := service.encryptCredential(ctx, credential)
		assert.ErrorIs(t, err, apperrors.ErrNameRequired)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil login", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		credential := models.Credential{
			UserUUID: userUUID,
			Login:    nil,
			Password: []byte("password"),
			Name:     "test",
		}

		_, err := service.encryptCredential(ctx, credential)
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil password", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		credential := models.Credential{
			UserUUID: userUUID,
			Login:    []byte("login"),
			Password: nil,
			Name:     "test",
		}

		_, err := service.encryptCredential(ctx, credential)
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_encryptCard_validation(t *testing.T) {
	ctx := context.Background()

	t.Run("missing name", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		card := models.Card{
			UserUUID:   userUUID,
			CardNumber: []byte("1234"),
			Owner:      []byte("owner"),
			Name:       "",
		}

		_, err := service.encryptCard(ctx, card)
		assert.ErrorIs(t, err, apperrors.ErrNameRequired)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil owner", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		card := models.Card{
			UserUUID:   userUUID,
			CardNumber: []byte("1234"),
			Owner:      nil,
			Name:       "test",
		}

		_, err := service.encryptCard(ctx, card)
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil card number", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		card := models.Card{
			UserUUID:   userUUID,
			CardNumber: nil,
			Owner:      []byte("owner"),
			Name:       "test",
		}

		_, err := service.encryptCard(ctx, card)
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_encryptFile_validation(t *testing.T) {
	ctx := context.Background()

	t.Run("missing name", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		file := models.File{
			UserUUID: userUUID,
			UserFile: []byte("content"),
			Name:     "",
		}

		err := service.encryptFile(ctx, file)
		assert.ErrorIs(t, err, apperrors.ErrNameRequired)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil file content", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := newTestService(mockRepo)

		userUUID := uuid.New()

		file := models.File{
			UserUUID: userUUID,
			UserFile: nil,
			Name:     "test",
		}

		err := service.encryptFile(ctx, file)
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_newGCM(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		key := make([]byte, 32)
		gcm, err := newGCM(key)
		assert.NoError(t, err)
		assert.NotNil(t, gcm)
	})

	t.Run("invalid key length", func(t *testing.T) {
		key := make([]byte, 15) // Invalid length - AES accepts 16, 24, or 32 bytes
		_, err := newGCM(key)
		assert.Error(t, err)
	})
}

func TestNewService(t *testing.T) {
	mockRepo := new(MockRepository)
	config := config.ServiceConf{MasterKey: "test_key"}
	logger := zerolog.Nop()

	service := NewService(mockRepo, logger, config)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, config, service.conf)
}
