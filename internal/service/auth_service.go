package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
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
	HD            string `json:"hd"` // G Suite hosted domain
}

type AuthService interface {
	HandleCallback(ctx context.Context, code, state string) (*model.User, error)
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

func (s *authService) HandleCallback(ctx context.Context, code, state string) (*model.User, error) {
	// Exchange code for token with context
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, &AuthError{
			Code:    ErrTokenExchangeFailed,
			Message: "Failed to exchange authorization code for token",
			Err:     err,
		}
	}

	// Validate token
	if !token.Valid() {
		return nil, &AuthError{
			Code:    ErrInvalidToken,
			Message: "Invalid or expired token",
		}
	}

	// Get user info with context
	userInfo, err := s.getUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, err // Error is already wrapped
	}

	// Validate email verification
	if !userInfo.VerifiedEmail {
		return nil, &AuthError{
			Code:    ErrUnverifiedEmail,
			Message: "Email address is not verified by Google",
		}
	}

	// Validate email domain
	if !strings.HasSuffix(userInfo.Email, "@mesika.org") {
		return nil, &AuthError{
			Code: ErrInvalidDomain,
			Message: fmt.Sprintf("Invalid email domain. Expected @mesika.org, got %s",
				strings.Split(userInfo.Email, "@")[1]),
		}
	}

	// Create or update user
	user := &model.User{
		Email:     userInfo.Email,
		Name:      userInfo.Name,
		LastLogin: time.Now(),
	}

	if err := s.userRepo.CreateOrUpdate(user); err != nil {
		return nil, &AuthError{
			Code:    ErrUserCreationFailed,
			Message: "Failed to create or update user record",
			Err:     err,
		}
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
