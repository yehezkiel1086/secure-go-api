package handler

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/yehezkiel1086/secure-go-api/docs"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

type Router struct {
	r    *gin.Engine
	conf *config.Container
}

// NewRouter constructs and configures the Gin HTTP router with all application routes.
func NewRouter(
	conf *config.Container,
	userHandler *UserHandler,
	authHandler *AuthHandler,
) *Router {
	if conf.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Swagger UI documentation endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Global Healthcheck
	// @Summary      Health check
	// @Description  Returns system health status and configuration details
	// @Tags         health
	// @Produce      json
	// @Success      200  {object}  map[string]string
	// @Router       /health [get]
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"app":    conf.App.Name,
			"env":    conf.App.Env,
		})
	})

	auth := AuthMiddleware(conf.JWT)

	// API v1 route group
	v1 := r.Group("/api/v1")
	{
		// public routes (no authentication required)
		v1.POST("/register", userHandler.RegisterUser)
		v1.GET("/confirm-email", userHandler.ConfirmEmail)
		v1.POST("/login", authHandler.Login)
		v1.POST("/refresh", authHandler.RefreshToken)

		// authenticated routes (requires valid token + user or admin role)
		authenticated := v1.Group("/", auth, RoleMiddleware(domain.RoleUser, domain.RoleAdmin))
		{
			authenticated.POST("/logout", authHandler.Logout)
		}

		// admin-only routes (requires valid token + admin role)
		adminOnly := v1.Group("/", auth, RoleMiddleware(domain.RoleAdmin))
		{
			adminOnly.GET("/users", userHandler.GetUsers)
			adminOnly.GET("/users/:id", userHandler.GetUserByID)
			adminOnly.PATCH("/users/:id", userHandler.UpdateUserName)
		}
	}

	return &Router{
		r:    r,
		conf: conf,
	}
}

// ServeHTTP implements the http.Handler interface for testing and interoperability.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.r.ServeHTTP(w, req)
}

// Run starts the HTTP server listening on the configured host and port.
func (r *Router) Run() error {
	addr := net.JoinHostPort(r.conf.HTTP.Host, r.conf.HTTP.Port)
	return r.r.Run(addr)
}
