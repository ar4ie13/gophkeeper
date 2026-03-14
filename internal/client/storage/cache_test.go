package storage

import (
	"sync"
	"testing"

	"github.com/ar4ie13/gophkeeper/internal/client/models"
	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	t.Run("creates new cache with initialized maps", func(t *testing.T) {
		cache := NewCache()

		assert.NotNil(t, cache)
		assert.NotNil(t, cache.texts)
		assert.NotNil(t, cache.credentials)
		assert.NotNil(t, cache.cards)
		assert.NotNil(t, cache.files)
		assert.NotNil(t, cache.entries)
		assert.Equal(t, 0, len(cache.texts))
		assert.Equal(t, 0, len(cache.credentials))
		assert.Equal(t, 0, len(cache.cards))
		assert.Equal(t, 0, len(cache.files))
		assert.Equal(t, 0, len(cache.entries))
	})
}

func TestCache_TextOperations(t *testing.T) {
	t.Run("set and get text secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.TextSecret{
			Name:        "test-text",
			Description: "test description",
			Text:        "secret message",
		}

		cache.SetText(secret)
		retrieved, found := cache.GetText("test-text")

		assert.True(t, found)
		assert.Equal(t, secret, retrieved)
	})

	t.Run("set creates entry automatically", func(t *testing.T) {
		cache := NewCache()
		secret := models.TextSecret{
			Name:        "test-text",
			Description: "test description",
			Text:        "secret message",
		}

		cache.SetText(secret)
		entries := cache.AllEntries()

		assert.Len(t, entries, 1)
		assert.Equal(t, "test-text", entries[0].Name)
		assert.Equal(t, "test description", entries[0].Description)
		assert.Equal(t, models.SecretTypeText, entries[0].Type)
		assert.False(t, entries[0].SyncedAt.IsZero())
	})

	t.Run("get non-existent text secret", func(t *testing.T) {
		cache := NewCache()

		retrieved, found := cache.GetText("nonexistent")

		assert.False(t, found)
		assert.Equal(t, models.TextSecret{}, retrieved)
	})

	t.Run("overwrite existing text secret", func(t *testing.T) {
		cache := NewCache()
		original := models.TextSecret{
			Name:        "test-text",
			Description: "original description",
			Text:        "original message",
		}
		updated := models.TextSecret{
			Name:        "test-text",
			Description: "updated description",
			Text:        "updated message",
		}

		cache.SetText(original)
		cache.SetText(updated)
		retrieved, found := cache.GetText("test-text")

		assert.True(t, found)
		assert.Equal(t, updated, retrieved)
		assert.NotEqual(t, original, retrieved)
	})

	t.Run("different names are different secrets", func(t *testing.T) {
		cache := NewCache()
		secret1 := models.TextSecret{Name: "name1", Text: "text1"}
		secret2 := models.TextSecret{Name: "name2", Text: "text2"}

		cache.SetText(secret1)
		cache.SetText(secret2)

		retrieved1, found1 := cache.GetText("name1")
		retrieved2, found2 := cache.GetText("name2")

		assert.True(t, found1)
		assert.True(t, found2)
		assert.Equal(t, secret1, retrieved1)
		assert.Equal(t, secret2, retrieved2)
	})
}

