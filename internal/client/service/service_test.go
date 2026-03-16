package service

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ar4ie13/gophkeeper/internal/client/api"
	"github.com/ar4ie13/gophkeeper/internal/client/config"
	"github.com/ar4ie13/gophkeeper/internal/client/models"
	"github.com/ar4ie13/gophkeeper/internal/client/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService wires a Service to the given httptest server.
func newTestService(t *testing.T, srv *httptest.Server) *Service {
	t.Helper()
	client, err := api.NewClient(config.Config{ServerURL: srv.URL})
	require.NoError(t, err)
	return NewService(client, storage.NewCache())
}

// writeJSON is a convenience helper for test handlers.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ---------- Register ----------

func TestRegister_EmptyLogin(t *testing.T) {
	svc := newTestService(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	err := svc.Register("", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login and password are required")
}

func TestRegister_EmptyPassword(t *testing.T) {
	svc := newTestService(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	err := svc.Register("user", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login and password are required")
}

func TestRegister_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	assert.NoError(t, newTestService(t, srv).Register("user", "pass"))
}

func TestRegister_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()
	err := newTestService(t, srv).Register("user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource already exists")
}

// ---------- Login ----------

func TestLogin_EmptyLogin(t *testing.T) {
	svc := newTestService(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	err := svc.Login("", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login and password are required")
}

func TestLogin_EmptyPassword(t *testing.T) {
	svc := newTestService(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	err := svc.Login("user", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login and password are required")
}

func TestLogin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	assert.NoError(t, newTestService(t, srv).Login("user", "pass"))
}

func TestLogin_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	err := newTestService(t, srv).Login("user", "wrong")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

// ---------- Sync ----------

func TestSync_ServerOffline(t *testing.T) {
	client, _ := api.NewClient(config.Config{ServerURL: "http://127.0.0.1:1"})
	svc := NewService(client, storage.NewCache())

	result := svc.Sync()
	assert.False(t, result.Online)
	assert.Error(t, result.Error)
}

func TestSync_FullSuccess(t *testing.T) {
	texts := []map[string]string{{"name": "note", "description": "d"}}
	creds := []map[string]string{{"name": "gh", "description": "d"}}
	cards := []map[string]string{{"name": "visa", "description": "d"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			writeJSON(w, texts)
		case "/api/user/text/get/note":
			writeJSON(w, models.TextSecret{Name: "note", Text: "hello"})
		case "/api/user/credential/list":
			writeJSON(w, creds)
		case "/api/user/credential/get/gh":
			writeJSON(w, models.CredentialSecret{Name: "gh", Login: "u", Password: "p"})
		case "/api/user/card/list":
			writeJSON(w, cards)
		case "/api/user/card/get/visa":
			writeJSON(w, models.CardSecret{Name: "visa", CardNumber: "1234"})
		case "/api/user/file/list":
			writeJSON(w, []map[string]string{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	result := newTestService(t, srv).Sync()
	assert.True(t, result.Online)
	assert.NoError(t, result.Error)
	assert.Equal(t, 1, result.TextCount)
	assert.Equal(t, 1, result.CredentialCount)
	assert.Equal(t, 1, result.CardCount)
	assert.Equal(t, 0, result.FilesCount)
}

func TestSync_ListTextsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("boom"))
		}
	}))
	defer srv.Close()

	result := newTestService(t, srv).Sync()
	assert.True(t, result.Online)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "list texts")
}

func TestSync_ListTextsEOF(t *testing.T) {
	// 200 with empty body → json.Decode returns io.EOF → early return, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			w.WriteHeader(http.StatusOK) // empty body
		}
	}))
	defer srv.Close()

	result := newTestService(t, srv).Sync()
	assert.True(t, result.Online)
	assert.NoError(t, result.Error)
}

func TestSync_ListCredentialsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			writeJSON(w, []any{})
		case "/api/user/credential/list":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("boom"))
		}
	}))
	defer srv.Close()

	result := newTestService(t, srv).Sync()
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "list credentials")
}

func TestSync_ListCardsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			writeJSON(w, []any{})
		case "/api/user/credential/list":
			writeJSON(w, []any{})
		case "/api/user/card/list":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("boom"))
		}
	}))
	defer srv.Close()

	result := newTestService(t, srv).Sync()
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "list cards")
}

