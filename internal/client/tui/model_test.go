package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ar4ie13/gophkeeper/internal/client/api"
	"github.com/ar4ie13/gophkeeper/internal/client/config"
	"github.com/ar4ie13/gophkeeper/internal/client/models"
	"github.com/ar4ie13/gophkeeper/internal/client/service"
	"github.com/ar4ie13/gophkeeper/internal/client/storage"
)

// ---------- helpers ---------------------------------------------------------

var testCfg = &config.Config{Version: "1.0", BuildDate: "2026-01-01", BuildCommit: "abc"}

// minimalSvc creates a Service backed by an empty in-memory cache.
// It never makes real HTTP calls because ListSecrets reads only the cache.
func minimalSvc(t *testing.T) *service.Service {
	t.Helper()
	c, err := api.NewClient(config.Config{ServerURL: "http://127.0.0.1:1"})
	require.NoError(t, err)
	return service.NewService(c, storage.NewCache())
}

// newModel returns a fresh model at screenAuthChoice with a real svc+cfg.
func newModel(t *testing.T) Model {
	t.Helper()
	return New(minimalSvc(t), testCfg)
}

// update is a convenience wrapper that casts the returned tea.Model back.
func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	m2, cmd := m.Update(msg)
	return m2.(Model), cmd
}

