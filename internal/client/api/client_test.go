package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/client/config"
	"github.com/ar4ie13/gophkeeper/internal/client/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// selfSignedCertPEM generates a minimal self-signed PEM certificate for tests.
func selfSignedCertPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// newTestClient builds a Client wired to the given test server.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(config.Config{ServerURL: srv.URL})
	require.NoError(t, err)
	return c
}

// ---------- NewClient ----------

func TestNewClient_NoCaCert(t *testing.T) {
	c, err := NewClient(config.Config{ServerURL: "http://localhost:8080"})
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "http://localhost:8080", c.baseURL)
}

func TestNewClient_InvalidCaCert(t *testing.T) {
	_, err := NewClient(config.Config{
		ServerURL: "http://localhost:8080",
		CaCert:    []byte("not a valid PEM certificate"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse CA certificate")
}

func TestNewClient_ValidCaCert(t *testing.T) {
	c, err := NewClient(config.Config{
		ServerURL: "https://localhost:8443",
		CaCert:    selfSignedCertPEM(t),
	})
	require.NoError(t, err)
	assert.NotNil(t, c)
}

// ---------- Register ----------

func TestRegister_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/register", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).Register("user", "pass")
	assert.NoError(t, err)
}

func TestRegister_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).Register("user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user already exists")
}

func TestRegister_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).Register("user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid login or password")
}

func TestRegister_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).Register("user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid login or password")
}

func TestRegister_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).Register("user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed (500)")
}

func TestRegister_NetworkError(t *testing.T) {
	c, _ := NewClient(config.Config{ServerURL: "http://127.0.0.1:1"})
	err := c.Register("user", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth request")
}

// ---------- Login ----------

func TestLogin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/login", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).Login("user", "pass")
	assert.NoError(t, err)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).Login("user", "wrong")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid login or password")
}

// ---------- Ping ----------

func TestPing_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/ping", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	assert.NoError(t, newTestClient(t, srv).Ping())
}

func TestPing_AnyStatusMeansAlive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	assert.NoError(t, newTestClient(t, srv).Ping())
}

func TestPing_NetworkError(t *testing.T) {
	c, _ := NewClient(config.Config{ServerURL: "http://127.0.0.1:1"})
	err := c.Ping()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server unreachable")
}

// ---------- StoreText ----------

func TestStoreText_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/text/store", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreText(models.TextSecret{Name: "note", Text: "hello"})
	assert.NoError(t, err)
}

func TestStoreText_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreText(models.TextSecret{Name: "note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestStoreText_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("oops"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreText(models.TextSecret{Name: "note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store text failed (500)")
}

// ---------- UpdateText ----------

func TestUpdateText_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/user/text/patch", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateText(models.TextSecret{Name: "note", Text: "updated"})
	assert.NoError(t, err)
}

func TestUpdateText_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateText(models.TextSecret{Name: "note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestUpdateText_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateText(models.TextSecret{Name: "note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update text failed (400)")
}

// ---------- GetText ----------

func TestGetText_Success(t *testing.T) {
	want := models.TextSecret{Name: "note", Description: "desc", Text: "hello"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/text/get/note", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv).GetText("note")
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestGetText_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetText("note")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetText_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetText("note")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestGetText_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetText("note")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get text failed (500)")
}

func TestGetText_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetText("note")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode text secret")
}

// ---------- StoreCredential ----------

func TestStoreCredential_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/credential/store", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreCredential(models.CredentialSecret{Name: "gh", Login: "u", Password: "p"})
	assert.NoError(t, err)
}

