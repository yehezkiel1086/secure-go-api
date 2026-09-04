package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

// UserHandler handles HTTP requests for user-related endpoints.
type UserHandler struct {
	userService port.UserService
}

// NewUserHandler creates a new UserHandler with the injected UserService.
func NewUserHandler(userService port.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// RegisterUser registers a new user.
// @Summary      Register a new user
// @Description  Creates a new user account with hashed password and default User role (2001)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body domain.RegisterUserRequest true "User registration details"
// @Success      201  {object}  domain.UserResponse
// @Failure      400  {object}  map[string]string "Bad Request (e.g. validation failure)"
// @Failure      409  {object}  map[string]string "Conflict (email already registered)"
// @Failure      500  {object}  map[string]string "Internal Server Error"
// @Router       /register [post]
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req domain.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.userService.RegisterUser(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ConfirmEmail confirms user email via one-time token.
// @Summary      Confirm user email
// @Description  Verifies a user's email using a SHA-256 hashed verification token from query parameters
// @Tags         users
// @Produce      json
// @Param        token  query     string  true  "Email verification token"
// @Success      200    {object}  map[string]string "Email verified successfully"
// @Failure      400    {object}  map[string]string "Invalid or expired verification token"
// @Failure      500    {object}  map[string]string "Internal Server Error"
// @Router       /confirm-email [get]
func (h *UserHandler) ConfirmEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token query parameter is required"})
		return
	}

	err := h.userService.VerifyEmail(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalidToken) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification token"})
			return
		}
		if errors.Is(err, domain.ErrTokenExpired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "verification token has expired"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}

// GetUsers retrieves a paginated list of users.
// @Summary      List users
// @Description  Retrieves a paginated list of users ordered by creation date descending
// @Tags         users
// @Produce      json
// @Param        page       query     int  false  "Page number (default: 1)"
// @Param        page_size  query     int  false  "Items per page (default: 10, max: 100)"
// @Success      200        {object}  domain.PaginatedUsersResponse
// @Failure      500        {object}  map[string]string "Internal Server Error"
// @Router       /users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	res, err := h.userService.GetUsers(c.Request.Context(), int32(page), int32(pageSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetUserByID retrieves a user profile by ID.
// @Summary      Get user by ID
// @Description  Retrieves user details by their UUID primary key
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  domain.UserResponse
// @Failure      400  {object}  map[string]string "Invalid UUID format"
// @Failure      404  {object}  map[string]string "User not found"
// @Failure      500  {object}  map[string]string "Internal Server Error"
// @Router       /users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil || !id.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id format"})
		return
	}

	res, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// UpdateUserName updates a user's display name.
// @Summary      Update user name
// @Description  Updates the display name of a user identified by UUID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id       path      string                        true  "User UUID"
// @Param        request  body      domain.UpdateUserNameRequest  true  "Updated user display name"
// @Success      200      {object}  domain.UserResponse
// @Failure      400      {object}  map[string]string "Invalid UUID or request body"
// @Failure      404      {object}  map[string]string "User not found"
// @Failure      500      {object}  map[string]string "Internal Server Error"
// @Router       /users/{id} [patch]
func (h *UserHandler) UpdateUserName(c *gin.Context) {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil || !id.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id format"})
		return
	}

	var req domain.UpdateUserNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.userService.UpdateUserName(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, res)
}
