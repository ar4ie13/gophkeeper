package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/ar4ie13/gophkeeper/internal/apperrors"
	"github.com/ar4ie13/gophkeeper/internal/server/models"
	"github.com/ar4ie13/gophkeeper/internal/server/service/config"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// chunkSize defines the maximum chunk size for processing large files.
const chunkSize = 5 * 1024 * 1024 // 5MB per chunk

// Service handles business logic including encryption and secret management.
type Service struct {
	repo Repository
	zlog zerolog.Logger
	conf config.ServiceConf
}

// NewService creates a new Service instance with the provided dependencies.
func NewService(repo Repository, zlog zerolog.Logger, conf config.ServiceConf) *Service {
	return &Service{
		repo: repo,
		zlog: zlog,
		conf: conf,
	}
}

// Repository interface used to communicate with repository from service
type Repository interface {
	CreateUser(ctx context.Context, user models.User) error
	GetUserByLogin(ctx context.Context, login string) (models.User, error)
	StoreSecret(ctx context.Context, secretType string, secret any) error
	GetSecret(ctx context.Context, userUUID uuid.UUID, secretType string, secretName string) (any, error)
	UpdateSecret(ctx context.Context, secretType string, secret any) error
	DeleteSecret(ctx context.Context, userUUID uuid.UUID, secretType string, secretName string) error
	GetUserKey(ctx context.Context, user uuid.UUID) ([]byte, error)
	GetAllUserTexts(ctx context.Context, user uuid.UUID) ([]models.Text, error)
	GetAllUserCredentials(ctx context.Context, user uuid.UUID) ([]models.Credential, error)
	GetAllUserCards(ctx context.Context, user uuid.UUID) ([]models.Card, error)
	GetAllUserFiles(ctx context.Context, user uuid.UUID) ([]models.File, error)
}

// checkLoginString is a helper to validate login string
func (s *Service) checkLoginString(login string) bool {
	// Check that only letters and digits are used for login
	for _, char := range login {
		if !(char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9') {
			return false
		}
	}

	return true
}

// LoginUser validates login format, retrieves user data, and decrypts the user's encryption key.
func (s *Service) LoginUser(ctx context.Context, login string) (models.User, error) {
	if !s.checkLoginString(login) {
		return models.User{}, apperrors.ErrInvalidLoginString
	}

	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return models.User{}, err
	}

	decryptedKey, err := s.decryptKey(user.EncryptedKey)
	if err != nil {
		return models.User{}, err
	}

	user.EncryptedKey = decryptedKey

	return user, nil
}

// CreateUser validates login format, generates a new encryption key, and stores the user.
func (s *Service) CreateUser(ctx context.Context, user models.User) (uuid.UUID, error) {
	if !s.checkLoginString(user.Login) {
		return uuid.Nil, apperrors.ErrInvalidLoginString
	}

	user.UUID = uuid.New()

	newUserKey, err := s.generateUserKey()
	if err != nil {
		return uuid.Nil, err
	}

	user.EncryptedKey, err = s.encryptKey(newUserKey)
	if err != nil {
		return uuid.Nil, err
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return uuid.Nil, err
	}

	return user.UUID, nil
}

// StoreSecret encrypts and stores a secret after validating its type.
func (s *Service) StoreSecret(ctx context.Context, secretType string, secret any) error {
	switch secretType {
	case "text":
		text, ok := secret.(models.Text)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		encryptedText, err := s.encryptText(ctx, text)
		if err != nil {
			return err
		}
		return s.repo.StoreSecret(ctx, "text", encryptedText)

	case "credential":
		credential, ok := secret.(models.Credential)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		encryptedCred, err := s.encryptCredential(ctx, credential)
		if err != nil {
			return err
		}
		return s.repo.StoreSecret(ctx, "credential", encryptedCred)
	case "card":
		card, ok := secret.(models.Card)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		encryptedCard, err := s.encryptCard(ctx, card)
		if err != nil {
			return err
		}
		return s.repo.StoreSecret(ctx, "card", encryptedCard)
	case "file":
		file, ok := secret.(models.File)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return s.encryptFile(ctx, file)
	default:
		return apperrors.ErrSecretInvalid
	}
}

