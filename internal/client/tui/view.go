package tui

import (
	"fmt"
	"strings"

	"github.com/ar4ie13/gophkeeper/internal/client/models"
)

// View renders the current screen.
func (m Model) View() string {
	var b strings.Builder

	switch m.screen {
	case screenAuthChoice:
		b.WriteString(m.renderAuthChoice())
	case screenAuthForm:
		b.WriteString(m.renderAuthForm())
	case screenSync:
		b.WriteString(m.renderSync())
	case screenMain:
		b.WriteString(m.renderMain())
	case screenAddSecretList:
		b.WriteString(m.renderAdd())
	case screenList:
		b.WriteString(m.renderList())
	case screenViewText:
		b.WriteString(m.renderViewText())
	case screenViewCred:
		b.WriteString(m.renderViewCred())
	case screenViewCard:
		b.WriteString(m.renderViewCard())
	case screenViewFile:
		b.WriteString(m.renderViewFile())
	case screenAddText:
		b.WriteString(m.renderAddText())
	case screenAddCred:
		b.WriteString(m.renderAddCred())
	case screenAddCard:
		b.WriteString(m.renderAddCard())
	case screenAddFile:
		b.WriteString(m.renderAddFile())
	case screenModifyText:
		b.WriteString(m.renderModifyText())
	case screenModifyCred:
		b.WriteString(m.renderModifyCred())
	case screenModifyCard:
		b.WriteString(m.renderModifyCard())
	case screenConfirmModify:
		b.WriteString(m.renderConfirmModify())
	case screenConfirmDelete:
		b.WriteString(m.renderConfirmDelete())
	}

	return b.String()
}

// --- Header -----------------------------------------------------------------

func (m Model) renderHeader() string {
	title := headerBoxStyle.Render("Secret Keeper")

	var status string
	if m.online {
		status = statusOnline.Render("● ONLINE")
	} else {
		status = statusOffline.Render("● OFFLINE (cached mode)")
	}

	user := ""
	if m.authUser != "" {
		user = "  " + labelStyle.Render("user: ") + secretNameStyle.Render(m.authUser)
	}

	return title + "\n" + status + user + "\n\n"
}

// --- Auth choice screen -----------------------------------------------------

func (m Model) renderAuthChoice() string {
	var b strings.Builder

	b.WriteString("\n" + titleStyle.Render("Secret Keeper") + "\n")
	b.WriteString(subtitleStyle.Render("Version: " + m.conf.Version))
	b.WriteString(subtitleStyle.Render("\n" + "Build Date: " + m.conf.BuildDate + "\n\n"))
	b.WriteString(labelStyle.Render("  Welcome! Please login or register:") + "\n\n")

	for i, item := range authChoiceItems {
		cursor := "  "
		style := menuItemStyle
		if i == m.cursor {
			cursor = menuCursorStyle.Render("▸ ")
			style = menuItemStyle.Bold(true).Foreground(colorText)
		}
		b.WriteString(cursor + style.Render(item) + "\n")
	}

	b.WriteString(helpStyle.Render("  ↑/↓ navigate • enter select • q quit"))
	return b.String()
}

// --- Auth form screen -------------------------------------------------------

func (m Model) renderAuthForm() string {
	var b strings.Builder

	title := "Login"
	if m.authIsRegister {
		title = "Register"
	}

	b.WriteString("\n" + titleStyle.Render("Secret Keeper — "+title) + "\n\n")
	b.WriteString(m.renderForm())

	if m.formErr != "" {
		b.WriteString(errorStyle.Render("  ✗ "+m.formErr) + "\n")
	}

	b.WriteString(helpStyle.Render("  tab next field • enter submit (on last field) • esc back"))
	return b.String()
}

// --- Sync screen ------------------------------------------------------------

func (m Model) renderSync() string {
	return "\n" + titleStyle.Render("Secret Keeper") + "\n\n" +
		subtitleStyle.Render("  Synchronizing with server...") + "\n"
}

// --- Main menu --------------------------------------------------------------

