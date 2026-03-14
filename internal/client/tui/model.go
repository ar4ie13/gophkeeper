package tui

import (
	"fmt"
	"strings"

	"github.com/ar4ie13/gophkeeper/internal/client/config"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ar4ie13/gophkeeper/internal/client/models"
	"github.com/ar4ie13/gophkeeper/internal/client/service"
)

// screen represents the current view in the TUI.
type screen int

const (
	screenAuthChoice screen = iota // login or register?
	screenAuthForm                 // login/register form
	screenSync                     // initial sync loading
	screenMain                     // main menu
	screenList                     // list all secrets
	screenViewText                 // view a single text secret
	screenViewCred
	screenViewCard
	screenViewFile
	screenAddText
	screenAddCred
	screenAddCard
	screenAddFile
	screenModifyText
	screenModifyCred
	screenModifyCard
	screenAddSecretList
	screenConfirmModify
	screenConfirmDelete
)

// maxTextLen defines the maximum allowed length for text secrets.
const maxTextLen = 1000

// Model is the root Bubble Tea models.
type Model struct {
	svc     *service.Service
	conf    *config.Config
	screen  screen
	online  bool
	cursor  int
	err     string
	success string

	// auth state
	authIsRegister bool   // true = register, false = login
	authUser       string // remembered after successful auth

	// sync screen
	syncing bool

	// list screen
	entries    []models.SecretEntry
	listCursor int

	// view screens
	viewText       *models.TextSecret
	viewCred       *models.CredentialSecret
	viewCard       *models.CardSecret
	viewFile       *models.FileSecret
	viewCursor     int               // for navigating fields in view screens
	viewSecretType models.SecretType // track current viewing secret type
	viewSecretName string            // track current viewing secret name

	// form state (shared between auth form, add-text, add-cred etc.)
	formFields []string
	formLabels []string
	formFocus  int
	formErr    string

	// confirmation dialog
	confirmCursor int    // 0 = yes, 1 = no
	confirmAction string // "modify" or "delete"
	prevScreen    screen // screen to return to if user cancels

	width  int
	height int
}

// New creates the initial TUI models.
func New(svc *service.Service, cfg *config.Config) Model {
	return Model{
		svc:    svc,
		conf:   cfg,
		screen: screenAuthChoice,
	}
}

// Init returns nil — we wait for the user to login/register first.
func (m Model) Init() tea.Cmd {
	return nil
}

// --- Commands (side-effects) ------------------------------------------------

// authCmd performs authentication (login or register) in a goroutine.
func (m Model) authCmd() tea.Cmd {
	login := m.formFields[0]
	password := m.formFields[1]
	isRegister := m.authIsRegister
	svc := m.svc
	return func() tea.Msg {
		var err error
		if isRegister {
			err = svc.Register(login, password)
		} else {
			err = svc.Login(login, password)
		}
		return authDoneMsg{err: err}
	}
}

// syncCmd synchronizes all secrets from the server in a goroutine.
func (m Model) syncCmd() tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		result := svc.Sync()
		return syncDoneMsg{result: result}
	}
}

// storeTextCmd stores a text secret to the server in a goroutine.
func (m Model) storeTextCmd() tea.Cmd {
	secret := models.TextSecret{
		Name:        m.formFields[0],
		Description: m.formFields[1],
		Text:        m.formFields[2],
	}
	svc := m.svc
	return func() tea.Msg {
		err := svc.StoreText(secret)
		return storeTextDoneMsg{err: err}
	}
}

// modifyTextCmd updates an existing text secret on the server in a goroutine.
func (m Model) modifyTextCmd() tea.Cmd {
	secret := models.TextSecret{
		Name:        m.formFields[0],
		Description: m.formFields[1],
		Text:        m.formFields[2],
	}
	svc := m.svc
	return func() tea.Msg {
		err := svc.UpdateText(secret)
		return storeTextDoneMsg{err: err}
	}
}