// DecryptSecret retrieves and decrypts a secret by type and name.
func (s *Service) DecryptSecret(ctx context.Context, secretType string, userUUID uuid.UUID, name string) (any, error) {
	switch secretType {
	case "text":
		return s.decryptText(ctx, userUUID, name)
	case "credential":
		return s.decryptCredential(ctx, userUUID, name)
	case "card":
		return s.decryptCard(ctx, userUUID, name)
	case "file":
		return s.decryptFile(ctx, userUUID, name)
	default:
		return nil, apperrors.ErrSecretInvalid
	}
}

// UpdateSecret re-encrypts and updates an existing secret.
func (s *Service) UpdateSecret(ctx context.Context, secretType string, secret any) error {
	switch secretType {
	case "text":
		text, ok := secret.(models.Text)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		encryptedText, err := s.encryptText(ctx, text)
		if err != nil {
			return err
		}
		return s.repo.UpdateSecret(ctx, "text", encryptedText)

	case "credential":
		credential, ok := secret.(models.Credential)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		encryptedCred, err := s.encryptCredential(ctx, credential)
		if err != nil {
			return err
		}
		return s.repo.UpdateSecret(ctx, "credential", encryptedCred)
	case "card":
		card, ok := secret.(models.Card)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		encryptedCard, err := s.encryptCard(ctx, card)
		if err != nil {
			return err
		}
		return s.repo.UpdateSecret(ctx, "card", encryptedCard)

	default:
		return apperrors.ErrSecretInvalid
	}
}

// DeleteSecret removes a secret by type, user, and name.
func (s *Service) DeleteSecret(ctx context.Context, secretType string, userUUID uuid.UUID, name string) error {
	if secretType == "text" || secretType == "credential" || secretType == "card" || secretType == "file" {
		err := s.repo.DeleteSecret(ctx, userUUID, secretType, name)
		if err != nil {
			return err
		}
		return nil
	}
	return apperrors.ErrSecretInvalid
}

// encryptKey encrypts a user key using the master key via AES-GCM.
func (s *Service) encryptKey(userKey []byte) ([]byte, error) {

	gcm, err := newGCM([]byte(s.conf.MasterKey))
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal encrypts and authenticates the userKey
	ciphertext := gcm.Seal(nonce, nonce, userKey, nil)
	return ciphertext, nil
}