func TestSync_ListFilesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			writeJSON(w, []any{})
		case "/api/user/credential/list":
			writeJSON(w, []any{})
		case "/api/user/card/list":
			writeJSON(w, []any{})
		case "/api/user/file/list":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("boom"))
		}
	}))
	defer srv.Close()

	result := newTestService(t, srv).Sync()
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "list files")
}

func TestSync_GetItemFailuresAreSkipped(t *testing.T) {
	texts := []map[string]string{{"name": "ok"}, {"name": "bad"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			writeJSON(w, texts)
		case "/api/user/text/get/ok":
			writeJSON(w, models.TextSecret{Name: "ok", Text: "content"})
		case "/api/user/text/get/bad":
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/user/credential/list":
			writeJSON(w, []any{})
		case "/api/user/card/list":
			writeJSON(w, []any{})
		case "/api/user/file/list":
			writeJSON(w, []any{})
		}
	}))
	defer srv.Close()

	result := newTestService(t, srv).Sync()
	assert.NoError(t, result.Error)
	assert.Equal(t, 1, result.TextCount) // "bad" was skipped
}

func TestSync_FileItemsAreSynced(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("file data"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/user/text/list":
			writeJSON(w, []any{})
		case "/api/user/credential/list":
			writeJSON(w, []any{})
		case "/api/user/card/list":
			writeJSON(w, []any{})
		case "/api/user/file/list":
			writeJSON(w, []map[string]string{{"name": "report"}})
		case "/api/user/file/get/report":
			writeJSON(w, map[string]string{
				"name":        "report.pdf",
				"description": "annual",
				"user_file":   encoded,
			})
		}
	}))
	defer srv.Close()
	t.Cleanup(func() { os.RemoveAll("./files") })

	result := newTestService(t, srv).Sync()
	assert.NoError(t, result.Error)
	assert.Equal(t, 1, result.FilesCount)
}

// ---------- ListSecrets ----------

func TestListSecrets_EmptyCache(t *testing.T) {
	svc := NewService(nil, storage.NewCache())
	assert.Empty(t, svc.ListSecrets())
}

func TestListSecrets_SortedByTypeThenName(t *testing.T) {
	cache := storage.NewCache()
	cache.SetText(models.TextSecret{Name: "z-note"})
	cache.SetText(models.TextSecret{Name: "a-note"})
	cache.SetCredential(models.CredentialSecret{Name: "gh"})
	cache.SetCard(models.CardSecret{Name: "visa"})

	entries := NewService(nil, cache).ListSecrets()

	require.Len(t, entries, 4)
	// Type ordering: Text(0) < Credential(1) < Card(2)
	assert.Equal(t, models.SecretTypeText, entries[0].Type)
	assert.Equal(t, "a-note", entries[0].Name)
	assert.Equal(t, models.SecretTypeText, entries[1].Type)
	assert.Equal(t, "z-note", entries[1].Name)
	assert.Equal(t, models.SecretTypeCredential, entries[2].Type)
	assert.Equal(t, models.SecretTypeCard, entries[3].Type)
}

// ---------- GetText ----------

func TestGetText_ServerSuccess_UpdatesCache(t *testing.T) {
	want := models.TextSecret{Name: "note", Text: "hello"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, want)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	got, err := svc.GetText("note")
	require.NoError(t, err)
	assert.Equal(t, &want, got)

	cached, ok := svc.cache.GetText("note")
	assert.True(t, ok)
	assert.Equal(t, want, cached)
}

func TestGetText_ServerFails_CacheFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	cached := models.TextSecret{Name: "note", Text: "cached"}
	svc.cache.SetText(cached)

	got, err := svc.GetText("note")
	require.NoError(t, err)
	assert.Equal(t, &cached, got)
}

func TestGetText_ServerFails_NotInCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestService(t, srv).GetText("note")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

// ---------- GetCredential ----------

func TestGetCredential_ServerSuccess_UpdatesCache(t *testing.T) {
	want := models.CredentialSecret{Name: "gh", Login: "u", Password: "p"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, want)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	got, err := svc.GetCredential("gh")
	require.NoError(t, err)
	assert.Equal(t, &want, got)

	cached, ok := svc.cache.GetCredential("gh")
	assert.True(t, ok)
	assert.Equal(t, want, cached)
}