// key builds a KeyMsg from a string, matching how bubbletea does it for
// special keys (ctrl+c, esc, enter, tab, …) and printable runes.
func key(s string) tea.KeyMsg {
	switch s {
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// ---------- New / Init ------------------------------------------------------

func TestNew_InitialScreen(t *testing.T) {
	m := newModel(t)
	assert.Equal(t, screenAuthChoice, m.screen)
	assert.Nil(t, m.Init())
}

// ---------- WindowSizeMsg ---------------------------------------------------

func TestUpdate_WindowSize(t *testing.T) {
	m, _ := update(newModel(t), tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

// assertQuit executes cmd and asserts it produces tea.QuitMsg.
func assertQuit(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	require.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok, "expected tea.QuitMsg from cmd")
}

// ---------- ctrl+c anywhere quits -------------------------------------------

func TestUpdate_CtrlC_Quits(t *testing.T) {
	_, cmd := update(newModel(t), key("ctrl+c"))
	assertQuit(t, cmd)
}

// ---------- screenAuthChoice ------------------------------------------------

func TestAuthChoice_DownMovesDown(t *testing.T) {
	m, _ := update(newModel(t), key("down"))
	assert.Equal(t, 1, m.cursor)
}

func TestAuthChoice_DownClamps(t *testing.T) {
	m := newModel(t)
	m.cursor = len(authChoiceItems) - 1
	m2, _ := update(m, key("down"))
	assert.Equal(t, len(authChoiceItems)-1, m2.cursor)
}

func TestAuthChoice_JMovesDown(t *testing.T) {
	m, _ := update(newModel(t), key("j"))
	assert.Equal(t, 1, m.cursor)
}

func TestAuthChoice_UpClamps(t *testing.T) {
	m, _ := update(newModel(t), key("up"))
	assert.Equal(t, 0, m.cursor)
}

func TestAuthChoice_KMovesUp(t *testing.T) {
	m := newModel(t)
	m.cursor = 1
	m2, _ := update(m, key("k"))
	assert.Equal(t, 0, m2.cursor)
}

func TestAuthChoice_EnterLogin(t *testing.T) {
	m, _ := update(newModel(t), key("enter")) // cursor=0 → Login
	assert.Equal(t, screenAuthForm, m.screen)
	assert.False(t, m.authIsRegister)
	assert.Equal(t, []string{"", ""}, m.formFields)
}

func TestAuthChoice_EnterRegister(t *testing.T) {
	m := newModel(t)
	m.cursor = 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenAuthForm, m2.screen)
	assert.True(t, m2.authIsRegister)
}

func TestAuthChoice_QQuits(t *testing.T) {
	_, cmd := update(newModel(t), key("q"))
	assertQuit(t, cmd)
}

// ---------- screenAuthForm --------------------------------------------------

func authFormModel(t *testing.T, isRegister bool) Model {
	m := newModel(t)
	m.cursor = 0
	if isRegister {
		m.cursor = 1
	}
	m, _ = update(m, key("enter"))
	require.Equal(t, screenAuthForm, m.screen)
	return m
}

func TestAuthForm_EscGoesBack(t *testing.T) {
	m, _ := update(authFormModel(t, false), key("esc"))
	assert.Equal(t, screenAuthChoice, m.screen)
	assert.Equal(t, 0, m.cursor)
}

func TestAuthForm_TabAdvancesFocus(t *testing.T) {
	m, _ := update(authFormModel(t, false), key("tab"))
	assert.Equal(t, 1, m.formFocus)
}

func TestAuthForm_ShiftTabDecreasesFocus(t *testing.T) {
	m := authFormModel(t, false)
	m.formFocus = 1
	m2, _ := update(m, key("shift+tab"))
	assert.Equal(t, 0, m2.formFocus)
}

func TestAuthForm_TabWrapsAround(t *testing.T) {
	m := authFormModel(t, false)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("tab"))
	assert.Equal(t, 0, m2.formFocus)
}

func TestAuthForm_EnterOnFirstFieldAdvancesFocus(t *testing.T) {
	m, _ := update(authFormModel(t, false), key("enter"))
	assert.Equal(t, 1, m.formFocus)
}

func TestAuthForm_EnterEmptyLoginShowsError(t *testing.T) {
	m := authFormModel(t, false)
	m.formFocus = 1 // on last field
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Login")
}

func TestAuthForm_EnterEmptyPasswordShowsError(t *testing.T) {
	m := authFormModel(t, false)
	m.formFields[0] = "user"
	m.formFocus = 1 // on last field
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Password")
}

func TestAuthForm_EnterValidFieldsReturnsCmd(t *testing.T) {
	m := authFormModel(t, false)
	m.formFields = []string{"user", "pass"}
	m.formFocus = 1
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

func TestAuthForm_BackspaceRemovesChar(t *testing.T) {
	m := authFormModel(t, false)
	m.formFields[0] = "abc"
	m2, _ := update(m, key("backspace"))
	assert.Equal(t, "ab", m2.formFields[0])
}

func TestAuthForm_BackspaceOnEmptyFieldIsNoop(t *testing.T) {
	m := authFormModel(t, false)
	m.formFields[0] = ""
	m2, _ := update(m, key("backspace"))
	assert.Equal(t, "", m2.formFields[0])
}

func TestAuthForm_RuneAppended(t *testing.T) {
	m := authFormModel(t, false)
	m2, _ := update(m, key("a"))
	assert.Equal(t, "a", m2.formFields[0])
}

func TestAuthForm_RunesClearFormErr(t *testing.T) {
	m := authFormModel(t, false)
	m.formErr = "some error"
	m2, _ := update(m, key("x"))
	assert.Empty(t, m2.formErr)
}

// ---------- handleAuthDone --------------------------------------------------

func TestHandleAuthDone_Error(t *testing.T) {
	m := authFormModel(t, false)
	m.formFields = []string{"u", "p"}
	m2, _ := update(m, authDoneMsg{err: errors.New("bad credentials")})
	assert.Contains(t, m2.formErr, "bad credentials")
	assert.Equal(t, screenAuthForm, m2.screen)
}

func TestHandleAuthDone_Success(t *testing.T) {
	m := authFormModel(t, false)
	m.formFields = []string{"alice", "pass"}
	m2, cmd := update(m, authDoneMsg{err: nil})
	assert.Equal(t, screenSync, m2.screen)
	assert.Equal(t, "alice", m2.authUser)
	assert.True(t, m2.syncing)
	assert.NotNil(t, cmd) // syncCmd
}

// ---------- handleSyncDone --------------------------------------------------

func syncModel(t *testing.T) Model {
	m := authFormModel(t, false)
	m.formFields = []string{"alice", "pass"}
	m2, _ := update(m, authDoneMsg{err: nil})
	return m2
}

func TestHandleSyncDone_OnlineNoError(t *testing.T) {
	m := syncModel(t)
	m2, _ := update(m, syncDoneMsg{result: service.SyncResult{
		Online: true, TextCount: 2, CredentialCount: 1,
	}})
	assert.Equal(t, screenMain, m2.screen)
	assert.True(t, m2.online)
	assert.False(t, m2.syncing)
	assert.Contains(t, m2.success, "2 texts")
	assert.Empty(t, m2.err)
}

func TestHandleSyncDone_OnlineWithError(t *testing.T) {
	m := syncModel(t)
	m2, _ := update(m, syncDoneMsg{result: service.SyncResult{
		Online: true, Error: errors.New("partial failure"),
	}})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.err, "partial failure")
}

func TestHandleSyncDone_Offline(t *testing.T) {
	m := syncModel(t)
	m2, _ := update(m, syncDoneMsg{result: service.SyncResult{Online: false}})
	assert.Equal(t, screenMain, m2.screen)
	assert.False(t, m2.online)
	assert.Empty(t, m2.success) // no sync count shown offline
}

// ---------- screenMain ------------------------------------------------------

func mainModel(t *testing.T) Model {
	m := syncModel(t)
	m2, _ := update(m, syncDoneMsg{result: service.SyncResult{Online: true}})
	require.Equal(t, screenMain, m2.screen)
	return m2
}

func TestMain_DownMovesDown(t *testing.T) {
	m, _ := update(mainModel(t), key("down"))
	assert.Equal(t, 1, m.cursor)
}

func TestMain_DownClamps(t *testing.T) {
	m := mainModel(t)
	m.cursor = len(mainMenuItems) - 1
	m2, _ := update(m, key("down"))
	assert.Equal(t, len(mainMenuItems)-1, m2.cursor)
}

func TestMain_UpClamps(t *testing.T) {
	m, _ := update(mainModel(t), key("up"))
	assert.Equal(t, 0, m.cursor)
}

func TestMain_EnterViewList(t *testing.T) {
	m := mainModel(t)
	m.cursor = 0
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenList, m2.screen)
}

func TestMain_EnterAddSecret(t *testing.T) {
	m := mainModel(t)
	m.cursor = 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenAddSecretList, m2.screen)
}

func TestMain_EnterResync(t *testing.T) {
	m := mainModel(t)
	m.cursor = 2
	m2, cmd := update(m, key("enter"))
	assert.Equal(t, screenSync, m2.screen)
	assert.NotNil(t, cmd)
}

