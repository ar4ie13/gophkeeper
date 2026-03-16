package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/ar4ie13/gophkeeper/internal/apperrors"
	"github.com/ar4ie13/gophkeeper/internal/server/handlers/config"
	"github.com/ar4ie13/gophkeeper/internal/server/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// maxBase64Len defines the maximum allowed base64-encoded file size (~5 MB).
const maxBase64Len = 6_990_508 // ~5 MB in base64

// errorStatusMap maps application errors to HTTP status codes.
var errorStatusMap = map[error]int{
	apperrors.ErrUserAlreadyExists:  http.StatusConflict,
	apperrors.ErrPasswordMinSymbols: http.StatusBadRequest,
	apperrors.ErrNotFound:           http.StatusNoContent,
}

// Handlers aggregates HTTP handler dependencies and provides request handling methods.
type Handlers struct {
	cfg  config.ServerConfig
	zlog zerolog.Logger
	auth Auth
	srv  Service
}

// NewHandler creates a new Handlers instance with the provided dependencies.
func NewHandler(cfg config.ServerConfig, zlog zerolog.Logger, auth Auth, srv Service) *Handlers {
	return &Handlers{
		cfg:  cfg,
		zlog: zlog,
		auth: auth,
		srv:  srv,
	}
}

// Auth used for authentication
type Auth interface {
	BuildJWTString(userUUID uuid.UUID) (string, error)
	ValidateUserUUID(tokenString string) (uuid.UUID, error)
	GenerateHashFromPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) bool
}

// Service interface used in handlers layer
type Service interface {
	LoginUser(ctx context.Context, login string) (models.User, error)
	CreateUser(ctx context.Context, user models.User) (uuid.UUID, error)
	StoreSecret(ctx context.Context, secretType string, secret any) error
	DecryptSecret(ctx context.Context, secretType string, userUUID uuid.UUID, name string) (any, error)
	UpdateSecret(ctx context.Context, secretType string, secret any) error
	DeleteSecret(ctx context.Context, secretType string, userUUID uuid.UUID, name string) error
	GetAllUserTexts(ctx context.Context, user uuid.UUID) ([]models.Text, error)
	GetAllUserCredentials(ctx context.Context, user uuid.UUID) ([]models.Credential, error)
	GetAllUserCards(ctx context.Context, user uuid.UUID) ([]models.Card, error)
	GetAllUserFiles(ctx context.Context, user uuid.UUID) ([]models.File, error)
}

// registerRoutes configures and returns the Gin router with all API endpoints.
func (h *Handlers) registerRoutes() *gin.Engine {
	router := gin.New()

	//middlewares for router
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	//API routes
	auth := router.Group("/api/user")
	{
		auth.POST("/register", h.userRegister)
		auth.POST("/login", h.userLogin)
	}

	user := router.Group("/api/user").Use(h.authMiddleware())
	{
		user.GET("/ping", h.ping)
		user.POST("/text/store", h.storeText)
		user.GET("/text/get/:name", h.getText)
		user.PATCH("/text/patch", h.updateText)
		user.POST("/credential/store", h.storeCredential)
		user.GET("/credential/get/:name", h.getCredential)
		user.PATCH("/credential/patch", h.updateCredential)
		user.POST("/card/store", h.storeCard)
		user.GET("/card/get/:name", h.getCard)
		user.PATCH("/card/patch", h.updateCard)
		user.POST("/file/store", h.storeFile)
		user.GET("/file/get/:name", h.getFile)
		user.GET("/text/list", h.getAllUserTexts)
		user.GET("/credential/list", h.getAllUserCredentials)
		user.GET("/card/list", h.getAllUserCards)
		user.GET("/file/list", h.getAllUserFiles)
		user.DELETE("/secret/delete", h.deleteSecret)
	}

	return router
}

// getStatusCode process error and return the correlated status code
func (h *Handlers) getStatusCode(err error) int {
	// fast error check
	if status, exists := errorStatusMap[err]; exists {
		return status
	}

	// For wrapped errors
	for errType, status := range errorStatusMap {
		if errors.Is(err, errType) {
			return status
		}
	}
	return http.StatusInternalServerError
}

