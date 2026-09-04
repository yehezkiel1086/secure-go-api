package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

// AuthHandler handles authentication endpoints (login, refresh, logout).
type AuthHandler struct {
	authService port.AuthService
	jwtCfg      *config.JWT
	appCfg      *config.App
}

// NewAuthHandler creates a new AuthHandler with the injected AuthService and configuration.
func NewAuthHandler(authService port.AuthService, jwtCfg *config.JWT, appCfg *config.App) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtCfg:      jwtCfg,
		appCfg:      appCfg,
	}
}

// setTokenCookies configures hardened HttpOnly cookies for access and refresh tokens.
func (h *AuthHandler) setTokenCookies(c *gin.Context, tokens *domain.TokenPair) {
	secure := h.appCfg.Env == "production"

	accessMaxAge := int(time.Until(tokens.AccessTokenExpiresAt).Seconds())
	if accessMaxAge > 0 {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("access_token", tokens.AccessToken, accessMaxAge, "/", "", secure, true)
	}

	refreshMaxAge := int(time.Until(tokens.RefreshTokenExpiresAt).Seconds())
	if refreshMaxAge > 0 {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("refresh_token", tokens.RefreshToken, refreshMaxAge, "/api/v1/refresh", "", secure, true)
	}
}

// clearTokenCookies clears active auth cookies by setting MaxAge to -1.
func (h *AuthHandler) clearTokenCookies(c *gin.Context) {
	secure := h.appCfg.Env == "production"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh_token", "", -1, "/api/v1/refresh", "", secure, true)
}

// Login authenticates a user and issues access/refresh tokens.
// @Summary      User login
// @Description  Authenticates user credentials, sets hardened HttpOnly cookies, and returns JWT token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.LoginRequest true "User credentials"
// @Success      200  {object}  domain.LoginResponse
// @Failure      400  {object}  map[string]string "Invalid request body"
// @Failure      401  {object}  map[string]string "Invalid credentials"
// @Failure      403  {object}  map[string]string "Email not verified"
// @Failure      500  {object}  map[string]string "Internal Server Error"
// @Router       /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		if errors.Is(err, domain.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, gin.H{"error": "email address is not verified. Please verify your email before logging in."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
			return
	}

	h.setTokenCookies(c, res.Tokens)
	c.JSON(http.StatusOK, res)
}

// RefreshToken rotates the access token and refresh token.
// @Summary      Refresh token
// @Description  Rotates access and refresh tokens, detecting token reuse and setting new HttpOnly cookies
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.RefreshTokenRequest false "Optional refresh token body if cookies are not used"
// @Success      200  {object}  domain.TokenPair
// @Failure      400  {object}  map[string]string "Missing refresh token"
// @Failure      401  {object}  map[string]string "Invalid, expired, or reused token"
// @Failure      500  {object}  map[string]string "Internal Server Error"
// @Router       /refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var tokenStr string

	// 1. Try to read from HttpOnly cookie
	if cookie, err := c.Cookie("refresh_token"); err == nil && cookie != "" {
		tokenStr = cookie
	}

	// 2. Try JSON body fallback
	if tokenStr == "" {
		var body domain.RefreshTokenRequest
		if err := c.ShouldBindJSON(&body); err == nil && body.RefreshToken != "" {
			tokenStr = body.RefreshToken
		}
	}

	// 3. Try Authorization header fallback
	if tokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenStr = strings.TrimSpace(parts[1])
			}
		}
	}

	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
		return
	}

	tokens, err := h.authService.RefreshToken(c.Request.Context(), tokenStr)
	if err != nil {
		h.clearTokenCookies(c)
		if errors.Is(err, domain.ErrTokenReuse) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token reuse detected: all sessions revoked"})
			return
		}
		if errors.Is(err, domain.ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has expired"})
			return
		}
		if errors.Is(err, domain.ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh token"})
		return
	}

	h.setTokenCookies(c, tokens)
	c.JSON(http.StatusOK, tokens)
}

// Logout invalidates the active session and clears cookies.
// @Summary      User logout
// @Description  Revokes user refresh tokens and clears authentication cookies
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string "Logged out successfully"
// @Router       /logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var tokenStr string
	if cookie, err := c.Cookie("refresh_token"); err == nil && cookie != "" {
		tokenStr = cookie
	}

	if tokenStr != "" {
		_ = h.authService.Logout(c.Request.Context(), tokenStr)
	}

	h.clearTokenCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