func TestMain_EnterLogout(t *testing.T) {
	m := mainModel(t)
	m.cursor = 3
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenAuthChoice, m2.screen)
	assert.False(t, m2.online)
	assert.Empty(t, m2.authUser)
}

func TestMain_QQuits(t *testing.T) {
	_, cmd := update(mainModel(t), key("q"))
	assertQuit(t, cmd)
}

// ---------- screenAddSecretList ---------------------------------------------

func addSecretModel(t *testing.T) Model {
	m := mainModel(t)
	m.cursor = 1
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenAddSecretList, m2.screen)
	return m2
}

func TestAddSecretList_DownUp(t *testing.T) {
	m, _ := update(addSecretModel(t), key("down"))
	assert.Equal(t, 1, m.cursor)
	m2, _ := update(m, key("up"))
	assert.Equal(t, 0, m2.cursor)
}

func TestAddSecretList_EscGoesBack(t *testing.T) {
	m, _ := update(addSecretModel(t), key("esc"))
	assert.Equal(t, screenMain, m.screen)
}

func TestAddSecretList_QGoesBack(t *testing.T) {
	m, _ := update(addSecretModel(t), key("q"))
	assert.Equal(t, screenMain, m.screen)
}

func TestAddSecretList_EnterAddText(t *testing.T) {
	m := addSecretModel(t)
	m.cursor = 0
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenAddText, m2.screen)
	assert.Len(t, m2.formFields, 3)
}

func TestAddSecretList_EnterAddCred(t *testing.T) {
	m := addSecretModel(t)
	m.cursor = 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenAddCred, m2.screen)
	assert.Len(t, m2.formFields, 4)
}

func TestAddSecretList_EnterAddCard(t *testing.T) {
	m := addSecretModel(t)
	m.cursor = 2
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenAddCard, m2.screen)
	assert.Len(t, m2.formFields, 6)
}

func TestAddSecretList_EnterAddFile(t *testing.T) {
	m := addSecretModel(t)
	m.cursor = 3
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenAddFile, m2.screen)
	assert.Len(t, m2.formFields, 3)
}

// ---------- screenList ------------------------------------------------------

func listModel(t *testing.T, entries ...models.SecretEntry) Model {
	m := mainModel(t)
	m.screen = screenList
	m.entries = entries
	m.listCursor = 0
	return m
}

func TestList_DownMovesListCursor(t *testing.T) {
	m := listModel(t,
		models.SecretEntry{Name: "a", Type: models.SecretTypeText},
		models.SecretEntry{Name: "b", Type: models.SecretTypeText},
	)
	m2, _ := update(m, key("down"))
	assert.Equal(t, 1, m2.listCursor)
}

func TestList_DownClampsAtEnd(t *testing.T) {
	m := listModel(t, models.SecretEntry{Name: "a", Type: models.SecretTypeText})
	m.listCursor = 0
	m2, _ := update(m, key("down"))
	assert.Equal(t, 0, m2.listCursor)
}

func TestList_UpClampsAtZero(t *testing.T) {
	m := listModel(t, models.SecretEntry{Name: "a", Type: models.SecretTypeText})
	m2, _ := update(m, key("up"))
	assert.Equal(t, 0, m2.listCursor)
}

func TestList_EscGoesBack(t *testing.T) {
	m, _ := update(listModel(t), key("esc"))
	assert.Equal(t, screenMain, m.screen)
}

func TestList_QGoesBack(t *testing.T) {
	m, _ := update(listModel(t), key("q"))
	assert.Equal(t, screenMain, m.screen)
}

func TestList_EnterOnEmptyIsNoop(t *testing.T) {
	m, cmd := update(listModel(t), key("enter"))
	assert.Equal(t, screenList, m.screen)
	assert.Nil(t, cmd)
}

func TestList_EnterOnItemReturnsCmd(t *testing.T) {
	m := listModel(t, models.SecretEntry{Name: "note", Type: models.SecretTypeText})
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

func TestList_DeleteOnItemGoesToConfirmDelete(t *testing.T) {
	m := listModel(t, models.SecretEntry{Name: "note", Type: models.SecretTypeText})
	m2, _ := update(m, key("delete"))
	assert.Equal(t, screenConfirmDelete, m2.screen)
	assert.Equal(t, "note", m2.viewSecretName)
	assert.Equal(t, 1, m2.confirmCursor) // default No
}

func TestList_DeleteOnEmptyListIsNoop(t *testing.T) {
	m, _ := update(listModel(t), key("delete"))
	assert.Equal(t, screenList, m.screen)
}

// ---------- screenAddText ---------------------------------------------------

func addTextModel(t *testing.T) Model {
	m := addSecretModel(t)
	m.cursor = 0
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenAddText, m2.screen)
	return m2
}

func TestAddText_EscGoesBack(t *testing.T) {
	m, _ := update(addTextModel(t), key("esc"))
	assert.Equal(t, screenAddSecretList, m.screen)
}

func TestAddText_TabAdvancesFocus(t *testing.T) {
	m, _ := update(addTextModel(t), key("tab"))
	assert.Equal(t, 1, m.formFocus)
}