func TestGetCredential_ServerFails_CacheFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	cred := models.CredentialSecret{Name: "gh", Login: "u", Password: "p"}
	svc.cache.SetCredential(cred)

	got, err := svc.GetCredential("gh")
	require.NoError(t, err)
	assert.Equal(t, &cred, got)
}

func TestGetCredential_ServerFails_NotInCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestService(t, srv).GetCredential("gh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

// ---------- GetCard ----------

func TestGetCard_ServerSuccess_UpdatesCache(t *testing.T) {
	want := models.CardSecret{Name: "visa", CardNumber: "1234"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, want)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	got, err := svc.GetCard("visa")
	require.NoError(t, err)
	assert.Equal(t, &want, got)

	cached, ok := svc.cache.GetCard("visa")
	assert.True(t, ok)
	assert.Equal(t, want, cached)
}

func TestGetCard_ServerFails_CacheFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	card := models.CardSecret{Name: "visa", CardNumber: "1234"}
	svc.cache.SetCard(card)

	got, err := svc.GetCard("visa")
	require.NoError(t, err)
	assert.Equal(t, &card, got)
}

func TestGetCard_ServerFails_NotInCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestService(t, srv).GetCard("visa")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

// ---------- GetFile ----------

func TestGetFile_ServerSuccess_UpdatesCache(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"name": "report.pdf", "description": "d", "user_file": encoded})
	}))
	defer srv.Close()
	t.Cleanup(func() { os.RemoveAll("./files") })

	svc := newTestService(t, srv)
	got, err := svc.GetFile("report")
	require.NoError(t, err)
	assert.Equal(t, "report.pdf", got.Name)

	_, ok := svc.cache.GetFile("report.pdf")
	assert.True(t, ok)
}

func TestGetFile_ServerFails_CacheFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	file := models.FileSecret{Name: "report", Path: "/tmp/report.pdf"}
	svc.cache.SetFile(file)

	got, err := svc.GetFile("report")
	require.NoError(t, err)
	assert.Equal(t, &file, got)
}

func TestGetFile_ServerFails_NotInCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestService(t, srv).GetFile("report")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

// ---------- StoreText ----------

