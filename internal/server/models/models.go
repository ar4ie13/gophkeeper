// Package models defines the domain models for the server.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered user account with encrypted credentials.
type User struct {
	UUID         uuid.UUID `json:"uuid" db:"uuid"`
	Login        string    `json:"login" db:"login"`
	PasswordHash string    `json:"-" db:"password_hash"`
	EncryptedKey []byte    `json:"-" db:"encrypted_key"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// File represents an encrypted file stored by a user.
type File struct {
	UserUUID    uuid.UUID `json:"user_uuid" db:"user_uuid"`
	UserFile    []byte    `json:"user_file" db:"user_file"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Version     int       `json:"version" db:"version"`
}

// Text represents encrypted text data stored by a user.
type Text struct {
	UserUUID    uuid.UUID `json:"user_uuid" db:"user_uuid"`
	UserText    []byte    `json:"user_text" db:"user_text"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Version     int       `json:"version" db:"version"`
}

// Credential represents encrypted login credentials stored by a user.
type Credential struct {
	UserUUID    uuid.UUID `json:"user_uuid" db:"user_uuid"`
	Login       []byte    `json:"login" db:"login"`
	Password    []byte    `json:"password" db:"password"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Version     int       `json:"version" db:"version"`
}

// Card represents encrypted bank card information stored by a user.
type Card struct {
	UserUUID    uuid.UUID `json:"user_uuid" db:"user_uuid"`
	CardNumber  []byte    `json:"card_number" db:"card_number"`
	Owner       []byte    `json:"owner" db:"owner"`
	ExpiresAt   []byte    `json:"expires_at" db:"expires_at"`
	CVC         []byte    `json:"cvc" db:"cvc"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Version     int       `json:"version" db:"version"`
}