func TestAddText_EnterOnFirstFieldAdvancesFocus(t *testing.T) {
	m, _ := update(addTextModel(t), key("enter"))
	assert.Equal(t, 1, m.formFocus)
}

func TestAddText_EnterEmptyNameShowsError(t *testing.T) {
	m := addTextModel(t)
	m.formFocus = len(m.formFields) - 1 // last field
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Name")
}

func TestAddText_EnterTooLongTextShowsError(t *testing.T) {
	m := addTextModel(t)
	m.formFields[0] = "name"
	m.formFields[2] = strings.Repeat("x", maxTextLen+1)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "too long")
}

func TestAddText_TypeAtMaxLimitShowsError(t *testing.T) {
	m := addTextModel(t)
	m.formFocus = 2
	m.formFields[2] = strings.Repeat("x", maxTextLen) // already at limit
	m2, _ := update(m, key("a"))
	assert.Contains(t, m2.formErr, "Maximum")
}

func TestAddText_ValidFormReturnsCmd(t *testing.T) {
	m := addTextModel(t)
	m.formFields[0] = "note"
	m.formFocus = len(m.formFields) - 1
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

func TestAddText_BackspaceRemovesChar(t *testing.T) {
	m := addTextModel(t)
	m.formFields[0] = "abc"
	m2, _ := update(m, key("backspace"))
	assert.Equal(t, "ab", m2.formFields[0])
}

// ---------- screenAddCred ---------------------------------------------------

func addCredModel(t *testing.T) Model {
	m := addSecretModel(t)
	m.cursor = 1
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenAddCred, m2.screen)
	return m2
}

func TestAddCred_EscGoesBack(t *testing.T) {
	m, _ := update(addCredModel(t), key("esc"))
	assert.Equal(t, screenAddSecretList, m.screen)
}

func TestAddCred_EmptyNameShowsError(t *testing.T) {
	m := addCredModel(t)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Name")
}

func TestAddCred_EmptyLoginPasswordShowsError(t *testing.T) {
	m := addCredModel(t)
	m.formFields[0] = "gh"
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Login")
}

func TestAddCred_ValidFormReturnsCmd(t *testing.T) {
	m := addCredModel(t)
	m.formFields = []string{"gh", "desc", "user", "pass"}
	m.formFocus = len(m.formFields) - 1
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

// ---------- screenAddCard ---------------------------------------------------

func addCardModel(t *testing.T) Model {
	m := addSecretModel(t)
	m.cursor = 2
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenAddCard, m2.screen)
	return m2
}

func TestAddCard_EscGoesBack(t *testing.T) {
	m, _ := update(addCardModel(t), key("esc"))
	assert.Equal(t, screenAddSecretList, m.screen)
}

func TestAddCard_EmptyNameShowsError(t *testing.T) {
	m := addCardModel(t)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Name")
}

func TestAddCard_EmptyCardNumberOwnerShowsError(t *testing.T) {
	m := addCardModel(t)
	m.formFields[0] = "visa"
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "CardNumber")
}

func TestAddCard_ValidFormReturnsCmd(t *testing.T) {
	m := addCardModel(t)
	m.formFields = []string{"visa", "desc", "1234", "Me", "12/26", "123"}
	m.formFocus = len(m.formFields) - 1
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

// ---------- screenAddFile ---------------------------------------------------

func addFileModel(t *testing.T) Model {
	m := addSecretModel(t)
	m.cursor = 3
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenAddFile, m2.screen)
	return m2
}

func TestAddFile_EscGoesBack(t *testing.T) {
	m, _ := update(addFileModel(t), key("esc"))
	assert.Equal(t, screenAddSecretList, m.screen)
}

func TestAddFile_EmptyNameShowsError(t *testing.T) {
	m := addFileModel(t)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Name")
}

func TestAddFile_ValidFormReturnsCmd(t *testing.T) {
	m := addFileModel(t)
	m.formFields = []string{"doc", "desc", "/tmp/file.pdf"}
	m.formFocus = len(m.formFields) - 1
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

// ---------- handleViewText / screenViewText ---------------------------------

func viewTextModel(t *testing.T) Model {
	secret := &models.TextSecret{Name: "note", Description: "d", Text: "hello"}
	m := mainModel(t)
	m2, _ := update(m, viewTextMsg{secret: secret})
	require.Equal(t, screenViewText, m2.screen)
	return m2
}

func TestViewText_MsgError(t *testing.T) {
	m, _ := update(mainModel(t), viewTextMsg{err: errors.New("fetch failed")})
	assert.Contains(t, m.err, "fetch failed")
	assert.Equal(t, screenMain, m.screen)
}

func TestViewText_EscGoesBackToList(t *testing.T) {
	m, _ := update(viewTextModel(t), key("esc"))
	assert.Equal(t, screenList, m.screen)
	assert.Nil(t, m.viewText)
}

func TestViewText_QGoesBackToList(t *testing.T) {
	m, _ := update(viewTextModel(t), key("q"))
	assert.Equal(t, screenList, m.screen)
}

func TestViewText_DownMovesCursor(t *testing.T) {
	m, _ := update(viewTextModel(t), key("down"))
	assert.Equal(t, 1, m.viewCursor)
}

func TestViewText_DownClamps(t *testing.T) {
	m := viewTextModel(t)
	m.viewCursor = 2
	m2, _ := update(m, key("down"))
	assert.Equal(t, 2, m2.viewCursor)
}

func TestViewText_UpClamps(t *testing.T) {
	m, _ := update(viewTextModel(t), key("up"))
	assert.Equal(t, 0, m.viewCursor)
}

func TestViewText_EnterOnNameFieldIsNoop(t *testing.T) {
	m := viewTextModel(t)
	m.viewCursor = 0
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenViewText, m2.screen)
}

