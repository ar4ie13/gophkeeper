package tui

import (
	"github.com/ar4ie13/gophkeeper/internal/client/models"
	"github.com/ar4ie13/gophkeeper/internal/client/service"
)

// --- Bubble Tea messages (commands return these) ---

// authDoneMsg signals that authentication (login/register) has completed.
type authDoneMsg struct {
	err error
}

// syncDoneMsg signals that server synchronization has completed.
type syncDoneMsg struct {
	result service.SyncResult
}

// storeTextDoneMsg signals that a text secret store operation has completed.
type storeTextDoneMsg struct {
	err error
}

// storeCredDoneMsg signals that a credential secret store operation has completed.
type storeCredDoneMsg struct {
	err error
}

// storeCardDoneMsg signals that a card secret store operation has completed.
type storeCardDoneMsg struct{
	err error
}

// storeFileDoneMsg signals that a file secret store operation has completed.
type storeFileDoneMsg struct {
	err error
}

// updateTextDoneMsg signals that a text secret update operation has completed.
type updateTextDoneMsg struct {
	err error
}

// updateCredDoneMsg signals that a credential secret update operation has completed.
type updateCredDoneMsg struct {
	err error
}

// updateCardDoneMsg signals that a card secret update operation has completed.
type updateCardDoneMsg struct {
	err error
}

// deleteSecretDoneMsg signals that a secret deletion operation has completed.
type deleteSecretDoneMsg struct {
	err error
}

// viewTextMsg signals that a text secret has been fetched for viewing.
type viewTextMsg struct {
	secret *models.TextSecret
	err    error
}

// viewCredMsg signals that a credential secret has been fetched for viewing.
type viewCredMsg struct {
	secret *models.CredentialSecret
	err    error
}

// viewCardMsg signals that a card secret has been fetched for viewing.
type viewCardMsg struct {
	secret *models.CardSecret
	err    error
}

// viewFileMsg signals that a file secret has been fetched for viewing.
type viewFileMsg struct {
	secret *models.FileSecret
	err    error
}
