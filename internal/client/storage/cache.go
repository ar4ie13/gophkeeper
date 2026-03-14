package storage

import (
	"os"
	"sync"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/client/models"
)

// Cache holds secrets exclusively in RAM. Nothing is written to disk.
// When the process exits, all cached data is lost — secrets must be
// re-synced from the server on the next startup.
type Cache struct {
	mu          sync.RWMutex
	texts       map[string]models.TextSecret
	credentials map[string]models.CredentialSecret
	cards       map[string]models.CardSecret
	files       map[string]models.FileSecret
	entries     map[string]models.SecretEntry
}

// NewCache creates an empty in-memory cache.
func NewCache() *Cache {
	return &Cache{
		texts:       make(map[string]models.TextSecret),
		credentials: make(map[string]models.CredentialSecret),
		cards:       make(map[string]models.CardSecret),
		files:       make(map[string]models.FileSecret),
		entries:     make(map[string]models.SecretEntry),
	}
}

// SetText stores a text secret in memory.
func (c *Cache) SetText(s models.TextSecret) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.texts[s.Name] = s
	c.entries[textKey(s.Name)] = models.SecretEntry{
		Name:        s.Name,
		Description: s.Description,
		Type:        models.SecretTypeText,
		SyncedAt:    time.Now(),
	}
}

// GetText retrieves a text secret from memory.
func (c *Cache) GetText(name string) (models.TextSecret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.texts[name]
	return s, ok
}

// SetCredential stores a credential secret in memory.
func (c *Cache) SetCredential(s models.CredentialSecret) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.credentials[s.Name] = s
	c.entries[credKey(s.Name)] = models.SecretEntry{
		Name:        s.Name,
		Description: s.Description,
		Type:        models.SecretTypeCredential,
		SyncedAt:    time.Now(),
	}
}

// GetCredential retrieves a credential secret from memory.
func (c *Cache) GetCredential(name string) (models.CredentialSecret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.credentials[name]
	return s, ok
}

// SetCard stores a card secret in memory.
func (c *Cache) SetCard(s models.CardSecret) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cards[s.Name] = s
	c.entries[cardKey(s.Name)] = models.SecretEntry{
		Name:        s.Name,
		Description: s.Description,
		Type:        models.SecretTypeCard,
		SyncedAt:    time.Now(),
	}
}

// GetCard retrieves a card secret from memory.
func (c *Cache) GetCard(name string) (models.CardSecret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.cards[name]
	return s, ok
}

// SetFile stores a path to file secret in memory.
func (c *Cache) SetFile(s models.FileSecret) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.files[s.Name] = s
	c.entries[fileKey(s.Name)] = models.SecretEntry{
		Name:        s.Name,
		Description: s.Description,
		Type:        models.SecretTypeFile,
		SyncedAt:    time.Now(),
	}
}

// GetFile retrieves a path to file secret from memory.
func (c *Cache) GetFile(name string) (models.FileSecret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.files[name]
	return s, ok
}

// DeleteSecret retrieves a path to file secret from memory.
func (c *Cache) DeleteSecret(name string, secretType string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch secretType {
	case "text":
		delete(c.texts, name)
		delete(c.entries, textKey(name))
	case "credential":
		delete(c.credentials, name)
		delete(c.entries, credKey(name))
	case "card":
		delete(c.cards, name)
		delete(c.entries, cardKey(name))
	case "file":
		os.Remove(c.files[name].Path)
		delete(c.files, name)
		delete(c.entries, fileKey(name))
	}
}

// --- Listing ----------------------------------------------------------------

// AllEntries returns a copy of every cached secret entry.
func (c *Cache) AllEntries() []models.SecretEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]models.SecretEntry, 0, len(c.entries))
	for _, e := range c.entries {
		out = append(out, e)
	}
	return out
}

// --- helpers ----------------------------------------------------------------

// textKey generates a unique cache key for text secrets.
func textKey(name string) string { return "text:" + name }

// credKey generates a unique cache key for credential secrets.
func credKey(name string) string { return "cred:" + name }

// cardKey generates a unique cache key for card secrets.
func cardKey(name string) string { return "card:" + name }

// fileKey generates a unique cache key for file secrets.
func fileKey(name string) string { return "file:" + name }
