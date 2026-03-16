package handlers

// registerRequest represents the user registration request payload.
type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// loginRequest represents the user login request payload.
type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// text represents a text secret in request and response payloads.
type text struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Text        string `json:"text"`
}

// credential represents login credentials in request and response payloads.
type credential struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Login       string `json:"login"`
	Password    string `json:"password"`
}

// card represents bank card information in request and response payloads.
type card struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CardNumber  string `json:"card_number"`
	Owner       string `json:"owner"`
	ExpiresAt   string `json:"expires_at"`
	CVC         string `json:"cvc"`
}

// file represents a base64-encoded file in request and response payloads.
type file struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UserFile    string `json:"user_file"`
}

// deleteRequest represents a secret deletion request.
type deleteRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// allSecrets represents metadata for listing secrets without exposing sensitive data.
type allSecrets struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