func TestCache_CredentialOperations(t *testing.T) {
	t.Run("set and get credential secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.CredentialSecret{
			Name:        "test-cred",
			Description: "test description",
			Login:       "user@example.com",
			Password:    "secretpass",
		}

		cache.SetCredential(secret)
		retrieved, found := cache.GetCredential("test-cred")

		assert.True(t, found)
		assert.Equal(t, secret, retrieved)
	})

	t.Run("set creates entry automatically", func(t *testing.T) {
		cache := NewCache()
		secret := models.CredentialSecret{
			Name:        "test-cred",
			Description: "login credentials",
			Login:       "user",
			Password:    "pass",
		}

		cache.SetCredential(secret)
		entries := cache.AllEntries()

		assert.Len(t, entries, 1)
		assert.Equal(t, "test-cred", entries[0].Name)
		assert.Equal(t, "login credentials", entries[0].Description)
		assert.Equal(t, models.SecretTypeCredential, entries[0].Type)
	})

	t.Run("get non-existent credential secret", func(t *testing.T) {
		cache := NewCache()

		retrieved, found := cache.GetCredential("nonexistent")

		assert.False(t, found)
		assert.Equal(t, models.CredentialSecret{}, retrieved)
	})

	t.Run("overwrite existing credential secret", func(t *testing.T) {
		cache := NewCache()
		original := models.CredentialSecret{
			Name:     "test-cred",
			Login:    "user1",
			Password: "pass1",
		}
		updated := models.CredentialSecret{
			Name:     "test-cred",
			Login:    "user2",
			Password: "pass2",
		}

		cache.SetCredential(original)
		cache.SetCredential(updated)
		retrieved, found := cache.GetCredential("test-cred")

		assert.True(t, found)
		assert.Equal(t, updated, retrieved)
	})
}

func TestCache_CardOperations(t *testing.T) {
	t.Run("set and get card secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.CardSecret{
			Name:        "test-card",
			Description: "test description",
			CardNumber:  "1234567890123456",
			Owner:       "John Doe",
			ExpiresAt:   "12/25",
			CVC:         "123",
		}

		cache.SetCard(secret)
		retrieved, found := cache.GetCard("test-card")

		assert.True(t, found)
		assert.Equal(t, secret, retrieved)
	})

	t.Run("set creates entry automatically", func(t *testing.T) {
		cache := NewCache()
		secret := models.CardSecret{
			Name:        "test-card",
			Description: "my visa card",
			CardNumber:  "4111",
			Owner:       "John",
		}

		cache.SetCard(secret)
		entries := cache.AllEntries()

		assert.Len(t, entries, 1)
		assert.Equal(t, "test-card", entries[0].Name)
		assert.Equal(t, "my visa card", entries[0].Description)
		assert.Equal(t, models.SecretTypeCard, entries[0].Type)
	})

	t.Run("get non-existent card secret", func(t *testing.T) {
		cache := NewCache()

		retrieved, found := cache.GetCard("nonexistent")

		assert.False(t, found)
		assert.Equal(t, models.CardSecret{}, retrieved)
	})

	t.Run("overwrite existing card secret", func(t *testing.T) {
		cache := NewCache()
		original := models.CardSecret{
			Name:       "test-card",
			CardNumber: "1111222233334444",
			Owner:      "Alice",
		}
		updated := models.CardSecret{
			Name:       "test-card",
			CardNumber: "5555666677778888",
			Owner:      "Bob",
		}

		cache.SetCard(original)
		cache.SetCard(updated)
		retrieved, found := cache.GetCard("test-card")

		assert.True(t, found)
		assert.Equal(t, updated, retrieved)
	})
}

func TestCache_FileOperations(t *testing.T) {
	t.Run("set and get file secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.FileSecret{
			Name:        "test-file",
			Description: "test description",
			Path:        "/path/to/document.pdf",
		}

		cache.SetFile(secret)
		retrieved, found := cache.GetFile("test-file")

		assert.True(t, found)
		assert.Equal(t, secret, retrieved)
	})

	t.Run("set creates entry automatically", func(t *testing.T) {
		cache := NewCache()
		secret := models.FileSecret{
			Name:        "test-file",
			Description: "important document",
			Path:        "/tmp/file.txt",
		}

		cache.SetFile(secret)
		entries := cache.AllEntries()

		assert.Len(t, entries, 1)
		assert.Equal(t, "test-file", entries[0].Name)
		assert.Equal(t, "important document", entries[0].Description)
		assert.Equal(t, models.SecretTypeFile, entries[0].Type)
	})

	t.Run("get non-existent file secret", func(t *testing.T) {
		cache := NewCache()

		retrieved, found := cache.GetFile("nonexistent")

		assert.False(t, found)
		assert.Equal(t, models.FileSecret{}, retrieved)
	})

	t.Run("overwrite existing file secret", func(t *testing.T) {
		cache := NewCache()
		original := models.FileSecret{
			Name: "test-file",
			Path: "/path/to/file1.txt",
		}
		updated := models.FileSecret{
			Name: "test-file",
			Path: "/path/to/file2.txt",
		}

		cache.SetFile(original)
		cache.SetFile(updated)
		retrieved, found := cache.GetFile("test-file")

		assert.True(t, found)
		assert.Equal(t, updated, retrieved)
	})

	t.Run("handles empty path", func(t *testing.T) {
		cache := NewCache()
		secret := models.FileSecret{
			Name: "empty-file",
			Path: "",
		}

		cache.SetFile(secret)
		retrieved, found := cache.GetFile("empty-file")

		assert.True(t, found)
		assert.Equal(t, "", retrieved.Path)
	})
}

