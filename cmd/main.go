// cmd/main.go
package main

import (
	"log"

	"github.com/yeboahd24/sso/config"
	"github.com/yeboahd24/sso/internal/handler"
	"github.com/yeboahd24/sso/internal/repository"
	"github.com/yeboahd24/sso/internal/route"
	"github.com/yeboahd24/sso/internal/service"
)

func main() {
	// Load configuration first
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize OAuth2 config
	oauth2Config := config.InitOAuth2Config(cfg)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, oauth2Config)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)

	// Initialize and start router
	r := route.SetupRouter(authHandler)
	r.Run(":8080")
}
