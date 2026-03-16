package service

import (
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/ar4ie13/gophkeeper/internal/client/api"
	"github.com/ar4ie13/gophkeeper/internal/client/models"
	"github.com/ar4ie13/gophkeeper/internal/client/storage"
)

// Service orchestrates operations between the API client and the local
// cache. It is the single source of truth for the TUI layer.
type Service struct {
	api   *api.Client
	cache *storage.Cache
}

// NewService creates a Service.
func NewService(apiClient *api.Client, cache *storage.Cache) *Service {
	return &Service{
		api:   apiClient,
		cache: cache,
	}
}

// Register creates a new user account and authenticates the session.
func (s *Service) Register(login, password string) error {
	if login == "" || password == "" {
		return fmt.Errorf("login and password are required")
	}
	return s.api.Register(login, password)
}

// Login authenticates an existing user.
func (s *Service) Login(login, password string) error {
	if login == "" || password == "" {
		return fmt.Errorf("login and password are required")
	}
	return s.api.Login(login, password)
}

// SyncResult summarizes the outcome of a sync operation including counts and errors.
type SyncResult struct {
	TextCount       int
	CredentialCount int
	CardCount       int
	FilesCount      int
	Online          bool
	Error           error
}

// Sync pulls every secret from the server into the local cache.
// If the server is unreachable the cache still contains previously synced data.
func (s *Service) Sync() SyncResult {
	result := SyncResult{Online: true}

	if err := s.api.Ping(); err != nil {
		return SyncResult{Online: false, Error: err}
	}

	// --- texts ---
	textList, err := s.api.ListTexts()
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return result
		default:
			result.Error = fmt.Errorf("list texts: %w", err)
			return result
		}
	}
	for _, item := range *textList {
		t, err := s.api.GetText(item.Name)
		if err != nil {
			continue // skip individual failures
		}
		s.cache.SetText(*t)
		result.TextCount++
	}

	// --- credentials ---
	credList, err := s.api.ListCredentials()
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return result
		default:
			result.Error = fmt.Errorf("list credentials: %w", err)
			return result
		}
	}
	for _, item := range *credList {
		c, err := s.api.GetCredential(item.Name)
		if err != nil {
			continue
		}
		s.cache.SetCredential(*c)
		result.CredentialCount++
	}

	// --- cards ---
	cardList, err := s.api.ListCards()
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return result
		default:
			result.Error = fmt.Errorf("list cards: %w", err)
			return result
		}
	}
	for _, item := range *cardList {
		c, err := s.api.GetCard(item.Name)
		if err != nil {
			continue
		}
		s.cache.SetCard(*c)
		result.CardCount++
	}

	// --- files ---
	fileList, err := s.api.ListFiles()
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return result
		default:
			result.Error = fmt.Errorf("list files: %w", err)
			return result
		}
	}
	for _, item := range *fileList {
		c, err := s.api.GetFile(item.Name)
		if err != nil {
			continue
		}
		s.cache.SetFile(*c)
		result.FilesCount++
	}

	return result
}

// ListSecrets returns all cached entries sorted by name.
func (s *Service) ListSecrets() []models.SecretEntry {
	entries := s.cache.AllEntries()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type < entries[j].Type
		}
		return entries[i].Name < entries[j].Name
	})
	return entries
}

// GetText returns a text secret, trying the server first then falling back to
// the local cache.
func (s *Service) GetText(name string) (*models.TextSecret, error) {
	t, err := s.api.GetText(name)
	if err == nil {
		s.cache.SetText(*t)
		return t, nil
	}
	cached, ok := s.cache.GetText(name)
	if ok {
		return &cached, nil
	}
	return nil, fmt.Errorf("text secret %q unavailable (server offline, not in cache)", name)
}

// GetCredential returns a credential secret, server-first then cache fallback.
func (s *Service) GetCredential(name string) (*models.CredentialSecret, error) {
	c, err := s.api.GetCredential(name)
	if err == nil {
		s.cache.SetCredential(*c)
		return c, nil
	}
	cached, ok := s.cache.GetCredential(name)
	if ok {
		return &cached, nil
	}
	return nil, fmt.Errorf("credential %q unavailable (server offline, not in cache)", name)
}