func TestCache_DeleteSecret(t *testing.T) {
	t.Run("delete text secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.TextSecret{Name: "test-text", Text: "text"}

		cache.SetText(secret)
		cache.DeleteSecret("test-text", "text")

		_, found := cache.GetText("test-text")
		assert.False(t, found)

		entries := cache.AllEntries()
		assert.Len(t, entries, 0)
	})

	t.Run("delete credential secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.CredentialSecret{Name: "test-cred", Login: "user"}

		cache.SetCredential(secret)
		cache.DeleteSecret("test-cred", "credential")

		_, found := cache.GetCredential("test-cred")
		assert.False(t, found)

		entries := cache.AllEntries()
		assert.Len(t, entries, 0)
	})

	t.Run("delete card secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.CardSecret{Name: "test-card", CardNumber: "1234"}

		cache.SetCard(secret)
		cache.DeleteSecret("test-card", "card")

		_, found := cache.GetCard("test-card")
		assert.False(t, found)

		entries := cache.AllEntries()
		assert.Len(t, entries, 0)
	})

	t.Run("delete file secret", func(t *testing.T) {
		cache := NewCache()
		secret := models.FileSecret{Name: "test-file", Path: "/nonexistent/path.txt"}

		cache.SetFile(secret)
		cache.DeleteSecret("test-file", "file")

		_, found := cache.GetFile("test-file")
		assert.False(t, found)

		entries := cache.AllEntries()
		assert.Len(t, entries, 0)
	})

	t.Run("delete non-existent secret does not panic", func(t *testing.T) {
		cache := NewCache()

		assert.NotPanics(t, func() {
			cache.DeleteSecret("nonexistent", "text")
			cache.DeleteSecret("nonexistent", "credential")
			cache.DeleteSecret("nonexistent", "card")
			cache.DeleteSecret("nonexistent", "file")
		})
	})

	t.Run("delete only removes specified secret", func(t *testing.T) {
		cache := NewCache()
		text1 := models.TextSecret{Name: "text1", Text: "content1"}
		text2 := models.TextSecret{Name: "text2", Text: "content2"}

		cache.SetText(text1)
		cache.SetText(text2)

		cache.DeleteSecret("text1", "text")

		_, found1 := cache.GetText("text1")
		_, found2 := cache.GetText("text2")

		assert.False(t, found1)
		assert.True(t, found2)
	})

	t.Run("delete with unknown type does nothing", func(t *testing.T) {
		cache := NewCache()
		secret := models.TextSecret{Name: "test-text", Text: "text"}

		cache.SetText(secret)
		cache.DeleteSecret("test-text", "unknown-type")

		// Secret should still exist
		_, found := cache.GetText("test-text")
		assert.True(t, found)
	})
}