// storeCredCmd stores credentials to the server in a goroutine.
func (m Model) storeCredCmd() tea.Cmd {
	secret := models.CredentialSecret{
		Name:        m.formFields[0],
		Description: m.formFields[1],
		Login:       m.formFields[2],
		Password:    m.formFields[3],
	}
	svc := m.svc
	return func() tea.Msg {
		err := svc.StoreCredential(secret)
		return storeCredDoneMsg{err: err}
	}
}

// modifyCredCmd updates existing credentials on the server in a goroutine.
func (m Model) modifyCredCmd() tea.Cmd {
	secret := models.CredentialSecret{
		Name:        m.formFields[0],
		Description: m.formFields[1],
		Login:       m.formFields[2],
		Password:    m.formFields[3],
	}
	svc := m.svc
	return func() tea.Msg {
		err := svc.UpdateCredential(secret)
		return storeCredDoneMsg{err: err}
	}
}

// storeCardCmd stores card information to the server in a goroutine.
func (m Model) storeCardCmd() tea.Cmd {
	secret := models.CardSecret{
		Name:        m.formFields[0],
		Description: m.formFields[1],
		CardNumber:  m.formFields[2],
		Owner:       m.formFields[3],
		ExpiresAt:   m.formFields[4],
		CVC:         m.formFields[5],
	}
	svc := m.svc
	return func() tea.Msg {
		err := svc.StoreCard(secret)
		return storeCardDoneMsg{err: err}
	}
}

// modifyCardCmd updates existing card information on the server in a goroutine.
func (m Model) modifyCardCmd() tea.Cmd {
	secret := models.CardSecret{
		Name:        m.formFields[0],
		Description: m.formFields[1],
		CardNumber:  m.formFields[2],
		Owner:       m.formFields[3],
		ExpiresAt:   m.formFields[4],
		CVC:         m.formFields[5],
	}
	svc := m.svc
	return func() tea.Msg {
		err := svc.UpdateCard(secret)
		return storeCardDoneMsg{err: err}
	}
}

// storeFileCmd stores a file to the server in a goroutine.
func (m Model) storeFileCmd() tea.Cmd {
	secret := models.FileSecret{
		Name:        m.formFields[0],
		Description: m.formFields[1],
		Path:        m.formFields[2],
	}
	svc := m.svc
	return func() tea.Msg {
		err := svc.StoreFile(secret)
		return storeFileDoneMsg{err: err}
	}
}

// deleteSecretCmd deletes a secret from the server in a goroutine.
func (m Model) deleteSecretCmd() tea.Cmd {
	name := m.viewSecretName
	secretType := m.viewSecretType.String()
	svc := m.svc
	return func() tea.Msg {
		err := svc.DeleteSecret(name, secretType)
		return deleteSecretDoneMsg{err: err}
	}
}

// viewSecretCmd fetches a secret from the server for viewing in a goroutine.
func (m Model) viewSecretCmd(entry models.SecretEntry) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		switch entry.Type {
		case models.SecretTypeText:
			s, err := svc.GetText(entry.Name)
			return viewTextMsg{secret: s, err: err}
		case models.SecretTypeCredential:
			s, err := svc.GetCredential(entry.Name)
			return viewCredMsg{secret: s, err: err}
		case models.SecretTypeCard:
			s, err := svc.GetCard(entry.Name)
			return viewCardMsg{secret: s, err: err}
		case models.SecretTypeFile:
			s, err := svc.GetFile(entry.Name)
			return viewFileMsg{secret: s, err: err}
		}
		return nil
	}
}

// --- Update -----------------------------------------------------------------