func (m Model) renderMain() string {
	var b strings.Builder

	b.WriteString(m.renderHeader())

	if m.success != "" {
		b.WriteString(successMsgStyle.Render("  ✓ "+m.success) + "\n\n")
	}
	if m.err != "" {
		b.WriteString(errorStyle.Render("  ✗ "+m.err) + "\n\n")
	}

	b.WriteString(labelStyle.Render("  Choose an action:") + "\n\n")

	for i, item := range mainMenuItems {
		cursor := "  "
		style := menuItemStyle
		if i == m.cursor {
			cursor = menuCursorStyle.Render("▸ ")
			style = menuItemStyle.Bold(true).Foreground(colorText)
		}
		b.WriteString(cursor + style.Render(item) + "\n")
	}

	b.WriteString(helpStyle.Render("  ↑/↓ navigate • enter select • q quit"))
	return b.String()
}

// --- Add menu --------------------------------------------------------------

func (m Model) renderAdd() string {
	var b strings.Builder

	b.WriteString(m.renderHeader())

	if m.success != "" {
		b.WriteString(successMsgStyle.Render("  ✓ "+m.success) + "\n\n")
	}
	if m.err != "" {
		b.WriteString(errorStyle.Render("  ✗ "+m.err) + "\n\n")
	}

	b.WriteString(labelStyle.Render("  Choose an action:") + "\n\n")

	for i, item := range addSecretMenuItems {
		cursor := "  "
		style := menuItemStyle
		if i == m.cursor {
			cursor = menuCursorStyle.Render("▸ ")
			style = menuItemStyle.Bold(true).Foreground(colorText)
		}
		b.WriteString(cursor + style.Render(item) + "\n")
	}

	b.WriteString(helpStyle.Render("  ↑/↓ navigate • enter select • esc back"))
	return b.String()
}

// --- List screen ------------------------------------------------------------

func (m Model) renderList() string {
	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  All Secrets") + "\n")

	if len(m.entries) == 0 {
		b.WriteString(labelStyle.Render("  No secrets stored yet.") + "\n")
	} else {
		for i, entry := range m.entries {
			cursor := "  "
			if i == m.listCursor {
				cursor = menuCursorStyle.Render("▸ ")
			}

			typeTag := renderTypeTag(entry.Type)
			name := secretNameStyle.Render(entry.Name)
			desc := ""
			if entry.Description != "" {
				desc = labelStyle.Render(" — " + entry.Description)
			}

			b.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, typeTag, name, desc))
		}
	}

	b.WriteString(helpStyle.Render("  ↑/↓ navigate • enter view • delete remove • esc back"))
	return b.String()
}

func renderTypeTag(t models.SecretType) string {
	switch t {
	case models.SecretTypeText:
		return labelStyle.Copy().
			Foreground(colorSecondary).
			Render("[TEXT]")
	case models.SecretTypeCredential:
		return labelStyle.Copy().
			Foreground(colorWarning).
			Render("[CRED]")
	case models.SecretTypeCard:
		return labelStyle.Copy().
			Foreground(colorThird).
			Render("[CARD]")
	case models.SecretTypeFile:
		return labelStyle.Copy().
			Foreground(colorDanger).
			Render("[FILE]")
	default:
		return labelStyle.Render("[???]")
	}
}

// --- View text secret -------------------------------------------------------