// GetCard returns a card secret, server-first then cache fallback.
func (s *Service) GetCard(name string) (*models.CardSecret, error) {
	c, err := s.api.GetCard(name)
	if err == nil {
		s.cache.SetCard(*c)
		return c, nil
	}
	cached, ok := s.cache.GetCard(name)
	if ok {
		return &cached, nil
	}
	return nil, fmt.Errorf("card %q unavailable (server offline, not in cache)", name)
}

// GetFile returns a file secret, server-first then cache fallback.
func (s *Service) GetFile(name string) (*models.FileSecret, error) {
	c, err := s.api.GetFile(name)
	if err == nil {
		s.cache.SetFile(*c)
		return c, nil
	}
	cached, ok := s.cache.GetFile(name)
	if ok {
		return &cached, nil
	}
	return nil, fmt.Errorf("file %q unavailable (server offline, not in cache)", name)
}

// StoreText sends a text secret to the server and caches it locally.
func (s *Service) StoreText(secret models.TextSecret) error {
	if len(secret.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(secret.Text) > 1000 {
		return fmt.Errorf("text exceeds 1000 character limit (%d chars)", len(secret.Text))
	}

	if err := s.api.StoreText(secret); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	s.cache.SetText(secret)
	return nil
}

// StoreCredential sends a credential secret to the server and caches it.
func (s *Service) StoreCredential(secret models.CredentialSecret) error {
	if len(secret.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(secret.Login) == 0 || len(secret.Password) == 0 {
		return fmt.Errorf("login and password are required")
	}

	if err := s.api.StoreCredential(secret); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	s.cache.SetCredential(secret)
	return nil
}

// StoreCard sends a card secret to the server and caches it.
func (s *Service) StoreCard(secret models.CardSecret) error {
	if len(secret.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(secret.CardNumber) == 0 || len(secret.Owner) == 0 {
		return fmt.Errorf("card number and owner are required")
	}

	if err := s.api.StoreCard(secret); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	s.cache.SetCard(secret)
	return nil
}

// StoreFile sends a file secret to the server and caches it.
func (s *Service) StoreFile(secret models.FileSecret) error {
	if len(secret.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(secret.Path) == 0 {
		return fmt.Errorf("file path is required")
	}

	if err := s.api.StoreFile(secret); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	s.cache.SetFile(secret)
	return nil
}

// UpdateText sends a text secret to the server and caches it locally.
func (s *Service) UpdateText(secret models.TextSecret) error {
	if len(secret.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(secret.Text) > 1000 {
		return fmt.Errorf("text exceeds 1000 character limit (%d chars)", len(secret.Text))
	}

	if err := s.api.UpdateText(secret); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	s.cache.SetText(secret)
	return nil
}

// UpdateCredential sends a credential secret to the server and caches it.
func (s *Service) UpdateCredential(secret models.CredentialSecret) error {
	if len(secret.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(secret.Login) == 0 || len(secret.Password) == 0 {
		return fmt.Errorf("login and password are required")
	}

	if err := s.api.UpdateCredential(secret); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	s.cache.SetCredential(secret)
	return nil
}

// UpdateCard sends a card secret to the server and caches it.
func (s *Service) UpdateCard(secret models.CardSecret) error {
	if len(secret.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(secret.CardNumber) == 0 || len(secret.Owner) == 0 {
		return fmt.Errorf("card number and owner are required")
	}

	if err := s.api.UpdateCard(secret); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	s.cache.SetCard(secret)
	return nil
}

// DeleteSecret deletes secret, server-first then cache fallback.
func (s *Service) DeleteSecret(name string, secretType string) error {
	err := s.api.DeleteSecret(name, secretType)
	if err == nil {
		s.cache.DeleteSecret(name, secretType)
		return nil
	}
	return fmt.Errorf("cannot delete secret: %w", err)
}
