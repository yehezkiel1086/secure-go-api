package handler

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/yehezkiel1086/secure-go-api/docs"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
)

type Router struct {
	r    *gin.Engine
	conf *config.Container
}

// NewRouter constructs and configures the Gin HTTP router with all application routes.
func NewRouter(
	conf *config.Container,
	userHandler *UserHandler,
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

	// API v1 route group
	v1 := r.Group("/api/v1")
	{
		// Public routes
		v1.POST("/register", userHandler.RegisterUser)
		v1.GET("/confirm-email", userHandler.ConfirmEmail)

		// User routes
		users := v1.Group("/users")
		{
			users.GET("", userHandler.GetUsers)
			users.GET("/:id", userHandler.GetUserByID)
			users.PATCH("/:id", userHandler.UpdateUserName)
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