func TestCache_AllEntries(t *testing.T) {
	t.Run("returns all entries", func(t *testing.T) {
		cache := NewCache()
		cache.SetText(models.TextSecret{Name: "text1", Description: "desc1"})
		cache.SetCredential(models.CredentialSecret{Name: "cred1", Description: "desc2"})
		cache.SetCard(models.CardSecret{Name: "card1", Description: "desc3"})
		cache.SetFile(models.FileSecret{Name: "file1", Description: "desc4"})

		entries := cache.AllEntries()

		assert.Len(t, entries, 4)

		// Verify all types are present
		types := make(map[models.SecretType]bool)
		for _, e := range entries {
			types[e.Type] = true
		}
		assert.True(t, types[models.SecretTypeText])
		assert.True(t, types[models.SecretTypeCredential])
		assert.True(t, types[models.SecretTypeCard])
		assert.True(t, types[models.SecretTypeFile])
	})

	t.Run("returns empty slice when no entries", func(t *testing.T) {
		cache := NewCache()

		entries := cache.AllEntries()

		assert.NotNil(t, entries)
		assert.Len(t, entries, 0)
	})

	t.Run("returns independent copies", func(t *testing.T) {
		cache := NewCache()
		cache.SetText(models.TextSecret{Name: "entry1", Description: "original"})

		entries1 := cache.AllEntries()
		entries2 := cache.AllEntries()

		// Modifying one slice should not affect the other
		if len(entries1) > 0 {
			entries1[0].Description = "modified"
		}

		assert.NotEqual(t, entries1[0].Description, entries2[0].Description)
	})

	t.Run("entries have correct metadata", func(t *testing.T) {
		cache := NewCache()
		cache.SetText(models.TextSecret{
			Name:        "test",
			Description: "test description",
		})

		entries := cache.AllEntries()

		assert.Len(t, entries, 1)
		assert.Equal(t, "test", entries[0].Name)
		assert.Equal(t, "test description", entries[0].Description)
		assert.Equal(t, models.SecretTypeText, entries[0].Type)
		assert.False(t, entries[0].SyncedAt.IsZero())
	})
}

func TestCache_KeyGenerators(t *testing.T) {
	t.Run("textKey generates correct key", func(t *testing.T) {
		key := textKey("test-name")
		assert.Equal(t, "text:test-name", key)
	})

	t.Run("credKey generates correct key", func(t *testing.T) {
		key := credKey("cred-name")
		assert.Equal(t, "cred:cred-name", key)
	})

	t.Run("cardKey generates correct key", func(t *testing.T) {
		key := cardKey("card-name")
		assert.Equal(t, "card:card-name", key)
	})

	t.Run("fileKey generates correct key", func(t *testing.T) {
		key := fileKey("file-name")
		assert.Equal(t, "file:file-name", key)
	})

	t.Run("different types same name produce different keys", func(t *testing.T) {
		name := "test"

		textK := textKey(name)
		credK := credKey(name)
		cardK := cardKey(name)
		fileK := fileKey(name)

		assert.NotEqual(t, textK, credK)
		assert.NotEqual(t, textK, cardK)
		assert.NotEqual(t, textK, fileK)
		assert.NotEqual(t, credK, cardK)
		assert.NotEqual(t, credK, fileK)
		assert.NotEqual(t, cardK, fileK)
	})

	t.Run("keys with special characters", func(t *testing.T) {
		specialName := "test-name_123@domain.com"
		key := textKey(specialName)
		assert.Contains(t, key, specialName)
		assert.Equal(t, "text:test-name_123@domain.com", key)
	})
}