// Update handles all incoming messages and updates the model state accordingly.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)

	case authDoneMsg:
		m2, cmd := m.handleAuthDone(msg)
		return m2, cmd

	case syncDoneMsg:
		return m.handleSyncDone(msg), nil

	case storeTextDoneMsg:
		return m.handleStoreTextDone(msg), nil

	case storeCredDoneMsg:
		return m.handleStoreCredDone(msg), nil

	case storeCardDoneMsg:
		return m.handleStoreCardDone(msg), nil

	case storeFileDoneMsg:
		return m.handleStoreFileDone(msg), nil

	case updateTextDoneMsg:
		return m.handleUpdateTextDone(msg), nil

	case updateCredDoneMsg:
		return m.handleUpdateCredDone(msg), nil

	case updateCardDoneMsg:
		return m.handleUpdateCardDone(msg), nil

	case deleteSecretDoneMsg:
		return m.handleDeleteSecretDone(msg), nil

	case viewTextMsg:
		return m.handleViewText(msg), nil

	case viewCredMsg:
		return m.handleViewCred(msg), nil

	case viewCardMsg:
		return m.handleViewCard(msg), nil

	case viewFileMsg:
		return m.handleViewFile(msg), nil
	}

	return m, nil
}

// handleKey routes keyboard input to the appropriate screen handler.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.screen {
	case screenAuthChoice:
		return m.handleAuthChoiceKey(key)

	case screenAuthForm:
		return m.handleAuthFormKey(key, msg)

	case screenSync:
		return m, nil

	case screenMain:
		return m.handleMainKey(key)

	case screenList:
		return m.handleListKey(key)

	case screenViewText:
		return m.handleViewTextKey(key, msg)

	case screenViewCred:
		return m.handleViewCredKey(key, msg)

	case screenViewCard:
		return m.handleViewCardKey(key, msg)

	case screenViewFile:
		return m.handleViewFileKey(key)

	case screenAddSecretList:
		return m.handleAddSecretKey(key)

	case screenAddText:
		return m.handleAddTextKey(key, msg)

	case screenAddCred:
		return m.handleAddCredKey(key, msg)

	case screenAddCard:
		return m.handleAddCardKey(key, msg)

	case screenAddFile:
		return m.handleAddFileKey(key, msg)

	case screenModifyText:
		return m.handleModifyTextKey(key, msg)

	case screenModifyCred:
		return m.handleModifyCredKey(key, msg)

	case screenModifyCard:
		return m.handleModifyCardKey(key, msg)

	case screenConfirmModify:
		return m.handleConfirmModifyKey(key)

	case screenConfirmDelete:
		return m.handleConfirmDeleteKey(key)
	}

	return m, nil
}

// --- Auth choice screen -----------------------------------------------------

// authChoiceItems defines the menu options for initial authentication choice.
var authChoiceItems = []string{
	"Login",
	"Register",
}

// handleAuthChoiceKey handles keyboard input on the authentication choice screen.
func (m Model) handleAuthChoiceKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(authChoiceItems)-1 {
			m.cursor++
		}
	case "enter":
		m.authIsRegister = m.cursor == 1
		m.formLabels = []string{"Login", "Password"}
		m.formFields = []string{"", ""}
		m.formFocus = 0
		m.formErr = ""
		m.screen = screenAuthForm
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// --- Auth form screen -------------------------------------------------------

// handleAuthFormKey handles keyboard input on the authentication form screen.
func (m Model) handleAuthFormKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = screenAuthChoice
		m.cursor = 0
		m.formErr = ""
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			return m, nil
		}
		// Validate
		if strings.TrimSpace(m.formFields[0]) == "" {
			m.formErr = "Login is required"
			return m, nil
		}
		if strings.TrimSpace(m.formFields[1]) == "" {
			m.formErr = "Password is required"
			return m, nil
		}
		m.formErr = ""
		return m, m.authCmd()
	case "backspace":
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- Main menu --------------------------------------------------------------

// mainMenuItems defines the options available in the main menu.
var mainMenuItems = []string{
	"View all secrets",
	"Add secret",
	"Re-sync with server",
	"Logout",
}

// handleMainKey handles keyboard input on the main menu screen.
func (m Model) handleMainKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(mainMenuItems)-1 {
			m.cursor++
		}
	case "enter":
		m.err = ""
		m.success = ""
		switch m.cursor {
		case 0: // list
			m.entries = m.svc.ListSecrets()
			m.listCursor = 0
			m.screen = screenList
		case 1: // add secret
			m.formLabels = addSecretMenuItems
			m.cursor = 0
			m.screen = screenAddSecretList
		case 2: // re-sync
			m.syncing = true
			m.screen = screenSync
			return m, m.syncCmd()
		case 3: // logout → back to auth
			m.screen = screenAuthChoice
			m.cursor = 0
			m.online = false
			m.authUser = ""
			m.err = ""
			m.success = ""
		}
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// --- Add secret menu --------------------------------------------------------------