// decryptKey decrypts a user key that was encrypted with the master key.
func (s *Service) decryptKey(encryptedKey []byte) ([]byte, error) {

	gcm, err := newGCM([]byte(s.conf.MasterKey))
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedKey) < nonceSize {
		return nil, errors.New("wrapped key too short")
	}

	nonce, ciphertext := encryptedKey[:nonceSize], encryptedKey[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// generateUserKey generates a random 32-byte encryption key for a new user.
func (s *Service) generateUserKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// encryptText validates and encrypts text data (max 1000 characters).
func (s *Service) encryptText(ctx context.Context, text models.Text) (models.Text, error) {
	if utf8.RuneCountInString(string(text.UserText)) > 1000 {
		return models.Text{}, errors.New("text is more than 1000 characters")
	}

	if text.Name == "" {
		return models.Text{}, apperrors.ErrNameRequired
	}

	userKey, err := s.getUserKey(ctx, text.UserUUID)
	if err != nil {
		return models.Text{}, err
	}

	text.UserText, err = s.encryptString(string(text.UserText), userKey)
	if err != nil {
		return models.Text{}, err
	}

	return text, nil
}

// decryptText retrieves and decrypts text data by user and name.
func (s *Service) decryptText(ctx context.Context, user uuid.UUID, name string) (models.Text, error) {
	userKey, err := s.getUserKey(ctx, user)
	if err != nil {
		return models.Text{}, err
	}

	get, err := s.repo.GetSecret(ctx, user, "text", name)
	if err != nil {
		return models.Text{}, err
	}
	text, ok := get.(models.Text)
	if !ok {
		return models.Text{}, apperrors.ErrSecretInvalid
	}

	decryptedText, err := s.decryptString(text.UserText, userKey)
	if err != nil {
		return models.Text{}, err
	}
	text.UserText = []byte(decryptedText)

	return text, nil
}

// encryptCredential validates and encrypts login credentials.
func (s *Service) encryptCredential(ctx context.Context, credential models.Credential) (models.Credential, error) {

	if credential.Name == "" {
		return models.Credential{}, apperrors.ErrNameRequired
	}

	if credential.Login == nil || credential.Password == nil {
		return models.Credential{}, apperrors.ErrSecretInvalid
	}

	userKey, err := s.getUserKey(ctx, credential.UserUUID)
	if err != nil {
		return models.Credential{}, err
	}

	credential.Login, err = s.encryptString(string(credential.Login), userKey)
	if err != nil {
		return models.Credential{}, err
	}
	credential.Password, err = s.encryptString(string(credential.Password), userKey)
	if err != nil {
		return models.Credential{}, err
	}

	return credential, nil
}

// decryptCredential retrieves and decrypts credentials by user and name.
func (s *Service) decryptCredential(ctx context.Context, user uuid.UUID, name string) (models.Credential, error) {
	userKey, err := s.getUserKey(ctx, user)
	if err != nil {
		return models.Credential{}, err
	}

	var credential models.Credential
	get, err := s.repo.GetSecret(ctx, user, "credential", name)
	if err != nil {
		return models.Credential{}, err
	}

	credential, ok := get.(models.Credential)
	if !ok {
		return models.Credential{}, apperrors.ErrSecretInvalid
	}
	decryptedLogin, err := s.decryptString(credential.Login, userKey)
	if err != nil {
		return models.Credential{}, err
	}
	credential.Login = []byte(decryptedLogin)

	decryptedPassword, err := s.decryptString(credential.Password, userKey)
	if err != nil {
		return models.Credential{}, err
	}
	credential.Password = []byte(decryptedPassword)

	return credential, nil
}

// encryptFile validates, encrypts, and stores file data.
func (s *Service) encryptFile(ctx context.Context, file models.File) error {

	if file.Name == "" {
		return apperrors.ErrNameRequired
	}

	if file.UserFile == nil {
		return apperrors.ErrSecretInvalid
	}

	userKey, err := s.getUserKey(ctx, file.UserUUID)
	if err != nil {
		return err
	}

	file.UserFile, err = s.encryptString(string(file.UserFile), userKey)
	if err != nil {
		return err
	}

	err = s.repo.StoreSecret(ctx, "file", file)
	if err != nil {
		return err
	}

	return nil
}

// decryptFile retrieves and decrypts file data by user and name.
func (s *Service) decryptFile(ctx context.Context, user uuid.UUID, name string) (models.File, error) {
	userKey, err := s.getUserKey(ctx, user)
	if err != nil {
		return models.File{}, err
	}

	var file models.File
	get, err := s.repo.GetSecret(ctx, user, "file", name)
	if err != nil {
		return models.File{}, err
	}

	file, ok := get.(models.File)
	if !ok {
		return models.File{}, apperrors.ErrSecretInvalid
	}
	decryptedFile, err := s.decryptString(file.UserFile, userKey)
	if err != nil {
		return models.File{}, err
	}
	file.UserFile = []byte(decryptedFile)

	return file, nil
}

// encryptCard validates and encrypts bank card information.
func (s *Service) encryptCard(ctx context.Context, card models.Card) (models.Card, error) {

	if card.Name == "" {
		return models.Card{}, apperrors.ErrNameRequired
	}

	if card.Owner == nil || card.CardNumber == nil {
		return models.Card{}, apperrors.ErrSecretInvalid
	}

	userKey, err := s.getUserKey(ctx, card.UserUUID)
	if err != nil {
		return models.Card{}, err
	}

	card.Owner, err = s.encryptString(string(card.Owner), userKey)
	if err != nil {
		return models.Card{}, err
	}
	card.CardNumber, err = s.encryptString(string(card.CardNumber), userKey)
	if err != nil {
		return models.Card{}, err
	}
	card.ExpiresAt, err = s.encryptString(string(card.ExpiresAt), userKey)
	if err != nil {
		return models.Card{}, err
	}
	card.CVC, err = s.encryptString(string(card.CVC), userKey)
	if err != nil {
		return models.Card{}, err
	}

	return card, nil
}

// decryptCard retrieves and decrypts card data by user and name.
func (s *Service) decryptCard(ctx context.Context, user uuid.UUID, name string) (models.Card, error) {
	userKey, err := s.getUserKey(ctx, user)
	if err != nil {
		return models.Card{}, err
	}

	var card models.Card
	get, err := s.repo.GetSecret(ctx, user, "card", name)
	if err != nil {
		return models.Card{}, err
	}

	card, ok := get.(models.Card)
	if !ok {
		return models.Card{}, apperrors.ErrSecretInvalid
	}
	decryptedCardNumber, err := s.decryptString(card.CardNumber, userKey)
	if err != nil {
		return models.Card{}, err
	}
	card.CardNumber = []byte(decryptedCardNumber)

	decryptedOwner, err := s.decryptString(card.Owner, userKey)
	if err != nil {
		return models.Card{}, err
	}
	card.Owner = []byte(decryptedOwner)

	decryptedExpiresAt, err := s.decryptString(card.ExpiresAt, userKey)
	if err != nil {
		return models.Card{}, err
	}
	card.ExpiresAt = []byte(decryptedExpiresAt)

	decryptedCVC, err := s.decryptString(card.CVC, userKey)
	if err != nil {
		return models.Card{}, err
	}
	card.CVC = []byte(decryptedCVC)

	return card, nil
}

// getUserKey retrieves and decrypts a user's encryption key.
func (s *Service) getUserKey(ctx context.Context, user uuid.UUID) ([]byte, error) {
	userKey, err := s.repo.GetUserKey(ctx, user)
	if err != nil {
		return nil, err
	}
	decryptedKey, err := s.decryptKey(userKey)
	if err != nil {
		return nil, err
	}
	return decryptedKey, nil
}

// encryptString encrypts a string using AES-GCM with the provided key.
func (s *Service) encryptString(text string, key []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return ciphertext, nil
}

// decryptString decrypts ciphertext using AES-GCM with the provided key.
func (s *Service) decryptString(ciphertext []byte, key []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	nonce, text := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, text, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GetAllUserTexts retrieves all text secrets for a user (metadata only).
func (s *Service) GetAllUserTexts(ctx context.Context, user uuid.UUID) ([]models.Text, error) {
	if user == uuid.Nil {
		return nil, apperrors.ErrInvalidUserUUID
	}
	return s.repo.GetAllUserTexts(ctx, user)
}

// GetAllUserCredentials retrieves all credentials for a user (metadata only).
func (s *Service) GetAllUserCredentials(ctx context.Context, user uuid.UUID) ([]models.Credential, error) {
	if user == uuid.Nil {
		return nil, apperrors.ErrInvalidUserUUID
	}
	return s.repo.GetAllUserCredentials(ctx, user)
}

// GetAllUserCards retrieves all cards for a user (metadata only).
func (s *Service) GetAllUserCards(ctx context.Context, user uuid.UUID) ([]models.Card, error) {
	if user == uuid.Nil {
		return nil, apperrors.ErrInvalidUserUUID
	}
	return s.repo.GetAllUserCards(ctx, user)
}

// GetAllUserFiles retrieves all files for a user (metadata only).
func (s *Service) GetAllUserFiles(ctx context.Context, user uuid.UUID) ([]models.File, error) {
	if user == uuid.Nil {
		return nil, apperrors.ErrInvalidUserUUID
	}
	return s.repo.GetAllUserFiles(ctx, user)
}

// newGCM creates a new AES-GCM cipher for authenticated encryption.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
