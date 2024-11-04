# Golang SSO (Single Sign-On) Service

A robust Single Sign-On service built with Go, leveraging OAuth2 for authentication with Google.

![SSO Architecture](images/sso.png)

## Features

- OAuth2 authentication with Google
- Session management
- User creation and updates
- Secure cookie handling
- Domain-specific email validation (@mesika.org)
- Clean architecture with separation of concerns

## Prerequisites

- Go 1.18+
- PostgreSQL 13+
- Google OAuth2 credentials

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yeboahd24/golang-sso.git
   cd golang-sso

2. Install dependencies:
    ```bash
    go mod download
    ```

3. Configure the application:

    1. Copy config/config.example.yml to config/config.yml
    2. Update the database connection details in config.yml
    3. Add your Google OAuth2 credentials (client ID, client secret) to config.yml

4. Build and run the application:
    ```bash
    go run cmd/main.go
    ```


Usage

    Access the login page at `http://localhost:8080`
    Click "Continue with SSO" to initiate the OAuth flow
    Authenticate with your Google account (must be @mesika.org domain)
    Upon successful authentication, you'll be redirected back to the application


API Endpoints

    GET /api/auth/sso: Initiates the SSO process
    GET /api/auth/callback: OAuth callback URL
    GET /api/auth/verify: Verifies the current session
    POST /api/auth/logout: Logs out the current user


Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
License

This project is licensed under the MIT License.
