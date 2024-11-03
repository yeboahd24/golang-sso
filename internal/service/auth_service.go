package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yeboahd24/sso/internal/model"
	"github.com/yeboahd24/sso/internal/repository"
	"golang.org/x/oauth2"
)

// Custom error types for better error handling
type AuthError struct {
	Code    string
	Message string
	Err     error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

const (
	ErrInvalidToken        = "INVALID_TOKEN"
	ErrUnverifiedEmail     = "UNVERIFIED_EMAIL"
	ErrInvalidDomain       = "INVALID_DOMAIN"
	ErrNetworkFailure      = "NETWORK_FAILURE"
	ErrInvalidResponse     = "INVALID_RESPONSE"
	ErrUserCreationFailed  = "USER_CREATION_FAILED"
	ErrTokenExchangeFailed = "TOKEN_EXCHANGE_FAILED"
)

type UserInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	VerifiedEmail bool   `json:"verified_email"`
	Picture       string `json:"picture"`
	HD            string `json:"hd"`  // G Suite hosted domain
	Sub           string `json:"sub"` // Google's unique identifier for the user
}

type AuthService interface {
	HandleCallback(ctx context.Context, code string) (*model.User, error)
	VerifySession(ctx context.Context, userID uint) (*model.User, error)
	GetAuthURL(state string) string
}

type authService struct {
	userRepo     repository.UserRepository
	oauth2Config *oauth2.Config
	httpClient   *http.Client
}

func NewAuthService(userRepo repository.UserRepository, oauth2Config *oauth2.Config) AuthService {
	// Create a secure HTTP client with custom settings
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		DisableKeepAlives: false,
		MaxIdleConns:      100,
		IdleConnTimeout:   90 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &authService{
		userRepo:     userRepo,
		oauth2Config: oauth2Config,
		httpClient:   httpClient,
	}
}

// internal/service/auth_service.go
func (s *authService) HandleCallback(ctx context.Context, code string) (*model.User, error) {
	// Get user info using the OAuth flow
	userInfo, err := s.getOAuthUserInfo(code)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		return nil, err
	}
	log.Printf("User info received: %+v", userInfo)

	if !strings.HasSuffix(userInfo.Email, "@mesika.org") {
		log.Printf("Unauthorized email domain: %s", userInfo.Email)
		return nil, &AuthError{
			Code:    ErrInvalidDomain,
			Message: "Only @mesika.org email addresses are allowed",
		}
	}

	// First check if user exists
	existingUser, err := s.userRepo.FindByEmail(userInfo.Email)
	if err != nil {
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}

	user := &model.User{
		Email:     userInfo.Email,
		Name:      userInfo.Name,
		SSOID:     userInfo.Sub,
		Role:      "user",
		LastLogin: time.Now(),
	}

	if existingUser != nil {
		// Update existing user
		user.ID = existingUser.ID               // Set the ID for update
		user.CreatedAt = existingUser.CreatedAt // Preserve creation time
	}

	if err := s.userRepo.CreateOrUpdate(user); err != nil {
		return nil, fmt.Errorf("failed to create/update user: %w", err)
	}

	return user, nil
}

func (s *authService) getUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx,
		"GET",
		"https://www.googleapis.com/oauth2/v2/userinfo",
		nil)
	if err != nil {
		return nil, &AuthError{
			Code:    ErrNetworkFailure,
			Message: "Failed to create request",
			Err:     err,
		}
	}

	// Set authorization header
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make request with secure client
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &AuthError{
			Code:    ErrNetworkFailure,
			Message: "Failed to fetch user info from Google",
			Err:     err,
		}
	}
	defer func() {
		// Properly drain and close response body
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &AuthError{
			Code: ErrInvalidResponse,
			Message: fmt.Sprintf("Invalid response from Google (Status: %d): %s",
				resp.StatusCode, string(body)),
		}
	}

	// Decode response with proper error handling
	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, &AuthError{
			Code:    ErrInvalidResponse,
			Message: "Failed to decode user info response",
			Err:     err,
		}
	}

	return &userInfo, nil
}

func (s *authService) VerifySession(ctx context.Context, userID uint) (*model.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, &AuthError{
			Code:    "SESSION_INVALID",
			Message: "Invalid or expired session",
			Err:     err,
		}
	}
	return user, nil
}

func (s *authService) GetAuthURL(state string) string {
	return s.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (s *authService) getOAuthUserInfo(code string) (*UserInfo, error) {
	// Exchange code for token
	token, err := s.oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		return nil, &AuthError{
			Code:    ErrTokenExchangeFailed,
			Message: "Failed to exchange auth code for token",
			Err:     err,
		}
	}

	// Get user info using the access token
	return s.getUserInfo(context.Background(), token.AccessToken)
}
