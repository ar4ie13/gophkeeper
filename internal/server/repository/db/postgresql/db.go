package postgresql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/apperrors"
	"github.com/ar4ie13/gophkeeper/internal/server/models"
	"github.com/ar4ie13/gophkeeper/internal/server/repository/db/postgresql/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PgxPool is an interface for pgxpool.Pool to allow mocking
type PgxPool interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Ping(ctx context.Context) error
	Close()
}

// DB is a main postgres repository object
type DB struct {
	pool PgxPool
	zlog zerolog.Logger
}

// NewDB construct postgres DB object
func NewDB(ctx context.Context, cfg config.PGConf, zlog zerolog.Logger) (*DB, error) {
	pool, err := initPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}
	return &DB{
		pool: pool,
		zlog: zlog,
	}, nil
}

// initPool initializes pgx connection pool
func initPool(ctx context.Context, cfg config.PGConf) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the DSN: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping the DB: %w", err)
	}
	return pool, nil
}

// Close closes pgx pool
func (db *DB) Close() error {
	db.pool.Close()
	return nil
}

// CreateUser stores user information to the db
func (db *DB) CreateUser(ctx context.Context, user models.User) error {
	const query = `INSERT INTO users (uuid, login, password_hash, encrypted_key) 
		VALUES ($1, $2, $3, $4) ON CONFLICT (login) DO NOTHING`

	tag, err := db.pool.Exec(ctx, query, user.UUID, user.Login, user.PasswordHash, user.EncryptedKey)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return apperrors.ErrUserAlreadyExists
	}

	return nil
}

// GetUserByLogin retrieves user information from db
func (db *DB) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	const query = `SELECT uuid, login, password_hash, encrypted_key, created_at, updated_at from users where login=$1`

	var user models.User

	row := db.pool.QueryRow(ctx, query, login)

	err := row.Scan(&user.UUID, &user.Login, &user.PasswordHash, &user.EncryptedKey, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.User{}, apperrors.ErrUserNotFound
		default:
			return models.User{}, fmt.Errorf("failed to scan a response row: %w", err)
		}
	}

	return user, nil
}

// GetUserKey retrieves the encrypted encryption key for a user.
func (db *DB) GetUserKey(ctx context.Context, user uuid.UUID) ([]byte, error) {
	const query = `SELECT encrypted_key FROM users WHERE uuid=$1`
	row := db.pool.QueryRow(ctx, query, user)
	var userKey []byte
	err := row.Scan(&userKey)
	if err != nil {
		return nil, fmt.Errorf("failed to scan a response row: %w", err)
	}

	return userKey, nil
}

// StoreSecret stores a secret of the specified type after type assertion.
func (db *DB) StoreSecret(ctx context.Context, secretType string, secret any) error {
	switch secretType {
	case "text":
		text, ok := secret.(models.Text)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return db.storeText(ctx, text)
	case "credential":
		credential, ok := secret.(models.Credential)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return db.storeCredential(ctx, credential)
	case "card":
		card, ok := secret.(models.Card)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return db.storeCard(ctx, card)
	case "file":
		file, ok := secret.(models.File)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return db.storeFile(ctx, file)
	default:
		return apperrors.ErrSecretInvalid
	}
}

// GetSecret retrieves a secret of the specified type by name.
func (db *DB) GetSecret(ctx context.Context, userUUID uuid.UUID, secretType string, secretName string) (any, error) {
	switch secretType {
	case "text":
		return db.getText(ctx, userUUID, secretName)
	case "credential":
		return db.getCredential(ctx, userUUID, secretName)
	case "card":
		return db.getCard(ctx, userUUID, secretName)
	case "file":
		return db.getFile(ctx, userUUID, secretName)
	default:
		return nil, apperrors.ErrSecretInvalid
	}
}

// UpdateSecret updates an existing secret after type assertion.
func (db *DB) UpdateSecret(ctx context.Context, secretType string, secret any) error {
	switch secretType {
	case "text":
		text, ok := secret.(models.Text)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return db.updateText(ctx, text)
	case "credential":
		credential, ok := secret.(models.Credential)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return db.updateCredential(ctx, credential)
	case "card":
		card, ok := secret.(models.Card)
		if !ok {
			return apperrors.ErrSecretInvalid
		}
		return db.updateCard(ctx, card)

	default:
		return apperrors.ErrSecretInvalid
	}
}

