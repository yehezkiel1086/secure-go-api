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

type JobHandler struct {
	jobService port.JobService
}

func NewJobHandler(jobService port.JobService) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

// @Summary      Create a job listing
// @Description  Creates a new job listing with authenticated admin as creator (Admin only)
// @Tags         jobs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.CreateJobRequest true "Job listing details"
// @Success      201  {object}  domain.JobResponse
// @Failure      400  {object}  map[string]string "Bad Request (validation error or invalid salary range)"
// @Failure      401  {object}  map[string]string "Unauthorized"
// @Failure      403  {object}  map[string]string "Forbidden: insufficient permissions"
// @Failure      500  {object}  map[string]string "Internal Server Error"
// @Router       /jobs [post]
func (h *JobHandler) CreateJob(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var adminID pgtype.UUID
	if err := adminID.Scan(userIDVal); err != nil || !adminID.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user identity in token"})
		return
	}

	var req domain.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.jobService.CreateJob(c.Request.Context(), adminID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSalaryRange) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job listing"})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// @Summary      Get job listing by ID
// @Description  Retrieves details of a specific job listing (User or Admin)
// @Tags         jobs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Job UUID"
// @Success      200  {object}  domain.JobResponse
// @Failure      400  {object}  map[string]string "Invalid UUID format"
// @Failure      401  {object}  map[string]string "Unauthorized"
// @Failure      404  {object}  map[string]string "Job not found"
// @Failure      500  {object}  map[string]string "Internal Server Error"
// @Router       /jobs/{id} [get]
func (h *JobHandler) GetJobByID(c *gin.Context) {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil || !id.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id format"})
		return
	}

	res, err := h.jobService.GetJobByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job listing not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve job listing"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// @Summary      List job listings
// @Description  Retrieves paginated job listings with optional full-text search (User or Admin)
// @Tags         jobs
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "Page number (default: 1)"
// @Param        page_size  query     int     false  "Items per page (default: 10, max: 100)"
// @Param        q          query     string  false  "Search query across job titles"
// @Success      200        {object}  domain.PaginatedJobsResponse
// @Failure      401        {object}  map[string]string "Unauthorized"
// @Failure      500        {object}  map[string]string "Internal Server Error"
// @Router       /jobs [get]
func (h *JobHandler) GetJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	query := c.Query("q")

	res, err := h.jobService.GetJobs(c.Request.Context(), int32(page), int32(pageSize), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve job listings"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// @Summary      Update a job listing
// @Description  Partially updates a job listing (Admin only)
// @Tags         jobs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                   true  "Job UUID"
// @Param        request  body      domain.UpdateJobRequest  true  "Updated job details"
// @Success      200      {object}  domain.JobResponse
// @Failure      400      {object}  map[string]string "Invalid UUID or request body"
// @Failure      401      {object}  map[string]string "Unauthorized"
// @Failure      403      {object}  map[string]string "Forbidden: insufficient permissions"
// @Failure      404      {object}  map[string]string "Job not found"
// @Failure      500      {object}  map[string]string "Internal Server Error"
// @Router       /jobs/{id} [patch]
func (h *JobHandler) UpdateJob(c *gin.Context) {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil || !id.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id format"})
		return
	}

	var req domain.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.jobService.UpdateJob(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job listing not found"})
			return
		}
		if errors.Is(err, domain.ErrInvalidSalaryRange) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update job listing"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// @Summary      Delete a job listing
// @Description  Permanently deletes a job listing (Admin only)
// @Tags         jobs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Job UUID"
// @Success      200  {object}  map[string]string "Job deleted successfully"
// @Failure      400  {object}  map[string]string "Invalid UUID format"
// @Failure      401  {object}  map[string]string "Unauthorized"
// @Failure      403  {object}  map[string]string "Forbidden: insufficient permissions"
// @Failure      404  {object}  map[string]string "Job not found"
// @Failure      500  {object}  map[string]string "Internal Server Error"
// @Router       /jobs/{id} [delete]
func (h *JobHandler) DeleteJob(c *gin.Context) {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil || !id.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id format"})
		return
	}

	err := h.jobService.DeleteJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job listing not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete job listing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job listing deleted successfully"})
}