func TestStoreText_EmptyName(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreText(models.TextSecret{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestStoreText_TooLong(t *testing.T) {
	long := strings.Repeat("x", 1001)
	err := NewService(nil, storage.NewCache()).StoreText(models.TextSecret{Name: "n", Text: long})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds 1000 character limit")
}

func TestStoreText_ExactlyAtLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	text := strings.Repeat("x", 1000)
	assert.NoError(t, newTestService(t, srv).StoreText(models.TextSecret{Name: "n", Text: text}))
}

func TestStoreText_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	secret := models.TextSecret{Name: "note", Text: "hello"}
	require.NoError(t, svc.StoreText(secret))

	cached, ok := svc.cache.GetText("note")
	assert.True(t, ok)
	assert.Equal(t, secret, cached)
}

func TestStoreText_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestService(t, srv).StoreText(models.TextSecret{Name: "note", Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

// ---------- StoreCredential ----------

func TestStoreCredential_EmptyName(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreCredential(models.CredentialSecret{Login: "u", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestStoreCredential_EmptyLogin(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreCredential(models.CredentialSecret{Name: "gh", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login and password are required")
}

func TestStoreCredential_EmptyPassword(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreCredential(models.CredentialSecret{Name: "gh", Login: "u"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login and password are required")
}

func TestStoreCredential_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	secret := models.CredentialSecret{Name: "gh", Login: "u", Password: "p"}
	require.NoError(t, svc.StoreCredential(secret))

	cached, ok := svc.cache.GetCredential("gh")
	assert.True(t, ok)
	assert.Equal(t, secret, cached)
}

func TestStoreCredential_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestService(t, srv).StoreCredential(models.CredentialSecret{Name: "gh", Login: "u", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

// ---------- StoreCard ----------

func TestStoreCard_EmptyName(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreCard(models.CardSecret{CardNumber: "1234", Owner: "Me"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestStoreCard_EmptyCardNumber(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreCard(models.CardSecret{Name: "visa", Owner: "Me"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "card number and owner are required")
}

func TestStoreCard_EmptyOwner(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreCard(models.CardSecret{Name: "visa", CardNumber: "1234"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "card number and owner are required")
}

func TestStoreCard_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	secret := models.CardSecret{Name: "visa", CardNumber: "1234", Owner: "Me"}
	require.NoError(t, svc.StoreCard(secret))

	cached, ok := svc.cache.GetCard("visa")
	assert.True(t, ok)
	assert.Equal(t, secret, cached)
}

func TestStoreCard_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestService(t, srv).StoreCard(models.CardSecret{Name: "visa", CardNumber: "1234", Owner: "Me"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

// ---------- StoreFile ----------

func TestStoreFile_EmptyName(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreFile(models.FileSecret{Path: "/tmp/f"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestStoreFile_EmptyPath(t *testing.T) {
	err := NewService(nil, storage.NewCache()).StoreFile(models.FileSecret{Name: "doc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file path is required")
}

func TestStoreFile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	tmpFile := t.TempDir() + "/doc.txt"
	require.NoError(t, os.WriteFile(tmpFile, []byte("content"), 0644))

	svc := newTestService(t, srv)
	secret := models.FileSecret{Name: "doc", Path: tmpFile}
	require.NoError(t, svc.StoreFile(secret))

	cached, ok := svc.cache.GetFile("doc")
	assert.True(t, ok)
	assert.Equal(t, secret, cached)
}

func TestStoreFile_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tmpFile := t.TempDir() + "/doc.txt"
	require.NoError(t, os.WriteFile(tmpFile, []byte("content"), 0644))

	err := newTestService(t, srv).StoreFile(models.FileSecret{Name: "doc", Path: tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

// ---------- UpdateText ----------

func TestUpdateText_EmptyName(t *testing.T) {
	err := NewService(nil, storage.NewCache()).UpdateText(models.TextSecret{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestUpdateText_TooLong(t *testing.T) {
	long := strings.Repeat("x", 1001)
	err := NewService(nil, storage.NewCache()).UpdateText(models.TextSecret{Name: "n", Text: long})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds 1000 character limit")
}

func TestUpdateText_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	secret := models.TextSecret{Name: "note", Text: "updated"}
	require.NoError(t, svc.UpdateText(secret))

	cached, ok := svc.cache.GetText("note")
	assert.True(t, ok)
	assert.Equal(t, secret, cached)
}

func TestUpdateText_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestService(t, srv).UpdateText(models.TextSecret{Name: "note", Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

// ---------- UpdateCredential ----------

func TestUpdateCredential_EmptyName(t *testing.T) {
	err := NewService(nil, storage.NewCache()).UpdateCredential(models.CredentialSecret{Login: "u", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestUpdateCredential_EmptyLogin(t *testing.T) {
	err := NewService(nil, storage.NewCache()).UpdateCredential(models.CredentialSecret{Name: "gh", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login and password are required")
}

func TestUpdateCredential_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	secret := models.CredentialSecret{Name: "gh", Login: "u2", Password: "p2"}
	require.NoError(t, svc.UpdateCredential(secret))

	cached, ok := svc.cache.GetCredential("gh")
	assert.True(t, ok)
	assert.Equal(t, secret, cached)
}

func TestUpdateCredential_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestService(t, srv).UpdateCredential(models.CredentialSecret{Name: "gh", Login: "u", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

// ---------- UpdateCard ----------

func TestUpdateCard_EmptyName(t *testing.T) {
	err := NewService(nil, storage.NewCache()).UpdateCard(models.CardSecret{CardNumber: "1234", Owner: "Me"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestUpdateCard_EmptyCardNumber(t *testing.T) {
	err := NewService(nil, storage.NewCache()).UpdateCard(models.CardSecret{Name: "visa", Owner: "Me"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "card number and owner are required")
}

func TestUpdateCard_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	secret := models.CardSecret{Name: "visa", CardNumber: "9999", Owner: "Me"}
	require.NoError(t, svc.UpdateCard(secret))

	cached, ok := svc.cache.GetCard("visa")
	assert.True(t, ok)
	assert.Equal(t, secret, cached)
}

func TestUpdateCard_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestService(t, srv).UpdateCard(models.CardSecret{Name: "visa", CardNumber: "1234", Owner: "Me"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

// ---------- DeleteSecret ----------

func TestDeleteSecret_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := newTestService(t, srv)
	svc.cache.SetText(models.TextSecret{Name: "note"})

	require.NoError(t, svc.DeleteSecret("note", "text"))

	_, ok := svc.cache.GetText("note")
	assert.False(t, ok)
}

func TestDeleteSecret_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newTestService(t, srv).DeleteSecret("note", "text")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete secret")
}