// userRegister is a handler used for user registration by using provided login and password
func (h *Handlers) userRegister(c *gin.Context) {
	var registerReq registerRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&registerReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Process the register data
	passwordHash, err := h.auth.GenerateHashFromPassword(registerReq.Password)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot generate hash from password",
			"details": err.Error(),
		})
		return
	}

	user := models.User{
		Login:        registerReq.Login,
		PasswordHash: passwordHash,
	}

	userUUID, err := h.srv.CreateUser(c, user)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot create user",
			"details": err.Error(),
		})
		return
	}

	tokenString, err := h.auth.BuildJWTString(userUUID)
	if err != nil {
		h.zlog.Error().Msgf("error building JWT string: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.SetCookie("user_uuid", tokenString, 0, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "user successfully registered",
		"login":   registerReq.Login,
	})
}

// userLogin is a handler used for users logging in
func (h *Handlers) userLogin(c *gin.Context) {
	var loginReq loginRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}
	// Process the login data
	user, err := h.srv.LoginUser(c, loginReq.Login)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "cannot login",
			"details": err.Error(),
		})
		return
	}

	if !h.auth.CheckPasswordHash(loginReq.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": apperrors.ErrInvalidPassword.Error(),
		})
		return
	}

	tokenString, err := h.auth.BuildJWTString(user.UUID)
	if err != nil {
		h.zlog.Error().Msgf("error building JWT string: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.SetCookie("user_uuid", tokenString, 0, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "user successfully logged in",
		"login":   loginReq.Login,
	})
}

// getUserUUIDFromRequest is a helper that retrieves user UUID from request
func (h *Handlers) getUserUUIDFromRequest(c *gin.Context) (uuid.UUID, error) {
	user, ok := c.Get("user_uuid")
	if !ok {
		return uuid.Nil, errors.New("user uuid not found")
	}

	userUUID, err := uuid.Parse(user.(string))
	if err != nil {
		h.zlog.Debug().Msgf("cannot parse user UUID: %v", err)
		return uuid.Nil, err
	}

	return userUUID, nil
}

// ping used for testing authentication middleware
func (h *Handlers) ping(c *gin.Context) {
	c.JSON(http.StatusOK, nil)

}

// storeText handles requests to store encrypted text data for the authenticated user.
func (h *Handlers) storeText(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	var storeTextReq text
	if err = c.ShouldBindJSON(&storeTextReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	storeText := models.Text{
		UserUUID:    userUUID,
		Name:        storeTextReq.Name,
		Description: storeTextReq.Description,
		UserText:    []byte(storeTextReq.Text),
	}

	err = h.srv.StoreSecret(c, "text", storeText)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot store text",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "text stored successfully",
	})
}

// getText handles requests to retrieve and decrypt a text secret by name.
func (h *Handlers) getText(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrNameRequired,
		})
		return
	}

	get, err := h.srv.DecryptSecret(c, "text", userUUID, name)
	getText, ok := get.(models.Text)
	if !ok {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrSecretInvalid.Error(),
		})
	}
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get text",
			"details": err.Error(),
		})
		return
	}

	var textResp text
	textResp.Name = getText.Name
	textResp.Description = getText.Description
	textResp.Text = string(getText.UserText)

	c.JSON(http.StatusOK, textResp)
}

// storeCredential handles requests to store encrypted login credentials.
func (h *Handlers) storeCredential(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	var storeCredentialReq credential
	if err = c.ShouldBindJSON(&storeCredentialReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	storeCredential := models.Credential{
		UserUUID:    userUUID,
		Name:        storeCredentialReq.Name,
		Description: storeCredentialReq.Description,
		Login:       []byte(storeCredentialReq.Login),
		Password:    []byte(storeCredentialReq.Password),
	}

	err = h.srv.StoreSecret(c, "credential", storeCredential)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot store credential",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "credential stored successfully",
	})
}