func TestViewText_EnterOnEditableFieldGoesToModify(t *testing.T) {
	m := viewTextModel(t)
	m.viewCursor = 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenModifyText, m2.screen)
	assert.Equal(t, 1, m2.formFocus)
}

func TestViewText_DeleteGoesToConfirmDelete(t *testing.T) {
	m, _ := update(viewTextModel(t), key("delete"))
	assert.Equal(t, screenConfirmDelete, m.screen)
	assert.Equal(t, 1, m.confirmCursor) // default No
}

func TestViewText_NilViewTextIsNoop(t *testing.T) {
	m := mainModel(t)
	m.screen = screenViewText
	m.viewText = nil
	m2, _ := update(m, key("down"))
	assert.Equal(t, screenViewText, m2.screen)
}

// ---------- handleViewCred / screenViewCred ---------------------------------

func viewCredModel(t *testing.T) Model {
	secret := &models.CredentialSecret{Name: "gh", Login: "u", Password: "p"}
	m := mainModel(t)
	m2, _ := update(m, viewCredMsg{secret: secret})
	require.Equal(t, screenViewCred, m2.screen)
	return m2
}

func TestViewCred_EscGoesBackToList(t *testing.T) {
	m, _ := update(viewCredModel(t), key("esc"))
	assert.Equal(t, screenList, m.screen)
	assert.Nil(t, m.viewCred)
}

func TestViewCred_DownClampsAt3(t *testing.T) {
	m := viewCredModel(t)
	m.viewCursor = 3
	m2, _ := update(m, key("down"))
	assert.Equal(t, 3, m2.viewCursor)
}

func TestViewCred_EnterOnNameIsNoop(t *testing.T) {
	m := viewCredModel(t)
	m.viewCursor = 0
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenViewCred, m2.screen)
}

func TestViewCred_EnterOnEditableGoesToModify(t *testing.T) {
	m := viewCredModel(t)
	m.viewCursor = 2
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenModifyCred, m2.screen)
}

func TestViewCred_DeleteGoesToConfirmDelete(t *testing.T) {
	m, _ := update(viewCredModel(t), key("delete"))
	assert.Equal(t, screenConfirmDelete, m.screen)
}

func TestViewCred_MsgError(t *testing.T) {
	m, _ := update(mainModel(t), viewCredMsg{err: errors.New("not found")})
	assert.Contains(t, m.err, "not found")
}

// ---------- handleViewCard / screenViewCard ---------------------------------

func viewCardModel(t *testing.T) Model {
	secret := &models.CardSecret{Name: "visa", CardNumber: "1234", Owner: "Me"}
	m := mainModel(t)
	m2, _ := update(m, viewCardMsg{secret: secret})
	require.Equal(t, screenViewCard, m2.screen)
	return m2
}

func TestViewCard_EscGoesBackToList(t *testing.T) {
	m, _ := update(viewCardModel(t), key("esc"))
	assert.Equal(t, screenList, m.screen)
	assert.Nil(t, m.viewCard)
}

func TestViewCard_DownClampsAt5(t *testing.T) {
	m := viewCardModel(t)
	m.viewCursor = 5
	m2, _ := update(m, key("down"))
	assert.Equal(t, 5, m2.viewCursor)
}

func TestViewCard_EnterOnEditableGoesToModify(t *testing.T) {
	m := viewCardModel(t)
	m.viewCursor = 3
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenModifyCard, m2.screen)
	assert.Equal(t, 3, m2.formFocus)
}

func TestViewCard_DeleteGoesToConfirmDelete(t *testing.T) {
	m, _ := update(viewCardModel(t), key("delete"))
	assert.Equal(t, screenConfirmDelete, m.screen)
}

func TestViewCard_MsgError(t *testing.T) {
	m, _ := update(mainModel(t), viewCardMsg{err: errors.New("oops")})
	assert.Contains(t, m.err, "oops")
}

// ---------- handleViewFile / screenViewFile ---------------------------------

func viewFileModel(t *testing.T) Model {
	secret := &models.FileSecret{Name: "report", Path: "/tmp/r.pdf"}
	m := mainModel(t)
	m2, _ := update(m, viewFileMsg{secret: secret})
	require.Equal(t, screenViewFile, m2.screen)
	return m2
}

func TestViewFile_EscGoesBackToList(t *testing.T) {
	m, _ := update(viewFileModel(t), key("esc"))
	assert.Equal(t, screenList, m.screen)
	assert.Nil(t, m.viewFile)
}

func TestViewFile_QGoesBackToList(t *testing.T) {
	m, _ := update(viewFileModel(t), key("q"))
	assert.Equal(t, screenList, m.screen)
}

