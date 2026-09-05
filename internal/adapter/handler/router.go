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

func NewRouter(
	conf *config.Container,
	userHandler *UserHandler,
	authHandler *AuthHandler,
	jobHandler *JobHandler,
) *Router {
	if conf.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

	v1 := r.Group("/api/v1")
	{
		// public routes
		v1.POST("/register", userHandler.RegisterUser)
		v1.GET("/confirm-email", userHandler.ConfirmEmail)
		v1.POST("/resend-verification", userHandler.ResendVerification)
		v1.POST("/login", authHandler.Login)
		v1.POST("/refresh", authHandler.RefreshToken)

		// authenticated (user or admin)
		authenticated := v1.Group("/", auth, RoleMiddleware(domain.RoleUser, domain.RoleAdmin))
		{
			authenticated.POST("/logout", authHandler.Logout)
			authenticated.GET("/jobs", jobHandler.GetJobs)
			authenticated.GET("/jobs/:id", jobHandler.GetJobByID)
		}

		// admin only
		adminOnly := v1.Group("/", auth, RoleMiddleware(domain.RoleAdmin))
		{
			adminOnly.GET("/users", userHandler.GetUsers)
			adminOnly.GET("/users/:id", userHandler.GetUserByID)
			adminOnly.PATCH("/users/:id", userHandler.UpdateUserName)

			adminOnly.POST("/jobs", jobHandler.CreateJob)
			adminOnly.PATCH("/jobs/:id", jobHandler.UpdateJob)
			adminOnly.DELETE("/jobs/:id", jobHandler.DeleteJob)
		}
	}

	return &Router{
		r:    r,
		conf: conf,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.r.ServeHTTP(w, req)
}

func (r *Router) Run() error {
	addr := net.JoinHostPort(r.conf.HTTP.Host, r.conf.HTTP.Port)
	return r.r.Run(addr)
}
