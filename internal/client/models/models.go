package models

import "time"

// SecretType distinguishes between text secrets and credential secrets.
type SecretType int

const (
	SecretTypeText SecretType = iota
	SecretTypeCredential
	SecretTypeCard
	SecretTypeFile
)

// String returns the string representation of a SecretType.
func (s SecretType) String() string {
	switch s {
	case SecretTypeText:
		return "text"
	case SecretTypeCredential:
		return "credential"
	case SecretTypeCard:
		return "card"
	case SecretTypeFile:
		return "file"
	default:
		return "unknown"
	}
}

// TextSecret represents a stored text secret.
type TextSecret struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Text        string `json:"text"`
}

// CredentialSecret represents a stored login/password pair.
type CredentialSecret struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Login       string `json:"login"`
	Password    string `json:"password"`
}

// CardSecret represents a stored card secret.
type CardSecret struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CardNumber  string `json:"card_number"`
	Owner       string `json:"owner"`
	ExpiresAt   string `json:"expires_at"`
	CVC         string `json:"cvc"`
}

// FileSecret represents a stored text secret.
type FileSecret struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// SecretEntry is a unified view used for listing secrets in the TUI.
type SecretEntry struct {
	Name        string
	Description string
	Type        SecretType
	SyncedAt    time.Time
}