func TestViewFile_DeleteGoesToConfirmDelete(t *testing.T) {
	m, _ := update(viewFileModel(t), key("delete"))
	assert.Equal(t, screenConfirmDelete, m.screen)
}

func TestViewFile_MsgError(t *testing.T) {
	m, _ := update(mainModel(t), viewFileMsg{err: errors.New("gone")})
	assert.Contains(t, m.err, "gone")
}

// ---------- screenModifyText ------------------------------------------------

func modifyTextModel(t *testing.T) Model {
	m := viewTextModel(t)
	m.viewCursor = 1
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenModifyText, m2.screen)
	return m2
}

func TestModifyText_EscGoesBack(t *testing.T) {
	m := modifyTextModel(t)
	prev := m.prevScreen
	m2, _ := update(m, key("esc"))
	assert.Equal(t, prev, m2.screen)
}

func TestModifyText_TabSkipsNameField(t *testing.T) {
	m := modifyTextModel(t)
	m.formFocus = len(m.formFields) - 1 // wrap around to 0 → should skip to 1
	m2, _ := update(m, key("tab"))
	assert.Equal(t, 1, m2.formFocus)
}

func TestModifyText_ShiftTabSkipsNameField(t *testing.T) {
	m := modifyTextModel(t)
	m.formFocus = 1 // back would go to 0 → should skip to last
	m2, _ := update(m, key("shift+tab"))
	assert.Equal(t, len(m.formFields)-1, m2.formFocus)
}

func TestModifyText_TypeOnNameFieldIsNoop(t *testing.T) {
	m := modifyTextModel(t)
	m.formFocus = 0
	before := m.formFields[0]
	m2, _ := update(m, key("z"))
	assert.Equal(t, before, m2.formFields[0])
}

func TestModifyText_BackspaceOnNameFieldIsNoop(t *testing.T) {
	m := modifyTextModel(t)
	m.formFocus = 0
	before := m.formFields[0]
	m2, _ := update(m, key("backspace"))
	assert.Equal(t, before, m2.formFields[0])
}

func TestModifyText_TooLongShowsError(t *testing.T) {
	m := modifyTextModel(t)
	m.formFields[2] = strings.Repeat("x", maxTextLen+1)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "too long")
}

func TestModifyText_ValidEnterGoesToConfirmModify(t *testing.T) {
	m := modifyTextModel(t)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenConfirmModify, m2.screen)
	assert.Equal(t, 0, m2.confirmCursor) // default Yes
}

// ---------- screenModifyCred ------------------------------------------------

func modifyCredModel(t *testing.T) Model {
	m := viewCredModel(t)
	m.viewCursor = 2
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenModifyCred, m2.screen)
	return m2
}

func TestModifyCred_EscGoesBack(t *testing.T) {
	m := modifyCredModel(t)
	prev := m.prevScreen
	m2, _ := update(m, key("esc"))
	assert.Equal(t, prev, m2.screen)
}

func TestModifyCred_EmptyLoginPasswordShowsError(t *testing.T) {
	m := modifyCredModel(t)
	m.formFields[2] = ""
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "Login")
}

func TestModifyCred_ValidEnterGoesToConfirmModify(t *testing.T) {
	m := modifyCredModel(t)
	m.formFields[2] = "user"
	m.formFields[3] = "pass"
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenConfirmModify, m2.screen)
}

// ---------- screenModifyCard ------------------------------------------------

func modifyCardModel(t *testing.T) Model {
	m := viewCardModel(t)
	m.viewCursor = 3
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenModifyCard, m2.screen)
	return m2
}

func TestModifyCard_EscGoesBack(t *testing.T) {
	m := modifyCardModel(t)
	prev := m.prevScreen
	m2, _ := update(m, key("esc"))
	assert.Equal(t, prev, m2.screen)
}

func TestModifyCard_EmptyCardNumberOwnerShowsError(t *testing.T) {
	m := modifyCardModel(t)
	m.formFields[2] = ""
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Contains(t, m2.formErr, "CardNumber")
}

func TestModifyCard_ValidEnterGoesToConfirmModify(t *testing.T) {
	m := modifyCardModel(t)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, screenConfirmModify, m2.screen)
}

// ---------- screenConfirmModify ---------------------------------------------

func confirmModifyModel(t *testing.T) Model {
	m := modifyTextModel(t)
	m.formFocus = len(m.formFields) - 1
	m2, _ := update(m, key("enter"))
	require.Equal(t, screenConfirmModify, m2.screen)
	return m2
}

func TestConfirmModify_LeftSelectsYes(t *testing.T) {
	m := confirmModifyModel(t)
	m.confirmCursor = 1
	m2, _ := update(m, key("left"))
	assert.Equal(t, 0, m2.confirmCursor)
}

func TestConfirmModify_HSelectsYes(t *testing.T) {
	m := confirmModifyModel(t)
	m.confirmCursor = 1
	m2, _ := update(m, key("h"))
	assert.Equal(t, 0, m2.confirmCursor)
}

func TestConfirmModify_RightSelectsNo(t *testing.T) {
	m := confirmModifyModel(t)
	m.confirmCursor = 0
	m2, _ := update(m, key("right"))
	assert.Equal(t, 1, m2.confirmCursor)
}

