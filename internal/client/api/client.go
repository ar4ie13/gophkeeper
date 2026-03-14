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

// doAuth is a helper that handles common authentication flow for register and login.
func (c *Client) doAuth(path, login, password string) error {
	body, err := json.Marshal(authRequest{Login: login, Password: password})
	if err != nil {
		return fmt.Errorf("marshal auth: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+path,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("user already exists")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid login or password")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
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
	body, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshal text secret: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/user/text/store",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("store text: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("store text failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// UpdateText sends update to a text secret to the server.
func (c *Client) UpdateText(secret models.TextSecret) error {
	body, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshal text secret: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.baseURL+"/api/user/text/patch", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("update text: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update text failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// GetText retrieves a text secret by name from the server.
func (c *Client) GetText(name string) (*models.TextSecret, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/text/get/" + name)
	if err != nil {
		return nil, fmt.Errorf("get text: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("not authenticated")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("text secret %q not found", name)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get text failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var secret models.TextSecret
	if err := json.NewDecoder(resp.Body).Decode(&secret); err != nil {
		return nil, fmt.Errorf("decode text secret: %w", err)
	}
	return &secret, nil
}

// StoreCredential sends a credential secret to the server.
func (c *Client) StoreCredential(secret models.CredentialSecret) error {
	body, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshal credential: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/user/credential/store",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("store credential: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("store credential failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// GetCredential retrieves a credential secret by name from the server.
func (c *Client) GetCredential(name string) (*models.CredentialSecret, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/credential/get/" + name)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("not authenticated")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("credential %q not found", name)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get credential failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var secret models.CredentialSecret
	if err := json.NewDecoder(resp.Body).Decode(&secret); err != nil {
		return nil, fmt.Errorf("decode credential: %w", err)
	}
	return &secret, nil
}

// UpdateCredential sends update to a credential secret to the server.
func (c *Client) UpdateCredential(secret models.CredentialSecret) error {
	body, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshal credential secret: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.baseURL+"/api/user/credential/patch", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("update credential: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update credential failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// StoreCard sends a card secret to the server.
func (c *Client) StoreCard(secret models.CardSecret) error {
	body, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/user/card/store",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("store card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("store card failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// GetCard retrieves a card secret by name from the server.
func (c *Client) GetCard(name string) (*models.CardSecret, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/card/get/" + name)
	if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("not authenticated")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("card %q not found", name)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get card failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var secret models.CardSecret
	if err = json.NewDecoder(resp.Body).Decode(&secret); err != nil {
		return nil, fmt.Errorf("decode card: %w", err)
	}
	return &secret, nil
}

// UpdateCard sends update to a card secret to the server.
func (c *Client) UpdateCard(secret models.CardSecret) error {
	body, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshal card secret: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.baseURL+"/api/user/card/patch", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("update card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update card failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// StoreFile sends a file secret to the server.
func (c *Client) StoreFile(secret models.FileSecret) error {
	file, err := os.ReadFile(secret.Path)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}
	var fileReq fileReqResp
	fileReq.UserFile = base64.StdEncoding.EncodeToString(file)
	// Exact check on decoded size
	if len(fileReq.UserFile) > 5<<20 {
		return fmt.Errorf("file exceeds 5MB limit")
	}

	fileReq.Name = secret.Name
	fileReq.Description = secret.Description

	body, err := json.Marshal(fileReq)
	if err != nil {
		return fmt.Errorf("marshal file secret: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/user/file/store",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("store file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("store file failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
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
func (c *Client) DeleteSecret(name string, secretType string) error {
	var deleteReq deleteRequest
	deleteReq.Name = name
	deleteReq.Type = secretType

	body, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("marshal secret: %w", err)
	}

	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/user/secret/delete", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("not authenticated — please log in again")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update text failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ListTexts returns all text secret names from the server.
func (c *Client) ListTexts() ([]listResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/text/list")
	if err != nil {
		return nil, fmt.Errorf("list texts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list texts failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var items []listResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode text list: %w", err)
	}
	return items, nil
}

// ListCredentials returns all credential secret names from the server.
func (c *Client) ListCredentials() ([]listResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/credential/list")
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list credentials failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var items []listResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode credential list: %w", err)
	}
	return items, nil
}

// ListCards returns all card secret names from the server.
func (c *Client) ListCards() ([]listResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/card/list")
	if err != nil {
		return nil, fmt.Errorf("list cards: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list cards failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var items []listResponse
	if err = json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode card list: %w", err)
	}
	return items, nil
}

// ListFiles returns all file secret names from the server.
func (c *Client) ListFiles() ([]listResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/user/file/list")
	if err != nil {
		return nil, fmt.Errorf("list file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list files failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var items []listResponse
	if err = json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode file list: %w", err)
	}
	return items, nil
}