func (m Model) renderViewText() string {
	if m.viewText == nil {
		return m.renderHeader() + errorStyle.Render("  No data")
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Text Secret") + "\n")

	// Field 0: Name (read-only)
	nameLabel := labelStyle.Render("Name:")
	nameValue := secretNameStyle.Render(m.viewText.Name)
	if m.viewCursor == 0 {
		nameLabel = labelStyle.Bold(true).Render("▸ Name:")
	}

	// Field 1: Description
	descLabel := labelStyle.Render("Desc:")
	descValue := secretValueStyle.Render(m.viewText.Description)
	if m.viewCursor == 1 {
		descLabel = labelStyle.Bold(true).Render("▸ Desc:")
	}

	// Field 2: Text
	textLabel := labelStyle.Render("Content:")
	textValue := secretValueStyle.Render(m.viewText.Text)
	if m.viewCursor == 2 {
		textLabel = labelStyle.Bold(true).Render("▸ Content:")
	}

	content := fmt.Sprintf(
		"%s  %s\n%s  %s\n\n%s\n%s",
		nameLabel, nameValue,
		descLabel, descValue,
		textLabel, textValue,
	)
	b.WriteString(boxStyle.Render(content) + "\n")
	b.WriteString(helpStyle.Render("  ↑/↓ navigate • enter edit (except name) • del delete secret • esc back"))
	return b.String()
}

// --- View credential secret -------------------------------------------------

func (m Model) renderViewCred() string {
	if m.viewCred == nil {
		return m.renderHeader() + errorStyle.Render("  No data")
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Credential Secret") + "\n")

	// Field 0: Name (read-only)
	nameLabel := labelStyle.Render("Name:")
	if m.viewCursor == 0 {
		nameLabel = labelStyle.Bold(true).Render("▸ Name:")
	}

	// Field 1: Description
	descLabel := labelStyle.Render("Desc:")
	if m.viewCursor == 1 {
		descLabel = labelStyle.Bold(true).Render("▸ Desc:")
	}

	// Field 2: Login
	loginLabel := labelStyle.Render("Login:")
	if m.viewCursor == 2 {
		loginLabel = labelStyle.Bold(true).Render("▸ Login:")
	}

	// Field 3: Password
	passwordLabel := labelStyle.Render("Password:")
	if m.viewCursor == 3 {
		passwordLabel = labelStyle.Bold(true).Render("▸ Password:")
	}

	content := fmt.Sprintf(
		"%s      %s\n%s      %s\n%s     %s\n%s  %s",
		nameLabel,
		secretNameStyle.Render(m.viewCred.Name),
		descLabel,
		secretValueStyle.Render(m.viewCred.Description),
		loginLabel,
		secretValueStyle.Render(m.viewCred.Login),
		passwordLabel,
		secretValueStyle.Render(m.viewCred.Password),
	)
	b.WriteString(boxStyle.Render(content) + "\n")
	b.WriteString(helpStyle.Render("  ↑/↓ navigate • enter edit (except name) • delete remove • esc back"))
	return b.String()
}

// --- View card secret -------------------------------------------------

func (m Model) renderViewCard() string {
	if m.viewCard == nil {
		return m.renderHeader() + errorStyle.Render("  No data")
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Card Secret") + "\n")

	// Field 0: Name (read-only)
	nameLabel := labelStyle.Render("Name:")
	if m.viewCursor == 0 {
		nameLabel = labelStyle.Bold(true).Render("▸ Name:")
	}

	// Field 1: Description
	descLabel := labelStyle.Render("Desc:")
	if m.viewCursor == 1 {
		descLabel = labelStyle.Bold(true).Render("▸ Desc:")
	}

	// Field 2: CardNumber
	cardNumberLabel := labelStyle.Render("CardNumber:")
	if m.viewCursor == 2 {
		cardNumberLabel = labelStyle.Bold(true).Render("▸ CardNumber:")
	}

	// Field 3: Owner
	ownerLabel := labelStyle.Render("Owner:")
	if m.viewCursor == 3 {
		ownerLabel = labelStyle.Bold(true).Render("▸ Owner:")
	}

	// Field 4: ExpiresAt
	expiresAtLabel := labelStyle.Render("ExpiresAt:")
	if m.viewCursor == 4 {
		expiresAtLabel = labelStyle.Bold(true).Render("▸ ExpiresAt:")
	}

	// Field 5: CVC
	cvcLabel := labelStyle.Render("CVC:")
	if m.viewCursor == 5 {
		cvcLabel = labelStyle.Bold(true).Render("▸ CVC:")
	}

	content := fmt.Sprintf(
		"%s       %s\n%s       %s\n%s %s\n%s      %s\n%s  %s\n%s        %s",
		nameLabel,
		secretNameStyle.Render(m.viewCard.Name),
		descLabel,
		secretValueStyle.Render(m.viewCard.Description),
		cardNumberLabel,
		secretValueStyle.Render(m.viewCard.CardNumber),
		ownerLabel,
		secretValueStyle.Render(m.viewCard.Owner),
		expiresAtLabel,
		secretValueStyle.Render(m.viewCard.ExpiresAt),
		cvcLabel,
		secretValueStyle.Render(m.viewCard.CVC),
	)
	b.WriteString(boxStyle.Render(content) + "\n")
	b.WriteString(helpStyle.Render("  ↑/↓ navigate • enter edit (except name) • delete remove • esc back"))
	return b.String()
}

// --- View file secret -------------------------------------------------------

func (m Model) renderViewFile() string {
	if m.viewFile == nil {
		return m.renderHeader() + errorStyle.Render("  No data")
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  File Secret") + "\n")

	content := fmt.Sprintf(
		"%s  %s\n%s  %s\n\n%s\n%s",
		labelStyle.Render("Name:"),
		secretNameStyle.Render(m.viewFile.Name),
		labelStyle.Render("Desc:"),
		secretValueStyle.Render(m.viewFile.Description),
		labelStyle.Render("Path:"),
		secretValueStyle.Render(m.viewFile.Path),
	)
	b.WriteString(boxStyle.Render(content) + "\n")
	b.WriteString(helpStyle.Render("  delete remove • esc back"))
	return b.String()
}

// --- Add text form ----------------------------------------------------------

func (m Model) renderAddText() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  New Text Secret") + "\n")

	b.WriteString(m.renderForm())

	if m.formFocus == 2 {
		count := len(m.formFields[2])
		color := colorMuted
		if count > 900 {
			color = colorWarning
		}
		if count >= maxTextLen {
			color = colorDanger
		}
		counter := fmt.Sprintf("  %d/%d characters", count, maxTextLen)
		b.WriteString(labelStyle.Copy().Foreground(color).Render(counter) + "\n")
	}

	if m.formErr != "" {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.formErr))
	}

	b.WriteString(helpStyle.Render("  tab next field • enter submit (on last field) • esc cancel"))
	return b.String()
}

// --- Add credential form ----------------------------------------------------

func (m Model) renderAddCred() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  New Credential Secret") + "\n")

	b.WriteString(m.renderForm())

	if m.formErr != "" {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.formErr))
	}

	b.WriteString(helpStyle.Render("  tab next field • enter submit (on last field) • esc cancel"))
	return b.String()
}

