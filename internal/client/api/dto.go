package api

// authRequest is the JSON body for register and login.
type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// listResponse represents one entry in a list returned by the server.
type listResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// deleteRequest specifies which secret to delete by name and type.
type deleteRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// fileReqResp represents file data in JSON format with base64-encoded content.
type fileReqResp struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UserFile    string `json:"user_file"`
}