// getCredential handles requests to retrieve and decrypt credentials by name.
func (h *Handlers) getCredential(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrNameRequired,
		})
		return
	}

	get, err := h.srv.DecryptSecret(c, "credential", userUUID, name)
	getCredential, ok := get.(models.Credential)
	if !ok {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrSecretInvalid.Error(),
		})
	}
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get credential",
			"details": err.Error(),
		})
		return
	}

	var credentialResp credential
	credentialResp.Name = getCredential.Name
	credentialResp.Description = getCredential.Description
	credentialResp.Login = string(getCredential.Login)
	credentialResp.Password = string(getCredential.Password)

	c.JSON(http.StatusOK, credentialResp)
}

// storeCard handles requests to store encrypted bank card information.
func (h *Handlers) storeCard(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	var storeCardReq card
	if err = c.ShouldBindJSON(&storeCardReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	storeCard := models.Card{
		UserUUID:    userUUID,
		Name:        storeCardReq.Name,
		Description: storeCardReq.Description,
		CardNumber:  []byte(storeCardReq.CardNumber),
		Owner:       []byte(storeCardReq.Owner),
		ExpiresAt:   []byte(storeCardReq.ExpiresAt),
		CVC:         []byte(storeCardReq.CVC),
	}

	err = h.srv.StoreSecret(c, "card", storeCard)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot store card",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "card stored successfully",
	})
}

// getCard handles requests to retrieve and decrypt bank card data by name.
func (h *Handlers) getCard(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrNameRequired,
		})
		return
	}

	get, err := h.srv.DecryptSecret(c, "card", userUUID, name)
	getCard, ok := get.(models.Card)
	if !ok {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrSecretInvalid.Error(),
		})
	}
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get card",
			"details": err.Error(),
		})
		return
	}

	var cardResp card
	cardResp.Name = getCard.Name
	cardResp.Description = getCard.Description
	cardResp.CardNumber = string(getCard.CardNumber)
	cardResp.Owner = string(getCard.Owner)
	cardResp.ExpiresAt = string(getCard.ExpiresAt)
	cardResp.CVC = string(getCard.CVC)

	c.JSON(http.StatusOK, cardResp)
}

// storeFile handles requests to store encrypted files (up to 5 MB) as base64.
func (h *Handlers) storeFile(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	// Limit request body (~7 MB to account for base64 overhead)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 7<<20)

	var req file
	if err := c.ShouldBindJSON(&req); err != nil {
		// MaxBytesReader returns *http.MaxBytesError when limit is exceeded
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body exceeds 7 MB limit",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Quick estimate check before decoding
	if len(req.UserFile) > maxBase64Len {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File exceeds 5 MB limit"})
		return
	}

	data, err := base64.StdEncoding.DecodeString(req.UserFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64"})
		return
	}

	// Exact check on decoded size
	if len(data) > 5<<20 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File exceeds 5 MB limit"})
		return
	}

	storeFile := models.File{
		UserUUID:    userUUID,
		Name:        req.Name,
		Description: req.Description,
		UserFile:    data,
	}

	err = h.srv.StoreSecret(c, "file", storeFile)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot store file",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "file stored successfully",
	})
}

// getFile handles requests to retrieve and decrypt a file by name, returning it as base64.
func (h *Handlers) getFile(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrNameRequired,
		})
		return
	}

	get, err := h.srv.DecryptSecret(c, "file", userUUID, name)
	getFile, ok := get.(models.File)
	if !ok {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "invalid request body",
			"details": apperrors.ErrSecretInvalid.Error(),
		})
	}
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get file",
			"details": err.Error(),
		})
		return
	}

	var textResp file
	textResp.Name = getFile.Name
	textResp.Description = getFile.Description
	textResp.UserFile = base64.StdEncoding.EncodeToString(getFile.UserFile)

	c.JSON(http.StatusOK, textResp)
}

// getAllUserTexts handles requests to list all text secrets for the authenticated user.
func (h *Handlers) getAllUserTexts(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	allTexts, err := h.srv.GetAllUserTexts(c, userUUID)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user's stored texts",
			"details": err.Error(),
		})
		return
	}

	var allTextsResponse []allSecrets
	for _, userText := range allTexts {
		var textResp allSecrets
		textResp.Name = userText.Name
		textResp.Description = userText.Description

		allTextsResponse = append(allTextsResponse, textResp)
	}

	c.JSON(http.StatusOK, allTextsResponse)
}