// --- Add card form ----------------------------------------------------

func (m Model) renderAddCard() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  New Card Secret") + "\n")

	b.WriteString(m.renderForm())

	if m.formErr != "" {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.formErr))
	}

	b.WriteString(helpStyle.Render("  tab next field • enter submit (on last field) • esc cancel"))
	return b.String()
}

// --- Add file form ----------------------------------------------------

func (m Model) renderAddFile() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  New File Secret") + "\n")

	b.WriteString(m.renderForm())

	if m.formErr != "" {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.formErr))
	}

	b.WriteString(helpStyle.Render("  tab next field • enter submit (on last field) • esc cancel"))
	return b.String()
}

// --- Modify text form -------------------------------------------------------

func (m Model) renderModifyText() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Modify Text Secret") + "\n")

	b.WriteString(m.renderModifyForm())

	if m.formFocus == 2 {
		count := len(m.formFields[2])
		color := colorMuted
		if count > 900 {
			color = colorWarning
		}
		if count >= 1000 {
			color = colorDanger
		}
		counter := fmt.Sprintf("  %d/%d characters", count, 1000)
		b.WriteString(labelStyle.Foreground(color).Render(counter) + "\n")
	}

	if m.formErr != "" {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.formErr))
	}

	b.WriteString(helpStyle.Render("  tab next field • enter save • esc cancel"))
	return b.String()
}