// DeleteSecret deletes a secret of the specified type by name.
func (db *DB) DeleteSecret(ctx context.Context, userUUID uuid.UUID, secretType string, secretName string) error {
	switch secretType {
	case "text":
		return db.deleteText(ctx, userUUID, secretName)
	case "credential":
		return db.deleteCredential(ctx, userUUID, secretName)
	case "card":
		return db.deleteCard(ctx, userUUID, secretName)
	case "file":
		return db.deleteFile(ctx, userUUID, secretName)
	default:
		return apperrors.ErrSecretInvalid
	}
}

// storeText inserts a text secret into the database.
func (db *DB) storeText(ctx context.Context, text models.Text) error {
	const query = `INSERT INTO text (user_uuid, user_text, name, description) 
		VALUES ($1, $2, $3, $4) ON CONFLICT (user_uuid, name) DO NOTHING`

	tag, err := db.pool.Exec(ctx, query, text.UserUUID, text.UserText, text.Name, text.Description)
	if err != nil {
		return fmt.Errorf("failed to store text: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("failed to store text: %w", err)
	}

	return nil
}

// getText retrieves a text secret by user UUID and name.
func (db *DB) getText(ctx context.Context, user uuid.UUID, name string) (models.Text, error) {
	const query = `SELECT name, description, user_uuid, user_text, created_at, updated_at, version from text 
                    where user_uuid=$1 and name=$2`

	var text models.Text

	row := db.pool.QueryRow(ctx, query, user, name)

	err := row.Scan(&text.Name, &text.Description, &text.UserUUID, &text.UserText, &text.CreatedAt,
		&text.UpdatedAt, &text.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.Text{}, apperrors.ErrNotFound
		default:
			return models.Text{}, fmt.Errorf("failed to scan a response row: %w", err)
		}
	}

	return text, nil
}

// updateText updates an existing text secret and increments its version.
func (db *DB) updateText(ctx context.Context, text models.Text) error {
	const query = `UPDATE text SET user_text = $1, description = $2, version = version + 1,
                updated_at = $3 WHERE user_uuid = $4 AND name = $5`

	tag, err := db.pool.Exec(ctx, query, text.UserText, text.Description, time.Now(), text.UserUUID, text.Name)
	if err != nil {
		return fmt.Errorf("failed to update text: %w", err)
	}

	rowsUpdated := tag.RowsAffected()

	if rowsUpdated == 0 {
		return fmt.Errorf("failed to update text: %w", err)
	}

	return nil
}

// deleteText removes a text secret by user UUID and name.
func (db *DB) deleteText(ctx context.Context, user uuid.UUID, name string) error {
	const query = `DELETE FROM text WHERE user_uuid = $1 AND name = $2`

	tag, err := db.pool.Exec(ctx, query, user, name)
	if err != nil {
		return fmt.Errorf("failed to delete text: %w", err)
	}

	rowsUpdated := tag.RowsAffected()

	if rowsUpdated == 0 {
		return fmt.Errorf("failed to delete text: %w", err)
	}

	return nil
}

// GetAllUserTexts retrieves all text secrets for a user (without sensitive content).
func (db *DB) GetAllUserTexts(ctx context.Context, user uuid.UUID) ([]models.Text, error) {
	const queryStmt = `SELECT name, description, user_uuid, created_at, updated_at, version from text 
                    where user_uuid=$1`

	rows, err := db.pool.Query(ctx, queryStmt, user)
	if err != nil {
		return nil, err
	}

	var userTexts []models.Text

	for rows.Next() {
		var text models.Text

		err = rows.Scan(&text.Name, &text.Description, &text.UserUUID, &text.CreatedAt,
			&text.UpdatedAt, &text.Version)
		if err != nil {
			return nil, err
		}
		userTexts = append(userTexts, text)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(userTexts) == 0 {
		return nil, apperrors.ErrNotFound
	}

	return userTexts, nil
}

// storeCredential inserts login credentials into the database.
func (db *DB) storeCredential(ctx context.Context, credential models.Credential) error {
	const query = `INSERT INTO credential (user_uuid, login, password, name, description) 
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_uuid, name) DO NOTHING`

	tag, err := db.pool.Exec(ctx, query, credential.UserUUID, credential.Login, credential.Password, credential.Name, credential.Description)
	if err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	return nil
}

// getCredential retrieves credentials by user UUID and name.
func (db *DB) getCredential(ctx context.Context, user uuid.UUID, name string) (models.Credential, error) {
	const query = `SELECT name, description, user_uuid, login, password, created_at, updated_at, version from credential 
                    where user_uuid=$1 and name=$2`

	var credential models.Credential

	row := db.pool.QueryRow(ctx, query, user, name)

	err := row.Scan(&credential.Name, &credential.Description, &credential.UserUUID, &credential.Login,
		&credential.Password, &credential.CreatedAt, &credential.UpdatedAt, &credential.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.Credential{}, apperrors.ErrNotFound
		default:
			return models.Credential{}, fmt.Errorf("failed to scan a response row: %w", err)
		}
	}

	return credential, nil
}

