package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/client/config"
	"github.com/ar4ie13/gophkeeper/internal/client/models"
)

// Client communicates with the secret's server.
// It uses a cookie jar to persist the user_uuid JWT cookie set by the server
// after a successful register or login call.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates an API client pointing at the given server URL.
// A cookie jar is attached so the JWT token set by /login or /register
// is automatically sent with every subsequent request.
// If caCert is non-nil it is added to the TLS root CA pool so that
// servers using custom or self-signed certificates are trusted.
func NewClient(cfg config.Config) (*Client, error) {
	jar, _ := cookiejar.New(nil) // stdlib jar never errors

	transport := http.DefaultTransport.(*http.Transport).Clone()

	if len(cfg.CaCert) > 0 {
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(cfg.CaCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		transport.TLSClientConfig = &tls.Config{
			RootCAs: pool,
		}
	}

	return &Client{
		baseURL: cfg.ServerURL,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Jar:       jar,
			Transport: transport,
		},
	}, nil
}

// Register creates a new user account on the server.
// On success the server sets a user_uuid cookie (JWT) that the cookie jar
// captures automatically.
func (c *Client) Register(login, password string) error {
	return c.doAuth("/api/user/register", login, password)
}

// Login authenticates an existing user.
// On success the server sets a user_uuid cookie (JWT).
func (c *Client) Login(login, password string) error {
	return c.doAuth("/api/user/login", login, password)
}

// Ping checks whether the server is reachable.
func (c *Client) Ping() error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/user/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()
	// Any response (even 404) means server is alive.
	return nil
}

// StoreText sends a text secret to the server.
func (c *Client) StoreText(secret models.TextSecret) error {
	return c.sendJSON(http.MethodPost, "/api/user/text/store", secret)
}

// UpdateText sends update to a text secret to the server.
func (c *Client) UpdateText(secret models.TextSecret) error {
	return c.sendJSON(http.MethodPatch, "/api/user/text/patch", secret)
}

// GetText retrieves a text secret by name from the server.
func (c *Client) GetText(name string) (*models.TextSecret, error) {
	return fetchJSON[models.TextSecret](c, "/api/user/text/get/"+name)
}

// StoreCredential sends a credential secret to the server.
func (c *Client) StoreCredential(secret models.CredentialSecret) error {
	return c.sendJSON(http.MethodPost, "/api/user/credential/store", secret)
}

// GetCredential retrieves a credential secret by name from the server.
func (c *Client) GetCredential(name string) (*models.CredentialSecret, error) {
	return fetchJSON[models.CredentialSecret](c, "/api/user/credential/get/"+name)
}

// UpdateCredential sends update to a credential secret to the server.
func (c *Client) UpdateCredential(secret models.CredentialSecret) error {
	return c.sendJSON(http.MethodPatch, "/api/user/credential/patch", secret)
}

// StoreCard sends a card secret to the server.
func (c *Client) StoreCard(secret models.CardSecret) error {
	return c.sendJSON(http.MethodPost, "/api/user/card/store", secret)
}

// GetCard retrieves a card secret by name from the server.
func (c *Client) GetCard(name string) (*models.CardSecret, error) {
	return fetchJSON[models.CardSecret](c, "/api/user/card/get/"+name)
}

// UpdateCard sends update to a card secret to the server.
func (c *Client) UpdateCard(secret models.CardSecret) error {
	return c.sendJSON(http.MethodPatch, "/api/user/card/patch", secret)
}

// StoreFile sends a file secret to the server.
func (c *Client) StoreFile(secret models.FileSecret) error {
	// Специфичная логика (base64 + size check) — остаётся как есть,
	// но финальный HTTP-вызов делегируется:
	file, err := os.ReadFile(secret.Path)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(file)
	if len(encoded) > 5<<20 {
		return fmt.Errorf("file exceeds 5MB limit")
	}
	return c.sendJSON(http.MethodPost, "/api/user/file/store", fileReqResp{
		Name:        secret.Name,
		Description: secret.Description,
		UserFile:    encoded,
	})
}

// GetFile retrieves a file secret by name from the server.
func (c *Client) GetFile(name string) (*models.FileSecret, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/file/get/" + name)
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("not authenticated")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file secret %q not found", name)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get file failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var fileResp fileReqResp
	if err = json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return nil, fmt.Errorf("unmarshall file secret: %w", err)
	}

	file, err := base64.StdEncoding.DecodeString(fileResp.UserFile)
	if err != nil {
		return nil, fmt.Errorf("decode file secret: %w", err)
	}

	if err = os.MkdirAll("./files", 0755); err != nil {
		return nil, fmt.Errorf("create file folder: %w", err)
	}

	var secret models.FileSecret

	secret.Path, err = filepath.Abs(filepath.Join("./files", fileResp.Name))
	if err != nil {
		return nil, fmt.Errorf("cannot obtain path to file: %w", err)
	}
	err = os.WriteFile(secret.Path, file, 0644)
	if err != nil {
		return nil, fmt.Errorf("write file secret: %w", err)
	}
	secret.Name = fileResp.Name
	secret.Description = fileResp.Description
	return &secret, nil
}

// DeleteSecret sends delete secret to the server.
func (c *Client) DeleteSecret(name, secretType string) error {
	return c.sendJSON(http.MethodDelete, "/api/user/secret/delete",
		deleteRequest{Name: name, Type: secretType})
}

// ListTexts returns all text secret names from the server.
func (c *Client) ListTexts() (*[]listResponse, error) {
	return fetchJSON[[]listResponse](c, "/api/user/text/list")
}

// ListCredentials returns all credential secret names from the server.
func (c *Client) ListCredentials() (*[]listResponse, error) {
	return fetchJSON[[]listResponse](c, "/api/user/credential/list")
}

// ListCards returns all card secret names from the server.
func (c *Client) ListCards() (*[]listResponse, error) {
	return fetchJSON[[]listResponse](c, "/api/user/card/list")
}

// ListFiles returns all file secret names from the server.
func (c *Client) ListFiles() (*[]listResponse, error) {
	return fetchJSON[[]listResponse](c, "/api/user/file/list")
}

// --api helpers--

// sendJSON marshals payload, executes method+path, checks status.
func (c *Client) sendJSON(method, path string, payload any) error {
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", path, err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	return checkStatus(resp)
}

// fetchJSON executes a GET, checks status, and decodes the JSON body into T.
func fetchJSON[T any](c *Client, path string) (*T, error) {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if err = checkStatus(resp); err != nil {
		return nil, err
	}

	var out T
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return &out, nil
}

// checkStatus maps HTTP status codes to sentinel errors.
func checkStatus(resp *http.Response) error {
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("not authenticated — please log in again")
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("resource not found") // callers могут обернуть
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("conflict: resource already exists")
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
}

// doAuth is a helper that handles common authentication flow for register and login.
func (c *Client) doAuth(path, login, password string) error {
	return c.sendJSON(http.MethodPost, path, authRequest{Login: login, Password: password})
}