func TestStoreCredential_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreCredential(models.CredentialSecret{Name: "gh"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestStoreCredential_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreCredential(models.CredentialSecret{Name: "gh"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store credential failed (500)")
}

// ---------- GetCredential ----------

func TestGetCredential_Success(t *testing.T) {
	want := models.CredentialSecret{Name: "gh", Login: "u", Password: "p"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/credential/get/gh", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv).GetCredential("gh")
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestGetCredential_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetCredential("gh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetCredential_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetCredential("gh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode credential")
}

// ---------- UpdateCredential ----------

func TestUpdateCredential_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/user/credential/patch", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateCredential(models.CredentialSecret{Name: "gh"})
	assert.NoError(t, err)
}

func TestUpdateCredential_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateCredential(models.CredentialSecret{Name: "gh"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestUpdateCredential_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateCredential(models.CredentialSecret{Name: "gh"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update credential failed (400)")
}

// ---------- StoreCard ----------

func TestStoreCard_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/card/store", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreCard(models.CardSecret{Name: "visa"})
	assert.NoError(t, err)
}

func TestStoreCard_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreCard(models.CardSecret{Name: "visa"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestStoreCard_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreCard(models.CardSecret{Name: "visa"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store card failed (500)")
}

// ---------- GetCard ----------

func TestGetCard_Success(t *testing.T) {
	want := models.CardSecret{Name: "visa", CardNumber: "1234", Owner: "Me"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/card/get/visa", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv).GetCard("visa")
	require.NoError(t, err)
	assert.Equal(t, &want, got)
}

func TestGetCard_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetCard("visa")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetCard_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetCard("visa")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestGetCard_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetCard("visa")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode card")
}

// ---------- UpdateCard ----------

func TestUpdateCard_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/user/card/patch", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateCard(models.CardSecret{Name: "visa"})
	assert.NoError(t, err)
}

func TestUpdateCard_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateCard(models.CardSecret{Name: "visa"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestUpdateCard_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).UpdateCard(models.CardSecret{Name: "visa"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update card failed (400)")
}

// ---------- StoreFile ----------

func TestStoreFile_Success(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("file content"), 0644))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/file/store", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreFile(models.FileSecret{Name: "doc", Path: tmpFile})
	assert.NoError(t, err)
}

func TestStoreFile_FileNotFound(t *testing.T) {
	err := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))).
		StoreFile(models.FileSecret{Name: "doc", Path: "/nonexistent/path/file.bin"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read file")
}

func TestStoreFile_ExceedsSizeLimit(t *testing.T) {
	// A raw file whose base64 representation exceeds 5<<20 bytes.
	// base64(N bytes) ≈ ceil(N/3)*4; we need that > 5*1024*1024 = 5242880.
	// N = 4*1024*1024 (4 MB) → base64 ≈ 5592405 > 5242880.
	big := make([]byte, 4*1024*1024)
	tmpFile := filepath.Join(t.TempDir(), "big.bin")
	require.NoError(t, os.WriteFile(tmpFile, big, 0644))

	err := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))).
		StoreFile(models.FileSecret{Name: "big", Path: tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file exceeds 5MB limit")
}

func TestStoreFile_Unauthorized(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0644))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreFile(models.FileSecret{Name: "f", Path: tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestStoreFile_ServerError(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0644))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).StoreFile(models.FileSecret{Name: "f", Path: tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store file failed (500)")
}

// ---------- GetFile ----------

func TestGetFile_Success(t *testing.T) {
	fileContent := []byte("downloaded content")
	encoded := base64.StdEncoding.EncodeToString(fileContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/file/get/doc", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(fileReqResp{
			Name:        "doc.txt",
			Description: "a doc",
			UserFile:    encoded,
		})
	}))
	defer srv.Close()

	// GetFile writes to ./files/ relative to cwd; clean up afterwards.
	t.Cleanup(func() { os.RemoveAll("./files") })

	got, err := newTestClient(t, srv).GetFile("doc")
	require.NoError(t, err)
	assert.Equal(t, "doc.txt", got.Name)
	assert.Equal(t, "a doc", got.Description)
	data, err := os.ReadFile(got.Path)
	require.NoError(t, err)
	assert.Equal(t, fileContent, data)
}

func TestGetFile_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetFile("doc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetFile_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetFile("doc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestGetFile_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetFile("doc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshall file secret")
}

func TestGetFile_InvalidBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(fileReqResp{Name: "f", UserFile: "!!!not-base64!!!"})
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetFile("doc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode file secret")
}

// ---------- DeleteSecret ----------

func TestDeleteSecret_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/user/secret/delete", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).DeleteSecret("note", "text")
	assert.NoError(t, err)
}

func TestDeleteSecret_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).DeleteSecret("note", "text")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestDeleteSecret_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).DeleteSecret("note", "text")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// ---------- ListTexts ----------

func TestListTexts_Success(t *testing.T) {
	want := []listResponse{{Name: "note1", Description: "d1"}, {Name: "note2", Description: "d2"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/text/list", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv).ListTexts()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListTexts_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListTexts()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list texts failed (500)")
}

func TestListTexts_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListTexts()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode text list")
}

// ---------- ListCredentials ----------

func TestListCredentials_Success(t *testing.T) {
	want := []listResponse{{Name: "gh"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/credential/list", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv).ListCredentials()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListCredentials_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListCredentials()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list credentials failed (500)")
}

func TestListCredentials_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListCredentials()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode credential list")
}

// ---------- ListCards ----------

func TestListCards_Success(t *testing.T) {
	want := []listResponse{{Name: "visa"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/card/list", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv).ListCards()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListCards_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListCards()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list cards failed (500)")
}

func TestListCards_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListCards()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode card list")
}

// ---------- ListFiles ----------

func TestListFiles_Success(t *testing.T) {
	want := []listResponse{{Name: "report.pdf"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/user/file/list", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv).ListFiles()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListFiles_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("err"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListFiles()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list files failed (500)")
}

func TestListFiles_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).ListFiles()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode file list")
}