// addSecretMenuItems defines the types of secrets that can be added.
var addSecretMenuItems = []string{
	"Add text secret",
	"Add credential secret",
	"Add card secret",
	"Add file secret",
}

// handleAddSecretKey handles keyboard input on the add secret menu screen.
func (m Model) handleAddSecretKey(key string) (tea.Model, tea.Cmd) {

	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(addSecretMenuItems)-1 {
			m.cursor++
		}
	case "enter":
		m.err = ""
		m.success = ""
		switch m.cursor {
		case 0: // add text
			m.formLabels = []string{"Name", "Description", "Text (max 1000 chars)"}
			m.formFields = []string{"", "", ""}
			m.formFocus = 0
			m.formErr = ""
			m.screen = screenAddText
		case 1: // add credential
			m.formLabels = []string{"Name", "Description", "Login", "Password"}
			m.formFields = []string{"", "", "", ""}
			m.formFocus = 0
			m.formErr = ""
			m.screen = screenAddCred
		case 2: // add card
			m.formLabels = []string{"Name", "Description", "CardNumber", "Owner", "ExpiresAt", "CVC"}
			m.formFields = []string{"", "", "", "", "", ""}
			m.formFocus = 0
			m.formErr = ""
			m.screen = screenAddCard
		case 3: // add file
			m.formLabels = []string{"Name", "Description", "PathToFile (max 5MB)"}
			m.formFields = []string{"", "", ""}
			m.formFocus = 0
			m.formErr = ""
			m.screen = screenAddFile
		}

	case "esc", "backspace", "q":
		m.screen = screenMain
	}

	return m, nil
}

// --- List screen ------------------------------------------------------------

func (m Model) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.entries)-1 {
			m.listCursor++
		}
	case "enter":
		if len(m.entries) > 0 {
			return m, m.viewSecretCmd(m.entries[m.listCursor])
		}
	case "delete":
		if len(m.entries) > 0 {
			// Set up for delete confirmation
			entry := m.entries[m.listCursor]
			m.viewSecretName = entry.Name
			m.viewSecretType = entry.Type
			m.confirmCursor = 1 // default to "No"
			m.confirmAction = "delete"
			m.prevScreen = screenList
			m.screen = screenConfirmDelete
		}
	case "esc", "backspace", "q":
		m.screen = screenMain
	}
	return m, nil
}

// --- Add text form ----------------------------------------------------------

func (m Model) handleAddTextKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = screenAddSecretList
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			return m, nil
		}
		if strings.TrimSpace(m.formFields[0]) == "" {
			m.formErr = "Name is required"
			return m, nil
		}
		if len(m.formFields[2]) > maxTextLen {
			m.formErr = fmt.Sprintf("Text too long: %d/%d", len(m.formFields[2]), maxTextLen)
			return m, nil
		}
		m.formErr = ""
		return m, m.storeTextCmd()
	case "backspace":
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			if m.formFocus == 2 && len(m.formFields[2]) >= maxTextLen {
				m.formErr = "Maximum 1000 characters reached"
				return m, nil
			}
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- Add credential form ----------------------------------------------------

func (m Model) handleAddCredKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = screenAddSecretList
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			return m, nil
		}
		if strings.TrimSpace(m.formFields[0]) == "" {
			m.formErr = "Name is required"
			return m, nil
		}
		if strings.TrimSpace(m.formFields[2]) == "" || strings.TrimSpace(m.formFields[3]) == "" {
			m.formErr = "Login and password are required"
			return m, nil
		}
		m.formErr = ""
		return m, m.storeCredCmd()
	case "backspace":
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- Add card form ----------------------------------------------------

