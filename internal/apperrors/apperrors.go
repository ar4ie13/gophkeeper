// Package apperrors defines application-wide error constants.
package apperrors

import "errors"

// Common application errors used across the codebase.
var (
	// ErrUserNotFound indicates that the requested user does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists indicates that a user with the given credentials already exists.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrInvalidUserUUID indicates that the provided user UUID is malformed.
	ErrInvalidUserUUID = errors.New("invalid user uuid")
	// ErrUserIsNotAuthorized indicates that the user lacks authorization for the requested action.
	ErrUserIsNotAuthorized = errors.New("user is not authorized")
	// ErrInvalidLoginString indicates that the login contains invalid characters.
	ErrInvalidLoginString = errors.New("invalid login string, use letters and digits only")
	// ErrPasswordMinSymbols indicates that the password does not meet minimum length requirements.
	ErrPasswordMinSymbols = errors.New("password minimum symbols")
	// ErrInvalidPassword indicates that the provided password is incorrect.
	ErrInvalidPassword = errors.New("invalid password")
	// ErrNotFound indicates that the requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrNameRequired indicates that a required name field is missing.
	ErrNameRequired = errors.New("name is required")
	// ErrSecretInvalid indicates that the provided secret data is malformed or invalid.
	ErrSecretInvalid = errors.New("secret is invalid")
)