// updateCredential updates existing credentials and increments the version.
func (db *DB) updateCredential(ctx context.Context, credential models.Credential) error {
	const query = `UPDATE credential SET login = $1, password = $2, description = $3,
                      version = version + 1, updated_at = $4 WHERE user_uuid = $5 AND name = $6`
	tag, err := db.pool.Exec(ctx, query, credential.Login, credential.Password, credential.Description,
		credential.UpdatedAt, credential.UserUUID, credential.Name)
	if err != nil {
		return fmt.Errorf("failed to update credential: %w", err)
	}

	rowsUpdated := tag.RowsAffected()

	if rowsUpdated == 0 {
		return fmt.Errorf("failed to update credential: %w", err)
	}

	return nil
}

// deleteCredential removes credentials by user UUID and name.
func (db *DB) deleteCredential(ctx context.Context, user uuid.UUID, name string) error {
	const query = `DELETE FROM credential WHERE user_uuid = $1 AND name = $2`

	tag, err := db.pool.Exec(ctx, query, user, name)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	rowsUpdated := tag.RowsAffected()

	if rowsUpdated == 0 {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	return nil
}

// GetAllUserCredentials retrieves all credentials for a user (without sensitive content).
func (db *DB) GetAllUserCredentials(ctx context.Context, user uuid.UUID) ([]models.Credential, error) {
	const queryStmt = `SELECT name, description, user_uuid, created_at, updated_at, version from credential 
                    where user_uuid=$1`

	rows, err := db.pool.Query(ctx, queryStmt, user)
	if err != nil {
		return nil, err
	}

	var userCredentials []models.Credential

	for rows.Next() {
		var credential models.Credential

		err = rows.Scan(&credential.Name, &credential.Description, &credential.UserUUID, &credential.CreatedAt,
			&credential.UpdatedAt, &credential.Version)
		if err != nil {
			return nil, err
		}
		userCredentials = append(userCredentials, credential)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(userCredentials) == 0 {
		return nil, apperrors.ErrNotFound
	}

	return userCredentials, nil
}

// storeCard inserts bank card information into the database.
func (db *DB) storeCard(ctx context.Context, card models.Card) error {
	const query = `INSERT INTO card (user_uuid, card_number, owner, expires_at, cvc, name, description) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (user_uuid, name) DO NOTHING`

	tag, err := db.pool.Exec(ctx, query, card.UserUUID, card.CardNumber, card.Owner, card.ExpiresAt,
		card.CVC, card.Name, card.Description)
	if err != nil {
		return fmt.Errorf("failed to store card: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("failed to store card: %w", err)
	}

	return nil
}

// getCard retrieves bank card data by user UUID and name.
func (db *DB) getCard(ctx context.Context, user uuid.UUID, name string) (models.Card, error) {
	const query = `SELECT name, description, user_uuid, card_number, owner, expires_at, cvc, created_at, updated_at, 
       version from card where user_uuid=$1 and name=$2`

	var card models.Card

	row := db.pool.QueryRow(ctx, query, user, name)

	err := row.Scan(&card.Name, &card.Description, &card.UserUUID, &card.CardNumber, &card.Owner, &card.ExpiresAt,
		&card.CVC, &card.CreatedAt, &card.UpdatedAt, &card.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.Card{}, apperrors.ErrNotFound
		default:
			return models.Card{}, fmt.Errorf("failed to scan a response row: %w", err)
		}
	}

	return card, nil
}

// updateCard updates existing card information and increments the version.
func (db *DB) updateCard(ctx context.Context, card models.Card) error {
	const query = `UPDATE card SET card_number = $1, owner = $2, expires_at = $3, cvc = $4, description = $5,
                      version = version + 1, updated_at = $6 WHERE user_uuid = $7 AND name = $8`
	tag, err := db.pool.Exec(ctx, query, card.CardNumber, card.Owner, card.ExpiresAt, card.CVC,
		card.Description, card.UpdatedAt, card.UserUUID, card.Name)
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}

	rowsUpdated := tag.RowsAffected()

	if rowsUpdated == 0 {
		return fmt.Errorf("failed to update card: %w", err)
	}

	return nil
}

// deleteCard removes card information by user UUID and name.
func (db *DB) deleteCard(ctx context.Context, user uuid.UUID, name string) error {
	const query = `DELETE FROM card WHERE user_uuid = $1 AND name = $2`

	tag, err := db.pool.Exec(ctx, query, user, name)
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}

	rowsUpdated := tag.RowsAffected()

	if rowsUpdated == 0 {
		return fmt.Errorf("failed to delete card: %w", err)
	}

	return nil
}