func (m Model) handleAddCardKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = screenAddSecretList
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			return m, nil
		}
		if strings.TrimSpace(m.formFields[0]) == "" {
			m.formErr = "Name is required"
			return m, nil
		}
		if strings.TrimSpace(m.formFields[2]) == "" || strings.TrimSpace(m.formFields[3]) == "" {
			m.formErr = "CardNumber and Owner are required"
			return m, nil
		}
		m.formErr = ""
		return m, m.storeCardCmd()
	case "backspace":
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- Add file form ----------------------------------------------------------

func (m Model) handleAddFileKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = screenAddSecretList
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			return m, nil
		}
		if strings.TrimSpace(m.formFields[0]) == "" {
			m.formErr = "Name is required"
			return m, nil
		}
		m.formErr = ""
		return m, m.storeFileCmd()
	case "backspace":
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- View text secret key handler -------------------------------------------

func (m Model) handleViewTextKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.viewText == nil {
		return m, nil
	}

	// Fields: 0=Name (read-only), 1=Description, 2=Text
	maxCursor := 2

	switch key {
	case "esc", "q", "backspace":
		m.screen = screenList
		m.viewText = nil
		m.viewCursor = 0
		return m, nil
	case "up", "k":
		if m.viewCursor > 0 {
			m.viewCursor--
		}
		return m, nil
	case "down", "j":
		if m.viewCursor < maxCursor {
			m.viewCursor++
		}
		return m, nil
	case "enter":
		// Can't edit name (field 0)
		if m.viewCursor == 0 {
			return m, nil
		}
		// Enter modify mode
		m.formLabels = []string{"Name", "Description", "Text (max 1000 chars)"}
		m.formFields = []string{
			m.viewText.Name,
			m.viewText.Description,
			m.viewText.Text,
		}
		m.formFocus = m.viewCursor
		m.formErr = ""
		m.prevScreen = screenViewText
		m.screen = screenModifyText
		return m, nil
	case "delete":
		// Prepare for delete confirmation
		m.confirmCursor = 1 // default to "No"
		m.confirmAction = "delete"
		m.prevScreen = screenViewText
		m.screen = screenConfirmDelete
		return m, nil
	}
	return m, nil
}

// --- View credential secret key handler -------------------------------------

func (m Model) handleViewCredKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.viewCred == nil {
		return m, nil
	}

	// Fields: 0=Name (read-only), 1=Description, 2=Login, 3=Password
	maxCursor := 3

	switch key {
	case "esc", "q", "backspace":
		m.screen = screenList
		m.viewCred = nil
		m.viewCursor = 0
		return m, nil
	case "up", "k":
		if m.viewCursor > 0 {
			m.viewCursor--
		}
		return m, nil
	case "down", "j":
		if m.viewCursor < maxCursor {
			m.viewCursor++
		}
		return m, nil
	case "enter":
		// Can't edit name (field 0)
		if m.viewCursor == 0 {
			return m, nil
		}
		// Enter modify mode
		m.formLabels = []string{"Name", "Description", "Login", "Password"}
		m.formFields = []string{
			m.viewCred.Name,
			m.viewCred.Description,
			m.viewCred.Login,
			m.viewCred.Password,
		}
		m.formFocus = m.viewCursor
		m.formErr = ""
		m.prevScreen = screenViewCred
		m.screen = screenModifyCred
		return m, nil
	case "delete":
		// Prepare for delete confirmation
		m.confirmCursor = 1 // default to "No"
		m.confirmAction = "delete"
		m.prevScreen = screenViewCred
		m.screen = screenConfirmDelete
		return m, nil
	}
	return m, nil
}

// --- View card secret key handler -------------------------------------------