func TestConfirmModify_EnterNoGoesBack(t *testing.T) {
	m := confirmModifyModel(t)
	m.confirmCursor = 1 // No
	m2, _ := update(m, key("enter"))
	assert.Equal(t, m.prevScreen, m2.screen)
}

func TestConfirmModify_EnterYesReturnsCmd(t *testing.T) {
	m := confirmModifyModel(t)
	m.confirmCursor = 0 // Yes
	m.prevScreen = screenModifyText
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

func TestConfirmModify_EscGoesBack(t *testing.T) {
	m := confirmModifyModel(t)
	prev := m.prevScreen
	m2, _ := update(m, key("esc"))
	assert.Equal(t, prev, m2.screen)
}

// ---------- screenConfirmDelete ---------------------------------------------

func confirmDeleteModel(t *testing.T) Model {
	m := viewTextModel(t)
	m2, _ := update(m, key("delete"))
	require.Equal(t, screenConfirmDelete, m2.screen)
	return m2
}

func TestConfirmDelete_LeftSelectsYes(t *testing.T) {
	m := confirmDeleteModel(t)
	m.confirmCursor = 1
	m2, _ := update(m, key("left"))
	assert.Equal(t, 0, m2.confirmCursor)
}

func TestConfirmDelete_RightSelectsNo(t *testing.T) {
	m := confirmDeleteModel(t)
	m.confirmCursor = 0
	m2, _ := update(m, key("right"))
	assert.Equal(t, 1, m2.confirmCursor)
}

func TestConfirmDelete_EnterNoGoesBack(t *testing.T) {
	m := confirmDeleteModel(t)
	m.confirmCursor = 1
	m2, _ := update(m, key("enter"))
	assert.Equal(t, m.prevScreen, m2.screen)
}

func TestConfirmDelete_EnterYesReturnsCmd(t *testing.T) {
	m := confirmDeleteModel(t)
	m.confirmCursor = 0 // Yes
	_, cmd := update(m, key("enter"))
	assert.NotNil(t, cmd)
}

func TestConfirmDelete_EscGoesBack(t *testing.T) {
	m := confirmDeleteModel(t)
	prev := m.prevScreen
	m2, _ := update(m, key("esc"))
	assert.Equal(t, prev, m2.screen)
}

// ---------- store/update done message handlers ------------------------------

func TestHandleStoreTextDone_Error(t *testing.T) {
	m := addTextModel(t)
	m.formFields = []string{"note", "", "hi"}
	m2, _ := update(m, storeTextDoneMsg{err: errors.New("server error")})
	assert.Contains(t, m2.formErr, "server error")
	assert.Equal(t, screenAddText, m2.screen) // stays on form
}

func TestHandleStoreTextDone_Success(t *testing.T) {
	m := addTextModel(t)
	m.formFields = []string{"note", "", "hi"}
	m2, _ := update(m, storeTextDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "note")
}

func TestHandleStoreCredDone_Error(t *testing.T) {
	m := addCredModel(t)
	m.formFields = []string{"gh", "", "u", "p"}
	m2, _ := update(m, storeCredDoneMsg{err: errors.New("conflict")})
	assert.Contains(t, m2.formErr, "conflict")
}

func TestHandleStoreCredDone_Success(t *testing.T) {
	m := addCredModel(t)
	m.formFields = []string{"gh", "", "u", "p"}
	m2, _ := update(m, storeCredDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "gh")
}

func TestHandleStoreCardDone_Error(t *testing.T) {
	m := addCardModel(t)
	m.formFields = []string{"visa", "", "1234", "Me", "", ""}
	m2, _ := update(m, storeCardDoneMsg{err: errors.New("oops")})
	assert.Contains(t, m2.formErr, "oops")
}

func TestHandleStoreCardDone_Success(t *testing.T) {
	m := addCardModel(t)
	m.formFields = []string{"visa", "", "1234", "Me", "", ""}
	m2, _ := update(m, storeCardDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "visa")
}

func TestHandleStoreFileDone_Error(t *testing.T) {
	m := addFileModel(t)
	m.formFields = []string{"doc", "", "/tmp/f"}
	m2, _ := update(m, storeFileDoneMsg{err: errors.New("read error")})
	assert.Contains(t, m2.formErr, "read error")
}

func TestHandleStoreFileDone_Success(t *testing.T) {
	m := addFileModel(t)
	m.formFields = []string{"doc", "", "/tmp/f"}
	m2, _ := update(m, storeFileDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "doc")
}

func TestHandleUpdateTextDone_Error(t *testing.T) {
	m := modifyTextModel(t)
	m2, _ := update(m, updateTextDoneMsg{err: errors.New("network error")})
	assert.Contains(t, m2.formErr, "network error")
}

func TestHandleUpdateTextDone_Success(t *testing.T) {
	m := modifyTextModel(t)
	m.formFields = []string{"note", "", "new text"}
	m2, _ := update(m, updateTextDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "note")
	assert.Nil(t, m2.viewText)
}

func TestHandleUpdateCredDone_Success(t *testing.T) {
	m := modifyCredModel(t)
	m.formFields = []string{"gh", "", "u2", "p2"}
	m2, _ := update(m, updateCredDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "gh")
	assert.Nil(t, m2.viewCred)
}

func TestHandleUpdateCardDone_Success(t *testing.T) {
	m := modifyCardModel(t)
	m.formFields = []string{"visa", "", "9999", "Me", "", ""}
	m2, _ := update(m, updateCardDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "visa")
	assert.Nil(t, m2.viewCard)
}

func TestHandleDeleteSecretDone_Error(t *testing.T) {
	m := confirmDeleteModel(t)
	m.viewSecretName = "note"
	m2, _ := update(m, deleteSecretDoneMsg{err: errors.New("forbidden")})
	assert.Contains(t, m2.err, "forbidden")
	assert.Equal(t, screenMain, m2.screen)
}

func TestHandleDeleteSecretDone_Success(t *testing.T) {
	m := confirmDeleteModel(t)
	m.viewSecretName = "note"
	m2, _ := update(m, deleteSecretDoneMsg{err: nil})
	assert.Equal(t, screenMain, m2.screen)
	assert.Contains(t, m2.success, "note")
	assert.Nil(t, m2.viewText)
	assert.Nil(t, m2.viewCred)
	assert.Nil(t, m2.viewCard)
	assert.Nil(t, m2.viewFile)
}

// ---------- View smoke tests ------------------------------------------------

func TestView_EachScreenRendersNonEmpty(t *testing.T) {
	screens := []struct {
		name  string
		model func() Model
	}{
		{"authChoice", func() Model { return newModel(t) }},
		{"authForm", func() Model { return authFormModel(t, false) }},
		{"sync", func() Model {
			m := authFormModel(t, false)
			m.formFields = []string{"u", "p"}
			m2, _ := update(m, authDoneMsg{err: nil})
			return m2
		}},
		{"main", func() Model { return mainModel(t) }},
		{"addSecretList", func() Model { return addSecretModel(t) }},
		{"list", func() Model { return listModel(t) }},
		{"addText", func() Model { return addTextModel(t) }},
		{"addCred", func() Model { return addCredModel(t) }},
		{"addCard", func() Model { return addCardModel(t) }},
		{"addFile", func() Model { return addFileModel(t) }},
		{"viewText", func() Model { return viewTextModel(t) }},
		{"viewCred", func() Model { return viewCredModel(t) }},
		{"viewCard", func() Model { return viewCardModel(t) }},
		{"viewFile", func() Model { return viewFileModel(t) }},
		{"modifyText", func() Model { return modifyTextModel(t) }},
		{"modifyCred", func() Model { return modifyCredModel(t) }},
		{"modifyCard", func() Model { return modifyCardModel(t) }},
		{"confirmModify", func() Model { return confirmModifyModel(t) }},
		{"confirmDelete", func() Model { return confirmDeleteModel(t) }},
	}

	for _, tc := range screens {
		t.Run(tc.name, func(t *testing.T) {
			view := tc.model().View()
			assert.NotEmpty(t, view)
		})
	}
}

func TestView_AuthChoiceContainsLoginRegister(t *testing.T) {
	view := newModel(t).View()
	assert.Contains(t, view, "Login")
	assert.Contains(t, view, "Register")
}

func TestView_MainContainsMenuItems(t *testing.T) {
	view := mainModel(t).View()
	for _, item := range mainMenuItems {
		assert.Contains(t, view, item)
	}
}

func TestView_AuthFormShowsTitle(t *testing.T) {
	view := authFormModel(t, false).View()
	assert.Contains(t, view, "Login")
}

func TestView_AuthFormRegisterTitle(t *testing.T) {
	view := authFormModel(t, true).View()
	assert.Contains(t, view, "Register")
}

func TestView_AuthFormShowsError(t *testing.T) {
	m := authFormModel(t, false)
	m.formErr = "invalid credentials"
	assert.Contains(t, m.View(), "invalid credentials")
}

func TestView_MainShowsSuccess(t *testing.T) {
	m := mainModel(t)
	m.success = "done!"
	assert.Contains(t, m.View(), "done!")
}

func TestView_MainShowsError(t *testing.T) {
	m := mainModel(t)
	m.err = "something wrong"
	assert.Contains(t, m.View(), "something wrong")
}

func TestView_ListShowsEmptyMessage(t *testing.T) {
	view := listModel(t).View()
	assert.Contains(t, view, "No secrets")
}

func TestView_ListShowsEntries(t *testing.T) {
	m := listModel(t, models.SecretEntry{Name: "mysecret", Type: models.SecretTypeText})
	assert.Contains(t, m.View(), "mysecret")
}

func TestView_ViewTextShowsSecretName(t *testing.T) {
	view := viewTextModel(t).View()
	assert.Contains(t, view, "note")
}

func TestView_ViewCredShowsName(t *testing.T) {
	view := viewCredModel(t).View()
	assert.Contains(t, view, "gh")
}

func TestView_ViewCardShowsName(t *testing.T) {
	view := viewCardModel(t).View()
	assert.Contains(t, view, "visa")
}

func TestView_ViewFileShowsName(t *testing.T) {
	view := viewFileModel(t).View()
	assert.Contains(t, view, "report")
}

func TestView_ConfirmDeleteShowsSecretName(t *testing.T) {
	m := confirmDeleteModel(t)
	m.viewSecretName = "my-secret"
	assert.Contains(t, m.View(), "my-secret")
}