// GetAllUserCards retrieves all cards for a user (without sensitive content).
func (db *DB) GetAllUserCards(ctx context.Context, user uuid.UUID) ([]models.Card, error) {
	const queryStmt = `SELECT name, description, user_uuid, created_at, updated_at, 
       version from card where user_uuid=$1`

	rows, err := db.pool.Query(ctx, queryStmt, user)
	if err != nil {
		return nil, err
	}

	var userCards []models.Card

	for rows.Next() {
		var card models.Card

		err = rows.Scan(&card.Name, &card.Description, &card.UserUUID, &card.CreatedAt, &card.UpdatedAt, &card.Version)
		if err != nil {
			return nil, err
		}
		userCards = append(userCards, card)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(userCards) == 0 {
		return nil, apperrors.ErrNotFound
	}

	return userCards, nil
}

// storeFile inserts a file into the database.
func (db *DB) storeFile(ctx context.Context, file models.File) error {
	const query = `INSERT INTO file (user_uuid, user_file, name, description) 
		VALUES ($1, $2, $3, $4) ON CONFLICT (user_uuid, name) DO NOTHING`

	tag, err := db.pool.Exec(ctx, query, file.UserUUID, file.UserFile, file.Name, file.Description)
	if err != nil {
		return fmt.Errorf("failed to store file: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("failed to store file: %w", err)
	}

	return nil
}

// getFile retrieves a file by user UUID and name.
func (db *DB) getFile(ctx context.Context, user uuid.UUID, name string) (models.File, error) {
	const query = `SELECT name, description, user_uuid, user_file, created_at, updated_at, 
       version from file where user_uuid=$1 and name=$2`

	var file models.File

	row := db.pool.QueryRow(ctx, query, user, name)

	err := row.Scan(&file.Name, &file.Description, &file.UserUUID, &file.UserFile, &file.CreatedAt,
		&file.UpdatedAt, &file.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.File{}, apperrors.ErrNotFound
		default:
			return models.File{}, fmt.Errorf("failed to scan a response row: %w", err)
		}
	}

	return file, nil
}
// deleteFile removes a file by user UUID and name.
func (db *DB) deleteFile(ctx context.Context, user uuid.UUID, name string) error {
	const query = `DELETE FROM file WHERE user_uuid = $1 AND name = $2`

	tag, err := db.pool.Exec(ctx, query, user, name)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rowsUpdated := tag.RowsAffected()

	if rowsUpdated == 0 {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetAllUserFiles retrieves all files for a user (without file content).
func (db *DB) GetAllUserFiles(ctx context.Context, user uuid.UUID) ([]models.File, error) {
	const queryStmt = `SELECT name, description, user_uuid,  created_at, updated_at, 
       version from file where user_uuid=$1`

	rows, err := db.pool.Query(ctx, queryStmt, user)
	if err != nil {
		return nil, err
	}

	var userFiles []models.File

	for rows.Next() {
		var file models.File

		err = rows.Scan(&file.Name, &file.Description, &file.UserUUID, &file.CreatedAt, &file.UpdatedAt, &file.Version)
		if err != nil {
			return nil, err
		}
		userFiles = append(userFiles, file)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(userFiles) == 0 {
		return nil, apperrors.ErrNotFound
	}

	return userFiles, nil
}
