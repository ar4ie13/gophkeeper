package postgresql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ar4ie13/gophkeeper/internal/apperrors"
	"github.com/ar4ie13/gophkeeper/internal/server/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDB_CreateUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()
	user := models.User{
		UUID:         userUUID,
		Login:        "testuser",
		PasswordHash: "hashed_password",
		EncryptedKey: []byte("encrypted_key"),
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO users").
			WithArgs(user.UUID, user.Login, user.PasswordHash, user.EncryptedKey).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := db.CreateUser(ctx, user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user already exists", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO users").
			WithArgs(user.UUID, user.Login, user.PasswordHash, user.EncryptedKey).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := db.CreateUser(ctx, user)
		assert.ErrorIs(t, err, apperrors.ErrUserAlreadyExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO users").
			WithArgs(user.UUID, user.Login, user.PasswordHash, user.EncryptedKey).
			WillReturnError(errors.New("database error"))

		err := db.CreateUser(ctx, user)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_GetUserByLogin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	login := "testuser"
	userUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"uuid", "login", "password_hash", "encrypted_key", "created_at", "updated_at"}).
			AddRow(userUUID, login, "hashed_password", []byte("encrypted_key"), now, now)

		mock.ExpectQuery("SELECT uuid, login, password_hash, encrypted_key, created_at, updated_at from users").
			WithArgs(login).
			WillReturnRows(rows)

		user, err := db.GetUserByLogin(ctx, login)
		assert.NoError(t, err)
		assert.Equal(t, userUUID, user.UUID)
		assert.Equal(t, login, user.Login)
		assert.Equal(t, "hashed_password", user.PasswordHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT uuid, login, password_hash, encrypted_key, created_at, updated_at from users").
			WithArgs(login).
			WillReturnError(pgx.ErrNoRows)

		user, err := db.GetUserByLogin(ctx, login)
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
		assert.Equal(t, models.User{}, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT uuid, login, password_hash, encrypted_key, created_at, updated_at from users").
			WithArgs(login).
			WillReturnError(errors.New("database error"))

		user, err := db.GetUserByLogin(ctx, login)
		assert.Error(t, err)
		assert.Equal(t, models.User{}, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_GetUserKey(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()
	encryptedKey := []byte("encrypted_key")

	t.Run("success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"encrypted_key"}).
			AddRow(encryptedKey)

		mock.ExpectQuery("SELECT encrypted_key FROM users").
			WithArgs(userUUID).
			WillReturnRows(rows)

		key, err := db.GetUserKey(ctx, userUUID)
		assert.NoError(t, err)
		assert.Equal(t, encryptedKey, key)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT encrypted_key FROM users").
			WithArgs(userUUID).
			WillReturnError(errors.New("database error"))

		key, err := db.GetUserKey(ctx, userUUID)
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_StoreSecret(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()

	t.Run("store text success", func(t *testing.T) {
		text := models.Text{
			UserUUID:    userUUID,
			UserText:    []byte("encrypted_text"),
			Name:        "test_text",
			Description: "test description",
		}

		mock.ExpectExec("INSERT INTO text").
			WithArgs(text.UserUUID, text.UserText, text.Name, text.Description).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := db.StoreSecret(ctx, "text", text)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("store credential success", func(t *testing.T) {
		credential := models.Credential{
			UserUUID:    userUUID,
			Login:       []byte("encrypted_login"),
			Password:    []byte("encrypted_password"),
			Name:        "test_credential",
			Description: "test description",
		}

		mock.ExpectExec("INSERT INTO credential").
			WithArgs(credential.UserUUID, credential.Login, credential.Password, credential.Name, credential.Description).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := db.StoreSecret(ctx, "credential", credential)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("store card success", func(t *testing.T) {
		card := models.Card{
			UserUUID:    userUUID,
			CardNumber:  []byte("encrypted_card"),
			Owner:       []byte("encrypted_owner"),
			ExpiresAt:   []byte("encrypted_expires"),
			CVC:         []byte("encrypted_cvc"),
			Name:        "test_card",
			Description: "test description",
		}

		mock.ExpectExec("INSERT INTO card").
			WithArgs(card.UserUUID, card.CardNumber, card.Owner, card.ExpiresAt, card.CVC, card.Name, card.Description).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := db.StoreSecret(ctx, "card", card)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("store file success", func(t *testing.T) {
		file := models.File{
			UserUUID:    userUUID,
			UserFile:    []byte("encrypted_file"),
			Name:        "test_file",
			Description: "test description",
		}

		mock.ExpectExec("INSERT INTO file").
			WithArgs(file.UserUUID, file.UserFile, file.Name, file.Description).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := db.StoreSecret(ctx, "file", file)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid secret type", func(t *testing.T) {
		err := db.StoreSecret(ctx, "invalid", "data")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
	})

	t.Run("wrong type assertion", func(t *testing.T) {
		err := db.StoreSecret(ctx, "text", "wrong_type")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
	})
}

func TestDB_GetSecret(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()
	now := time.Now()

	t.Run("get text success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "user_text", "created_at", "updated_at", "version"}).
			AddRow("test_text", "description", userUUID, []byte("encrypted_text"), now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid, user_text, created_at, updated_at, version from text").
			WithArgs(userUUID, "test_text").
			WillReturnRows(rows)

		secret, err := db.GetSecret(ctx, userUUID, "text", "test_text")
		assert.NoError(t, err)
		text, ok := secret.(models.Text)
		assert.True(t, ok)
		assert.Equal(t, "test_text", text.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get credential success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "login", "password", "created_at", "updated_at", "version"}).
			AddRow("test_cred", "description", userUUID, []byte("encrypted_login"), []byte("encrypted_password"), now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid, login, password, created_at, updated_at, version from credential").
			WithArgs(userUUID, "test_cred").
			WillReturnRows(rows)

		secret, err := db.GetSecret(ctx, userUUID, "credential", "test_cred")
		assert.NoError(t, err)
		credential, ok := secret.(models.Credential)
		assert.True(t, ok)
		assert.Equal(t, "test_cred", credential.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get card success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "card_number", "owner", "expires_at", "cvc", "created_at", "updated_at", "version"}).
			AddRow("test_card", "description", userUUID, []byte("encrypted_card"), []byte("encrypted_owner"), []byte("encrypted_expires"), []byte("encrypted_cvc"), now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid, card_number, owner, expires_at, cvc, created_at, updated_at").
			WithArgs(userUUID, "test_card").
			WillReturnRows(rows)

		secret, err := db.GetSecret(ctx, userUUID, "card", "test_card")
		assert.NoError(t, err)
		card, ok := secret.(models.Card)
		assert.True(t, ok)
		assert.Equal(t, "test_card", card.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get file success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "user_file", "created_at", "updated_at", "version"}).
			AddRow("test_file", "description", userUUID, []byte("encrypted_file"), now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid, user_file, created_at, updated_at").
			WithArgs(userUUID, "test_file").
			WillReturnRows(rows)

		secret, err := db.GetSecret(ctx, userUUID, "file", "test_file")
		assert.NoError(t, err)
		file, ok := secret.(models.File)
		assert.True(t, ok)
		assert.Equal(t, "test_file", file.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid secret type", func(t *testing.T) {
		secret, err := db.GetSecret(ctx, userUUID, "invalid", "test")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
		assert.Nil(t, secret)
	})

	t.Run("secret not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT name, description, user_uuid, user_text, created_at, updated_at, version from text").
			WithArgs(userUUID, "nonexistent").
			WillReturnError(pgx.ErrNoRows)

		secret, err := db.GetSecret(ctx, userUUID, "text", "nonexistent")
		assert.ErrorIs(t, err, apperrors.ErrNotFound)
		assert.Equal(t, models.Text{}, secret)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_UpdateSecret(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()

	t.Run("update text success", func(t *testing.T) {
		text := models.Text{
			UserUUID:    userUUID,
			UserText:    []byte("updated_text"),
			Name:        "test_text",
			Description: "updated description",
		}

		mock.ExpectExec("UPDATE text SET").
			WithArgs(text.UserText, text.Description, pgxmock.AnyArg(), text.UserUUID, text.Name).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := db.UpdateSecret(ctx, "text", text)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update credential success", func(t *testing.T) {
		credential := models.Credential{
			UserUUID:    userUUID,
			Login:       []byte("updated_login"),
			Password:    []byte("updated_password"),
			Name:        "test_credential",
			Description: "updated description",
			UpdatedAt:   time.Now(),
		}

		mock.ExpectExec("UPDATE credential SET").
			WithArgs(credential.Login, credential.Password, credential.Description, credential.UpdatedAt, credential.UserUUID, credential.Name).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := db.UpdateSecret(ctx, "credential", credential)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update card success", func(t *testing.T) {
		card := models.Card{
			UserUUID:    userUUID,
			CardNumber:  []byte("updated_card"),
			Owner:       []byte("updated_owner"),
			ExpiresAt:   []byte("updated_expires"),
			CVC:         []byte("updated_cvc"),
			Name:        "test_card",
			Description: "updated description",
			UpdatedAt:   time.Now(),
		}

		mock.ExpectExec("UPDATE card SET").
			WithArgs(card.CardNumber, card.Owner, card.ExpiresAt, card.CVC, card.Description, card.UpdatedAt, card.UserUUID, card.Name).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := db.UpdateSecret(ctx, "card", card)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid secret type", func(t *testing.T) {
		err := db.UpdateSecret(ctx, "invalid", "data")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
	})

	t.Run("wrong type assertion", func(t *testing.T) {
		err := db.UpdateSecret(ctx, "text", "wrong_type")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
	})

	t.Run("no rows affected", func(t *testing.T) {
		text := models.Text{
			UserUUID:    userUUID,
			UserText:    []byte("updated_text"),
			Name:        "nonexistent",
			Description: "description",
		}

		mock.ExpectExec("UPDATE text SET").
			WithArgs(text.UserText, text.Description, pgxmock.AnyArg(), text.UserUUID, text.Name).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := db.UpdateSecret(ctx, "text", text)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_DeleteSecret(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()

	t.Run("delete text success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM text").
			WithArgs(userUUID, "test_text").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := db.DeleteSecret(ctx, userUUID, "text", "test_text")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete credential success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM credential").
			WithArgs(userUUID, "test_credential").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := db.DeleteSecret(ctx, userUUID, "credential", "test_credential")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete card success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM card").
			WithArgs(userUUID, "test_card").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := db.DeleteSecret(ctx, userUUID, "card", "test_card")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete file success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM file").
			WithArgs(userUUID, "test_file").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := db.DeleteSecret(ctx, userUUID, "file", "test_file")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid secret type", func(t *testing.T) {
		err := db.DeleteSecret(ctx, userUUID, "invalid", "test")
		assert.ErrorIs(t, err, apperrors.ErrSecretInvalid)
	})

	t.Run("no rows affected", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM text").
			WithArgs(userUUID, "nonexistent").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := db.DeleteSecret(ctx, userUUID, "text", "nonexistent")
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_GetAllUserTexts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"}).
			AddRow("text1", "desc1", userUUID, now, now, 1).
			AddRow("text2", "desc2", userUUID, now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid, created_at, updated_at, version from text").
			WithArgs(userUUID).
			WillReturnRows(rows)

		texts, err := db.GetAllUserTexts(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, texts, 2)
		assert.Equal(t, "text1", texts[0].Name)
		assert.Equal(t, "text2", texts[1].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no texts found", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"})

		mock.ExpectQuery("SELECT name, description, user_uuid, created_at, updated_at, version from text").
			WithArgs(userUUID).
			WillReturnRows(rows)

		texts, err := db.GetAllUserTexts(ctx, userUUID)
		assert.ErrorIs(t, err, apperrors.ErrNotFound)
		assert.Nil(t, texts)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT name, description, user_uuid, created_at, updated_at, version from text").
			WithArgs(userUUID).
			WillReturnError(errors.New("database error"))

		texts, err := db.GetAllUserTexts(ctx, userUUID)
		assert.Error(t, err)
		assert.Nil(t, texts)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_GetAllUserCredentials(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"}).
			AddRow("cred1", "desc1", userUUID, now, now, 1).
			AddRow("cred2", "desc2", userUUID, now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid, created_at, updated_at, version from credential").
			WithArgs(userUUID).
			WillReturnRows(rows)

		credentials, err := db.GetAllUserCredentials(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, credentials, 2)
		assert.Equal(t, "cred1", credentials[0].Name)
		assert.Equal(t, "cred2", credentials[1].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no credentials found", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"})

		mock.ExpectQuery("SELECT name, description, user_uuid, created_at, updated_at, version from credential").
			WithArgs(userUUID).
			WillReturnRows(rows)

		credentials, err := db.GetAllUserCredentials(ctx, userUUID)
		assert.ErrorIs(t, err, apperrors.ErrNotFound)
		assert.Nil(t, credentials)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_GetAllUserCards(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"}).
			AddRow("card1", "desc1", userUUID, now, now, 1).
			AddRow("card2", "desc2", userUUID, now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid, created_at, updated_at").
			WithArgs(userUUID).
			WillReturnRows(rows)

		cards, err := db.GetAllUserCards(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, cards, 2)
		assert.Equal(t, "card1", cards[0].Name)
		assert.Equal(t, "card2", cards[1].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no cards found", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"})

		mock.ExpectQuery("SELECT name, description, user_uuid, created_at, updated_at").
			WithArgs(userUUID).
			WillReturnRows(rows)

		cards, err := db.GetAllUserCards(ctx, userUUID)
		assert.ErrorIs(t, err, apperrors.ErrNotFound)
		assert.Nil(t, cards)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_GetAllUserFiles(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	ctx := context.Background()
	userUUID := uuid.New()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"}).
			AddRow("file1", "desc1", userUUID, now, now, 1).
			AddRow("file2", "desc2", userUUID, now, now, 1)

		mock.ExpectQuery("SELECT name, description, user_uuid,  created_at, updated_at").
			WithArgs(userUUID).
			WillReturnRows(rows)

		files, err := db.GetAllUserFiles(ctx, userUUID)
		assert.NoError(t, err)
		assert.Len(t, files, 2)
		assert.Equal(t, "file1", files[0].Name)
		assert.Equal(t, "file2", files[1].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no files found", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "description", "user_uuid", "created_at", "updated_at", "version"})

		mock.ExpectQuery("SELECT name, description, user_uuid,  created_at, updated_at").
			WithArgs(userUUID).
			WillReturnRows(rows)

		files, err := db.GetAllUserFiles(ctx, userUUID)
		assert.ErrorIs(t, err, apperrors.ErrNotFound)
		assert.Nil(t, files)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDB_Close(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	db := &DB{
		pool: mock.(PgxPool),
		zlog: zerolog.Nop(),
	}

	// Close should not return error
	err = db.Close()
	assert.NoError(t, err)
}