func TestCache_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent text operations", func(t *testing.T) {
		cache := NewCache()
		var wg sync.WaitGroup

		// Concurrent writes
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				secret := models.TextSecret{
					Name: "concurrent-text",
					Text: "content",
				}
				cache.SetText(secret)
			}(i)
		}

		// Concurrent reads
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.GetText("concurrent-text")
			}()
		}

		wg.Wait()
		// If we get here without data races, the test passes
	})

	t.Run("concurrent operations across types", func(t *testing.T) {
		cache := NewCache()
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(4)

			go func(idx int) {
				defer wg.Done()
				cache.SetText(models.TextSecret{Name: "text", Text: "data"})
			}(i)

			go func(idx int) {
				defer wg.Done()
				cache.SetCredential(models.CredentialSecret{Name: "cred", Login: "user"})
			}(i)

			go func(idx int) {
				defer wg.Done()
				cache.SetCard(models.CardSecret{Name: "card", CardNumber: "1234"})
			}(i)

			go func(idx int) {
				defer wg.Done()
				cache.SetFile(models.FileSecret{Name: "file", Path: "/test.txt"})
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent delete and read", func(t *testing.T) {
		cache := NewCache()
		var wg sync.WaitGroup

		// Populate cache with different names
		for i := 0; i < 100; i++ {
			cache.SetText(models.TextSecret{Name: "test", Text: "content"})
		}

		// Concurrent deletes and reads
		for i := 0; i < 100; i++ {
			wg.Add(2)

			go func() {
				defer wg.Done()
				cache.DeleteSecret("test", "text")
			}()

			go func() {
				defer wg.Done()
				cache.GetText("test")
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent AllEntries calls", func(t *testing.T) {
		cache := NewCache()
		var wg sync.WaitGroup

		// Populate with entries
		for i := 0; i < 10; i++ {
			cache.SetText(models.TextSecret{Name: "entry", Description: "desc"})
		}

		// Concurrent reads of all entries
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.AllEntries()
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent set and AllEntries", func(t *testing.T) {
		cache := NewCache()
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(2)

			go func(idx int) {
				defer wg.Done()
				cache.SetText(models.TextSecret{Name: "concurrent", Text: "data"})
			}(i)

			go func() {
				defer wg.Done()
				cache.AllEntries()
			}()
		}

		wg.Wait()
	})
}

func TestCache_Integration(t *testing.T) {
	t.Run("full cache workflow", func(t *testing.T) {
		cache := NewCache()

		// Add various types of secrets
		text := models.TextSecret{Name: "note", Description: "My note", Text: "secret text"}
		cred := models.CredentialSecret{Name: "login", Description: "Email login", Login: "user@example.com", Password: "pass"}
		card := models.CardSecret{Name: "visa", Description: "Main card", CardNumber: "4111111111111111", Owner: "John Doe"}
		file := models.FileSecret{Name: "doc", Description: "Important doc", Path: "/tmp/report.pdf"}

		cache.SetText(text)
		cache.SetCredential(cred)
		cache.SetCard(card)
		cache.SetFile(file)

		// Verify all are retrievable
		retrievedText, foundText := cache.GetText("note")
		retrievedCred, foundCred := cache.GetCredential("login")
		retrievedCard, foundCard := cache.GetCard("visa")
		retrievedFile, foundFile := cache.GetFile("doc")

		assert.True(t, foundText)
		assert.True(t, foundCred)
		assert.True(t, foundCard)
		assert.True(t, foundFile)
		assert.Equal(t, text, retrievedText)
		assert.Equal(t, cred, retrievedCred)
		assert.Equal(t, card, retrievedCard)
		assert.Equal(t, file, retrievedFile)

		// Verify entries
		entries := cache.AllEntries()
		assert.Len(t, entries, 4)

		// Delete one
		cache.DeleteSecret("login", "credential")

		// Verify deletion
		_, foundAfterDelete := cache.GetCredential("login")
		assert.False(t, foundAfterDelete)

		// Verify others still exist
		entries = cache.AllEntries()
		assert.Len(t, entries, 3)
	})

	t.Run("update workflow", func(t *testing.T) {
		cache := NewCache()

		// Create initial secret
		original := models.TextSecret{
			Name:        "my-note",
			Description: "original description",
			Text:        "original text",
		}
		cache.SetText(original)

		// Update the secret
		updated := models.TextSecret{
			Name:        "my-note",
			Description: "updated description",
			Text:        "updated text",
		}
		cache.SetText(updated)

		// Verify update
		retrieved, found := cache.GetText("my-note")
		assert.True(t, found)
		assert.Equal(t, "updated text", retrieved.Text)
		assert.Equal(t, "updated description", retrieved.Description)

		// Verify only one entry exists
		entries := cache.AllEntries()
		assert.Len(t, entries, 1)
		assert.Equal(t, "updated description", entries[0].Description)
	})
}