func (m Model) handleViewCardKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.viewCard == nil {
		return m, nil
	}

	// Fields: 0=Name (read-only), 1=Description, 2=CardNumber, 3=Owner, 4=ExpiresAt, 5=CVC
	maxCursor := 5

	switch key {
	case "esc", "q", "backspace":
		m.screen = screenList
		m.viewCard = nil
		m.viewCursor = 0
		return m, nil
	case "up", "k":
		if m.viewCursor > 0 {
			m.viewCursor--
		}
		return m, nil
	case "down", "j":
		if m.viewCursor < maxCursor {
			m.viewCursor++
		}
		return m, nil
	case "enter":
		// Can't edit name (field 0)
		if m.viewCursor == 0 {
			return m, nil
		}
		// Enter modify mode
		m.formLabels = []string{"Name", "Description", "CardNumber", "Owner", "ExpiresAt", "CVC"}
		m.formFields = []string{
			m.viewCard.Name,
			m.viewCard.Description,
			m.viewCard.CardNumber,
			m.viewCard.Owner,
			m.viewCard.ExpiresAt,
			m.viewCard.CVC,
		}
		m.formFocus = m.viewCursor
		m.formErr = ""
		m.prevScreen = screenViewCard
		m.screen = screenModifyCard
		return m, nil
	case "delete":
		// Prepare for delete confirmation
		m.confirmCursor = 1 // default to "No"
		m.confirmAction = "delete"
		m.prevScreen = screenViewCard
		m.screen = screenConfirmDelete
		return m, nil
	}
	return m, nil
}

// --- View file secret key handler -------------------------------------------

func (m Model) handleViewFileKey(key string) (tea.Model, tea.Cmd) {
	// File secrets are read-only (no modification)
	switch key {
	case "esc", "q", "backspace":
		m.screen = screenList
		m.viewFile = nil
		return m, nil
	case "delete":
		// Prepare for delete confirmation
		m.confirmCursor = 1 // default to "No"
		m.confirmAction = "delete"
		m.prevScreen = screenViewFile
		m.screen = screenConfirmDelete
		return m, nil
	}
	return m, nil
}

// --- Modify text form -------------------------------------------------------

func (m Model) handleModifyTextKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = m.prevScreen
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		// Skip name field (field 0 is read-only)
		if m.formFocus == 0 {
			m.formFocus = 1
		}
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		// Skip name field (field 0 is read-only)
		if m.formFocus == 0 {
			m.formFocus = len(m.formFields) - 1
		}
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			// Skip name field
			if m.formFocus == 0 {
				m.formFocus = 1
			}
			return m, nil
		}
		// Validate
		if len(m.formFields[2]) > maxTextLen {
			m.formErr = fmt.Sprintf("Text too long: %d/%d", len(m.formFields[2]), maxTextLen)
			return m, nil
		}
		m.formErr = ""
		// Show confirmation
		m.confirmCursor = 0 // default to "Yes"
		m.confirmAction = "modify"
		m.prevScreen = screenModifyText
		m.screen = screenConfirmModify
		return m, nil
	case "backspace":
		// Name field is read-only
		if m.formFocus == 0 {
			return m, nil
		}
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		// Name field is read-only
		if m.formFocus == 0 {
			return m, nil
		}
		if len(msg.Runes) > 0 {
			if m.formFocus == 2 && len(m.formFields[2]) >= maxTextLen {
				m.formErr = "Maximum 1000 characters reached"
				return m, nil
			}
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- Modify credential form -------------------------------------------------

func (m Model) handleModifyCredKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = m.prevScreen
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		// Skip name field (field 0 is read-only)
		if m.formFocus == 0 {
			m.formFocus = 1
		}
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		// Skip name field (field 0 is read-only)
		if m.formFocus == 0 {
			m.formFocus = len(m.formFields) - 1
		}
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			// Skip name field
			if m.formFocus == 0 {
				m.formFocus = 1
			}
			return m, nil
		}
		// Validate
		if strings.TrimSpace(m.formFields[2]) == "" || strings.TrimSpace(m.formFields[3]) == "" {
			m.formErr = "Login and password are required"
			return m, nil
		}
		m.formErr = ""
		// Show confirmation
		m.confirmCursor = 0 // default to "Yes"
		m.confirmAction = "modify"
		m.prevScreen = screenModifyCred
		m.screen = screenConfirmModify
		return m, nil
	case "backspace":
		// Name field is read-only
		if m.formFocus == 0 {
			return m, nil
		}
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		// Name field is read-only
		if m.formFocus == 0 {
			return m, nil
		}
		if len(msg.Runes) > 0 {
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- Modify card form -------------------------------------------------------

func (m Model) handleModifyCardKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.screen = m.prevScreen
		return m, nil
	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % len(m.formFields)
		// Skip name field (field 0 is read-only)
		if m.formFocus == 0 {
			m.formFocus = 1
		}
		return m, nil
	case "shift+tab", "up":
		m.formFocus = (m.formFocus - 1 + len(m.formFields)) % len(m.formFields)
		// Skip name field (field 0 is read-only)
		if m.formFocus == 0 {
			m.formFocus = len(m.formFields) - 1
		}
		return m, nil
	case "enter":
		if m.formFocus < len(m.formFields)-1 {
			m.formFocus++
			// Skip name field
			if m.formFocus == 0 {
				m.formFocus = 1
			}
			return m, nil
		}
		// Validate
		if strings.TrimSpace(m.formFields[2]) == "" || strings.TrimSpace(m.formFields[3]) == "" {
			m.formErr = "CardNumber and Owner are required"
			return m, nil
		}
		m.formErr = ""
		// Show confirmation
		m.confirmCursor = 0 // default to "Yes"
		m.confirmAction = "modify"
		m.prevScreen = screenModifyCard
		m.screen = screenConfirmModify
		return m, nil
	case "backspace":
		// Name field is read-only
		if m.formFocus == 0 {
			return m, nil
		}
		if len(m.formFields[m.formFocus]) > 0 {
			m.formFields[m.formFocus] = m.formFields[m.formFocus][:len(m.formFields[m.formFocus])-1]
		}
		return m, nil
	default:
		// Name field is read-only
		if m.formFocus == 0 {
			return m, nil
		}
		if len(msg.Runes) > 0 {
			m.formFields[m.formFocus] += string(msg.Runes)
			m.formErr = ""
		}
		return m, nil
	}
}