// --- Modify credential form -------------------------------------------------

func (m Model) renderModifyCred() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Modify Credential Secret") + "\n")

	b.WriteString(m.renderModifyForm())

	if m.formErr != "" {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.formErr))
	}

	b.WriteString(helpStyle.Render("  tab next field • enter save • esc cancel"))
	return b.String()
}

// --- Modify card form -------------------------------------------------------

func (m Model) renderModifyCard() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Modify Card Secret") + "\n")

	b.WriteString(m.renderModifyForm())

	if m.formErr != "" {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.formErr))
	}

	b.WriteString(helpStyle.Render("  tab next field • enter save • esc cancel"))
	return b.String()
}

// --- Confirmation dialogs ---------------------------------------------------

func (m Model) renderConfirmModify() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Confirm Modification") + "\n\n")

	b.WriteString(labelStyle.Render("  Are you sure you want to save these changes?") + "\n\n")

	// Render Yes/No buttons
	yes := "  Yes  "
	no := "  No  "
	if m.confirmCursor == 0 {
		yes = menuCursorStyle.Render("▸") + " " + menuItemStyle.Bold(true).Render("Yes") + "  "
		no = "  " + menuItemStyle.Render("No") + "  "
	} else {
		yes = "  " + menuItemStyle.Render("Yes") + "  "
		no = menuCursorStyle.Render("▸") + " " + menuItemStyle.Bold(true).Render("No") + "  "
	}

	b.WriteString("  " + yes + no + "\n\n")
	b.WriteString(helpStyle.Render("  ←/→ or h/l select • enter confirm • esc cancel"))
	return b.String()
}

func (m Model) renderConfirmDelete() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString(titleStyle.Render("  Confirm Deletion") + "\n\n")

	b.WriteString(errorStyle.Render(fmt.Sprintf("  Are you sure you want to delete secret %q?", m.viewSecretName)) + "\n")
	b.WriteString(labelStyle.Render("  This action cannot be undone.") + "\n\n")

	// Render Yes/No buttons
	yes := "  Yes  "
	no := "  No  "
	if m.confirmCursor == 0 {
		yes = menuCursorStyle.Render("▸") + " " + menuItemStyle.Bold(true).Render("Yes") + "  "
		no = "  " + menuItemStyle.Render("No") + "  "
	} else {
		yes = "  " + menuItemStyle.Render("Yes") + "  "
		no = menuCursorStyle.Render("▸") + " " + menuItemStyle.Bold(true).Render("No") + "  "
	}

	b.WriteString("  " + yes + no + "\n\n")
	b.WriteString(helpStyle.Render("  ←/→ or h/l select • enter confirm • esc cancel"))
	return b.String()
}

// --- Shared form renderer ---------------------------------------------------

func (m Model) renderForm() string {
	var b strings.Builder

	for i, label := range m.formLabels {
		lbl := inputLabelStyle.Render("  " + label + ": ")

		val := m.formFields[i]
		// Only mask password on the auth (login/register) screen
		if label == "Password" && m.screen == screenAuthForm {
			val = strings.Repeat("•", len(val))
		}

		if i == m.formFocus {
			val = inputActiveStyle.Render(val + "█")
		} else {
			val = inputInactiveStyle.Render(val)
		}

		b.WriteString(lbl + val + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// --- Modify form renderer (name is read-only) ------------------------------

func (m Model) renderModifyForm() string {
	var b strings.Builder

	for i, label := range m.formLabels {
		lbl := inputLabelStyle.Render("  " + label + ": ")

		val := m.formFields[i]

		// Field 0 (Name) is always read-only
		if i == 0 {
			val = inputInactiveStyle.Render(val + " (read-only)")
		} else if i == m.formFocus {
			val = inputActiveStyle.Render(val + "█")
		} else {
			val = inputInactiveStyle.Render(val)
		}

		b.WriteString(lbl + val + "\n")
	}
	b.WriteString("\n")
	return b.String()
}
