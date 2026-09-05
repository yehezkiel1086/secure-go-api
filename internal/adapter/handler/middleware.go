package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

func AuthMiddleware(cfg *config.JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// check authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenStr = strings.TrimSpace(parts[1])
			}
		}

		// fallback to httponly cookie
		if tokenStr == "" {
			if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
				tokenStr = cookie
			}
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
			return
		}

		claims, err := util.ParseToken(cfg, util.TokenAccess, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired access token"})
			return
		}

		c.Set("userID", claims.Subject)
		c.Set("role", claims.Role)
		c.Set("email", claims.Email)
		c.Set("claims", claims)

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userRole, ok := val.(int32)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid role claim"})
			return
		}

		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: insufficient permissions"})
	}
}