// --- Confirmation dialog handlers -------------------------------------------

func (m Model) handleConfirmModifyKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "left", "h":
		m.confirmCursor = 0 // Yes
		return m, nil
	case "right", "l":
		m.confirmCursor = 1 // No
		return m, nil
	case "enter":
		if m.confirmCursor == 1 { // No
			m.screen = m.prevScreen
			return m, nil
		}
		// Yes - perform the modification
		switch m.prevScreen {
		case screenModifyText:
			return m, m.modifyTextCmd()
		case screenModifyCred:
			return m, m.modifyCredCmd()
		case screenModifyCard:
			return m, m.modifyCardCmd()
		}
		m.screen = m.prevScreen
		return m, nil
	case "esc", "q":
		m.screen = m.prevScreen
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmDeleteKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "left", "h":
		m.confirmCursor = 0 // Yes
		return m, nil
	case "right", "l":
		m.confirmCursor = 1 // No
		return m, nil
	case "enter":
		if m.confirmCursor == 1 { // No
			m.screen = m.prevScreen
			return m, nil
		}
		// Yes - perform the deletion
		return m, m.deleteSecretCmd()
	case "esc", "q":
		m.screen = m.prevScreen
		return m, nil
	}
	return m, nil
}

// --- Message handlers -------------------------------------------------------

func (m Model) handleAuthDone(msg authDoneMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m, nil
	}
	// Auth succeeded — remember username, kick off sync.
	m.authUser = m.formFields[0]
	m.syncing = true
	m.screen = screenSync
	return m, m.syncCmd()
}

func (m Model) handleSyncDone(msg syncDoneMsg) Model {
	m.syncing = false
	m.online = msg.result.Online
	m.screen = screenMain
	m.cursor = 0
	if msg.result.Error != nil && msg.result.Online {
		m.err = msg.result.Error.Error()
	}
	if msg.result.Online {
		m.success = fmt.Sprintf("Synced %d texts, %d credentials, %d cards, %d files",
			msg.result.TextCount, msg.result.CredentialCount, msg.result.CardCount, msg.result.FilesCount)
	}
	return m
}