// getAllUserCredentials handles requests to list all stored credentials for the authenticated user.
func (h *Handlers) getAllUserCredentials(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	allCredentials, err := h.srv.GetAllUserCredentials(c, userUUID)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user's stored credentials",
			"details": err.Error(),
		})
		return
	}

	var allCredentialsResponse []allSecrets
	for _, userCredential := range allCredentials {
		var credentialResponse allSecrets
		credentialResponse.Name = userCredential.Name
		credentialResponse.Description = userCredential.Description

		allCredentialsResponse = append(allCredentialsResponse, credentialResponse)
	}

	c.JSON(http.StatusOK, allCredentialsResponse)
}

// getAllUserCards handles requests to list all stored cards for the authenticated user.
func (h *Handlers) getAllUserCards(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	allCards, err := h.srv.GetAllUserCards(c, userUUID)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user's stored cards",
			"details": err.Error(),
		})
		return
	}

	var allCardsResponse []allSecrets
	for _, userCard := range allCards {
		var credentialResponse allSecrets
		credentialResponse.Name = userCard.Name
		credentialResponse.Description = userCard.Description

		allCardsResponse = append(allCardsResponse, credentialResponse)
	}

	c.JSON(http.StatusOK, allCardsResponse)
}

// getAllUserFiles handles requests to list all stored files for the authenticated user.
func (h *Handlers) getAllUserFiles(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	allFiles, err := h.srv.GetAllUserFiles(c, userUUID)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user's stored files",
			"details": err.Error(),
		})
		return
	}

	var allFilesResponse []allSecrets
	for _, userFile := range allFiles {
		var fileResponse allSecrets
		fileResponse.Name = userFile.Name
		fileResponse.Description = userFile.Description

		allFilesResponse = append(allFilesResponse, fileResponse)
	}

	c.JSON(http.StatusOK, allFilesResponse)
}

// deleteSecret handles requests to delete a secret of any type by name.
func (h *Handlers) deleteSecret(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	var delReq deleteRequest
	if err = c.ShouldBindJSON(&delReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	err = h.srv.DeleteSecret(c, delReq.Type, userUUID, delReq.Name)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot delete secret",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "secret deleted successfully",
	})
}

// updateText handles requests to update an existing text secret.
func (h *Handlers) updateText(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	var updateTextReq text
	if err = c.ShouldBindJSON(&updateTextReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	updateCredential := models.Text{
		UserUUID:    userUUID,
		Name:        updateTextReq.Name,
		Description: updateTextReq.Description,
		UserText:    []byte(updateTextReq.Text),
	}

	err = h.srv.UpdateSecret(c, "text", updateCredential)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot update text",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "text updated successfully",
	})
}

// updateCredential handles requests to update existing login credentials.
func (h *Handlers) updateCredential(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	var updateCredentialReq credential
	if err = c.ShouldBindJSON(&updateCredentialReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	updateCredential := models.Credential{
		UserUUID:    userUUID,
		Name:        updateCredentialReq.Name,
		Description: updateCredentialReq.Description,
		Login:       []byte(updateCredentialReq.Login),
		Password:    []byte(updateCredentialReq.Password),
	}

	err = h.srv.UpdateSecret(c, "credential", updateCredential)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot update credential",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "credential updated successfully",
	})
}

// updateCard handles requests to update existing bank card information.
func (h *Handlers) updateCard(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user uuid from request",
			"details": err.Error(),
		})
		return
	}

	var updateCardReq card
	if err = c.ShouldBindJSON(&updateCardReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	updateCard := models.Card{
		UserUUID:    userUUID,
		Name:        updateCardReq.Name,
		Description: updateCardReq.Description,
		CardNumber:  []byte(updateCardReq.CardNumber),
		Owner:       []byte(updateCardReq.Owner),
		ExpiresAt:   []byte(updateCardReq.ExpiresAt),
		CVC:         []byte(updateCardReq.CVC),
	}

	err = h.srv.UpdateSecret(c, "card", updateCard)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot update card",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "card updated successfully",
	})
}
