// internal/route/route.go
package route

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/yeboahd24/sso/config"
	"github.com/yeboahd24/sso/internal/handler"
	"github.com/yeboahd24/sso/internal/middleware"
)

func SetupRouter(authHandler *handler.AuthHandler) *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", authHandler.RenderLoginPage) // Render the login page

	// Session middleware
	store := cookie.NewStore(config.GetSessionSecret())
	r.Use(sessions.Sessions("auth-session", store))

	// Auth routes
	auth := r.Group("/api/auth")
	{
		auth.GET("/sso", authHandler.InitiateSSO)
		auth.GET("/callback", authHandler.Callback)
		auth.GET("/verify", middleware.AuthRequired(), authHandler.VerifySession)
		auth.POST("/logout", middleware.AuthRequired(), authHandler.Logout)
	}

	return r
}