func (m Model) handleStoreTextDone(msg storeTextDoneMsg) Model {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m
	}
	m.screen = screenMain
	m.success = fmt.Sprintf("Text secret %q stored successfully", m.formFields[0])
	return m
}

func (m Model) handleStoreCredDone(msg storeCredDoneMsg) Model {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m
	}
	m.screen = screenMain
	m.success = fmt.Sprintf("Credential %q stored successfully", m.formFields[0])
	return m
}

func (m Model) handleStoreCardDone(msg storeCardDoneMsg) Model {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m
	}
	m.screen = screenMain
	m.success = fmt.Sprintf("Card %q stored successfully", m.formFields[0])
	return m
}

func (m Model) handleStoreFileDone(msg storeFileDoneMsg) Model {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m
	}
	m.screen = screenMain
	m.success = fmt.Sprintf("File secret %q stored successfully", m.formFields[0])
	return m
}

func (m Model) handleUpdateTextDone(msg updateTextDoneMsg) Model {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m
	}
	// Refresh the list
	m.entries = m.svc.ListSecrets()
	m.screen = screenMain
	m.success = fmt.Sprintf("Text secret %q updated successfully", m.formFields[0])
	// Clear view state
	m.viewText = nil
	m.viewCursor = 0
	return m
}

func (m Model) handleUpdateCredDone(msg updateCredDoneMsg) Model {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m
	}
	// Refresh the list
	m.entries = m.svc.ListSecrets()
	m.screen = screenMain
	m.success = fmt.Sprintf("Credential secret %q updated successfully", m.formFields[0])
	// Clear view state
	m.viewCred = nil
	m.viewCursor = 0
	return m
}

func (m Model) handleUpdateCardDone(msg updateCardDoneMsg) Model {
	if msg.err != nil {
		m.formErr = msg.err.Error()
		return m
	}
	// Refresh the list
	m.entries = m.svc.ListSecrets()
	m.screen = screenMain
	m.success = fmt.Sprintf("Card secret %q updated successfully", m.formFields[0])
	// Clear view state
	m.viewCard = nil
	m.viewCursor = 0
	return m
}

func (m Model) handleDeleteSecretDone(msg deleteSecretDoneMsg) Model {
	if msg.err != nil {
		m.err = msg.err.Error()
		m.screen = screenMain
		return m
	}
	// Refresh the list
	m.entries = m.svc.ListSecrets()
	m.screen = screenMain
	m.success = fmt.Sprintf("Secret %q deleted successfully", m.viewSecretName)
	// Clear view state
	m.viewText = nil
	m.viewCred = nil
	m.viewCard = nil
	m.viewFile = nil
	m.viewCursor = 0
	return m
}

func (m Model) handleViewText(msg viewTextMsg) Model {
	if msg.err != nil {
		m.err = msg.err.Error()
		return m
	}
	m.viewText = msg.secret
	m.viewSecretName = msg.secret.Name
	m.viewSecretType = models.SecretTypeText
	m.viewCursor = 0
	m.screen = screenViewText
	return m
}

func (m Model) handleViewCred(msg viewCredMsg) Model {
	if msg.err != nil {
		m.err = msg.err.Error()
		return m
	}
	m.viewCred = msg.secret
	m.viewSecretName = msg.secret.Name
	m.viewSecretType = models.SecretTypeCredential
	m.viewCursor = 0
	m.screen = screenViewCred
	return m
}

func (m Model) handleViewCard(msg viewCardMsg) Model {
	if msg.err != nil {
		m.err = msg.err.Error()
		return m
	}
	m.viewCard = msg.secret
	m.viewSecretName = msg.secret.Name
	m.viewSecretType = models.SecretTypeCard
	m.viewCursor = 0
	m.screen = screenViewCard
	return m
}

func (m Model) handleViewFile(msg viewFileMsg) Model {
	if msg.err != nil {
		m.err = msg.err.Error()
		return m
	}
	m.viewFile = msg.secret
	m.viewSecretName = msg.secret.Name
	m.viewSecretType = models.SecretTypeFile
	m.screen = screenViewFile
	return m
}
