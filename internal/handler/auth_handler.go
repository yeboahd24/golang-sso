// internal/handler/auth_handler.go
package handler

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/yeboahd24/sso/internal/service"
	"github.com/yeboahd24/sso/pkg/util"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) InitiateSSO(c *gin.Context) {
	state := util.GenerateRandomState()
	session := sessions.Default(c)

	// Store both raw and encoded state
	session.Set("state", state)
	session.Set("encoded_state", url.QueryEscape(state))

	if err := session.Save(); err != nil {
		log.Printf("Failed to save session: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "SESSION_ERROR",
			Message: "Failed to save session",
		})
		return
	}

	// Debug session after saving
	log.Printf("Session ID: %v", session.Get("state"))
	log.Printf("Stored state: %v", state)
	log.Printf("Encoded state: %v", url.QueryEscape(state))

	url := h.authService.GetAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// func (h *AuthHandler) InitiateSSO(c *gin.Context) {
// 	state := util.GenerateRandomState()
// 	session := sessions.Default(c)
// 	session.Clear()
// 	session.Set("state", state)
// 	session.Options(sessions.Options{
// 		Path:     "/",
// 		MaxAge:   300, // 5 minutes
// 		Secure:   true,
// 		HttpOnly: true,
// 		SameSite: http.SameSiteStrictMode,
// 	})
// 	if err := session.Save(); err != nil {
// 		c.JSON(http.StatusInternalServerError, ErrorResponse{
// 			Code:    "SESSION_ERROR",
// 			Message: "Failed to save session",
// 		})
// 		return
// 	}
// 	fmt.Printf("Stored state in session: %v\n", state)
// 	url := h.authService.GetAuthURL(state)
// 	c.Redirect(http.StatusTemporaryRedirect, url)
// }

func (h *AuthHandler) Callback(c *gin.Context) {
	session := sessions.Default(c)
	state := session.Get("state")

	log.Printf("Received state: %s, Expected state: %v", c.Query("state"), state)

	if state != c.Query("state") {
		log.Printf("State mismatch: received=%s, expected=%v", c.Query("state"), state)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_STATE",
			Message: "Invalid state parameter",
		})
		return
	}

	// Clear the state from session
	session.Delete("state")
	session.Save()

	log.Printf("OAuth callback received. Code: %s", c.Query("code"))
	user, err := h.authService.HandleCallback(c.Request.Context(), c.Query("code"))
	if err != nil {
		log.Printf("HandleCallback error: %T - %+v", err, err)
		// Type assert to get our custom error
		if authErr, ok := err.(*service.AuthError); ok {
			status := getStatusCodeForError(authErr.Code)
			log.Printf("Auth error: code=%s, message=%s", authErr.Code, authErr.Message)
			c.JSON(status, ErrorResponse{
				Code:    authErr.Code,
				Message: authErr.Message,
			})
			return
		}

		log.Printf("Callback error details: %+v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "INVALID_USER_DATA",
			Message: "Missing required user information from OAuth provider",
		})
		return
	}

	// Set session with secure options
	session.Set("user_id", user.ID)
	session.Set("email", user.Email)
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24, // 24 hours
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "SESSION_ERROR",
			Message: "Failed to save session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully authenticated",
		"user": gin.H{
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func getStatusCodeForError(code string) int {
	switch code {
	case service.ErrInvalidToken, service.ErrUnverifiedEmail, service.ErrInvalidDomain:
		return http.StatusUnauthorized
	case service.ErrNetworkFailure:
		return http.StatusServiceUnavailable
	case service.ErrInvalidResponse:
		return http.StatusBadGateway
	case service.ErrUserCreationFailed:
		return http.StatusInternalServerError
	case service.ErrTokenExchangeFailed:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func (h *AuthHandler) VerifySession(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, err := h.authService.VerifySession(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user": gin.H{
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func (h *AuthHandler) RenderLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "SSO Login",
	})
}
