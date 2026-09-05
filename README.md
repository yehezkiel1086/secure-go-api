# Secure Go API

A production-focused REST API written in Go, structured with Hexagonal Architecture (Ports and Adapters) and built with defense-in-depth security practices.

---

## Architecture Overview

Hexagonal Architecture decouples business domain logic from external frameworks, databases, and message brokers.

### Layer Responsibilities

- **`internal/core/domain`**: Entities (`User`, `Job`, `RefreshToken`, `EmailPayload`), domain types (`Role`), and domain errors. Zero external dependencies.
- **`internal/core/port`**: Primary driving ports (`AuthService`, `UserService`, `JobService`) and secondary driven ports (`UserRepository`, `AuthRepository`, `JobRepository`, `EmailPublisher`, `EmailSender`).
- **`internal/core/service`**: Core business rules, password policies, JWT token lifecycle with reuse detection, asynchronous email verification, and job management logic.
- **`internal/core/util`**: Cryptographic routines, bcrypt hashing, JWT issuance and verification, crypto random token generation, and RFC 5321/5322 email syntax validation.
- **`internal/adapter/handler`**: Driving HTTP adapter using Gin. Request deserialization, boundary validation, status code mapping, cookie handling, and Swagger docs integration.
- **`internal/adapter/storage/postgres`**: Driven persistence adapter implementing repositories with `pgx/v5` connection pool and `sqlc` generated queries with parameterized statements.
- **`internal/adapter/rabbitmq`**: Driven messaging adapter for asynchronous background email dispatching via RabbitMQ, including an SMTP worker delivering to Mailpit or mail servers.
- **`internal/adapter/config`**: Environment variable parsing and typed application configuration.

---

## Security Practices

### 1. Input Validation at the Boundary

Input constraints are enforced at the HTTP boundary before reaching domain services:

```go
// bad: unvalidated struct accepting arbitrary lengths and inverted salary ranges
type CreateJobRequest struct {
	Title     string `json:"title"`
	SalaryMin *int64 `json:"salary_min"`
	SalaryMax *int64 `json:"salary_max"`
}
```

```go
// good: struct validation tags and domain boundary checks
type CreateJobRequest struct {
	Title       string `json:"title" binding:"required,min=3,max=150"`
	Description string `json:"description" binding:"required,min=10,max=10000"`
	Company     string `json:"company" binding:"required,min=2,max=100"`
	Location    string `json:"location" binding:"required,min=2,max=100"`
	SalaryMin   *int64 `json:"salary_min,omitempty" binding:"omitempty,gte=0"`
	SalaryMax   *int64 `json:"salary_max,omitempty" binding:"omitempty,gte=0"`
}

if req.SalaryMin != nil && req.SalaryMax != nil && *req.SalaryMin > *req.SalaryMax {
	return nil, domain.ErrInvalidSalaryRange
}
```

### 2. Parameterized Database Queries (SQL Injection Prevention)

Database operations pass values strictly as parameterized arguments via `sqlc` and `pgx/v5`:

```go
// bad: string concatenation vulnerable to sql injection
query := fmt.Sprintf("SELECT id, email, password_hash FROM users WHERE email = '%s'", email)
row := db.QueryRow(ctx, query)
```

```go
// good: parameterized prepared queries via sqlc
row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
	Name:         user.Name,
	Email:        user.Email,
	PasswordHash: user.PasswordHash,
	Role:         user.Role,
})
```

### 3. Elimination of Command Injection

External processes are not invoked via shell wrappers. All network communications rely on native Go drivers:

```go
// bad: spawning a subshell with user-controlled input
cmd := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' | mail -s '%s' %s", body, subject, to))
_ = cmd.Run()
```

```go
// good: native network protocol client without shell execution
auth := smtp.PlainAuth("", s.user, s.password, s.host)
msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
	s.from, to, subject, body)
return smtp.SendMail(s.addr, auth, s.from, []string{to}, []byte(msg))
```

### 4. Strongly-Typed Deserialization

HTTP handlers bind incoming JSON into concrete domain structs rather than untyped containers:

```go
// bad: untyped container prone to type confusion and unvalidated fields
var payload map[string]any
if err := c.ShouldBindJSON(&payload); err != nil {
	return
}
title := payload["title"].(string)
```

```go
// good: strongly-typed struct with validation tags
var req domain.CreateJobRequest
if err := c.ShouldBindJSON(&req); err != nil {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	return
}
```

### 5. Secrets Management and Sensitive Data Handling

Credentials and tokens are never stored in plaintext. Passwords use `bcrypt`, and tokens are hashed at rest with SHA-256:

```go
// bad: plaintext password and raw token stored in database
user.Password = req.Password
user.VerificationToken = rawToken
```

```go
// good: bcrypt password hashing and sha-256 token hashing at rest
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
tokenHash := util.HashToken(rawToken)
```

### 6. Explicit Error Handling and Information Leak Prevention

Internal infrastructure details and database errors are suppressed. Endpoints resist user enumeration:

```go
// bad: exposing internal sql errors and leaking account existence
if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	return
}
if user == nil {
	c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	return
}
```

```go
// good: masked domain errors and uniform response to prevent user enumeration
user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
if err != nil {
	if errors.Is(err, domain.ErrNotFound) {
		return nil // silent return to prevent email enumeration
	}
	return domain.ErrInternal
}
```

### 7. Context and Timeout Propagation

Request contexts flow from driving HTTP handlers down through domain services to database queries:

```go
// bad: ignoring request context; query continues running after client disconnects
go func() {
	_, _ = r.queries.CreateJob(context.Background(), params)
}()
```

```go
// good: propagating gin request context with cancellation
createdJob, err := h.jobService.CreateJob(c.Request.Context(), &req, userID)
if err != nil {
	h.handleError(c, err)
	return
}
```

### 8. Race Condition Prevention and Concurrency Safety

Handlers and domain services maintain zero mutable shared state:

```go
// bad: package-level shared state accessed concurrently without synchronization
var jobCache = make(map[string]*domain.Job)

func (h *JobHandler) GetJobByID(c *gin.Context) {
	id := c.Param("id")
	if job, ok := jobCache[id]; ok {
		c.JSON(http.StatusOK, job)
		return
	}
	jobCache[id] = fetchedJob // data race under concurrent requests
}
```

```go
// good: stateless handlers with dependency injection verified by -race
type JobHandler struct {
	jobService port.JobService
}

func NewJobHandler(jobService port.JobService) *JobHandler {
	return &JobHandler{jobService: jobService}
}
```

### 9. Token Lifecycle and Refresh Token Reuse Detection

Single-use refresh token rotation invalidates consumed tokens and triggers session revocation upon replay detection:

```go
// bad: accepting refresh tokens repeatedly without rotation or reuse detection
tokenRecord, err := s.authRepo.GetRefreshToken(ctx, hash)
if err != nil {
	return nil, err
}
return s.generateAccessToken(tokenRecord.UserID)
```

```go
// good: detect token replay and revoke all active sessions for the user
if tokenRecord.IsRevoked {
	_ = s.authRepo.RevokeAllUserRefreshTokens(ctx, tokenRecord.UserID)
	return nil, domain.ErrTokenReuse
}
if err := s.authRepo.RevokeRefreshToken(ctx, tokenRecord.ID); err != nil {
	return nil, domain.ErrInternal
}
```

### 10. Least Privilege HTTP Routing and Scoped Middleware

Endpoints enforce access control at the route group boundary, and audit fields are extracted strictly from verified token context:

```go
// bad: trusting client-supplied creator id or role in json body
v1.POST("/jobs", func(c *gin.Context) {
	var req struct {
		CreatedBy string `json:"created_by"`
	}
	_ = c.ShouldBindJSON(&req) // attacker can spoof any user id
})
```

```go
// good: route group middleware enforcement and context-derived identity
adminOnly := v1.Group("/", auth, RoleMiddleware(domain.RoleAdmin))
adminOnly.POST("/jobs", jobHandler.CreateJob)

// creator id extracted strictly from verified token claims
userIDVal, _ := c.Get("userID")
userID := userIDVal.(uuid.UUID)
```

---

## Project Structure

```
secure-go-api/
├── cmd/
│   ├── http/
│   │   └── main.go                    # application entry point & dependency injection
│   └── migrate/
│       └── main.go                    # standalone database migration runner
├── db/
│   └── migrations/
│       └── schema.sql                 # postgresql ddl schema
├── docs/                              # generated swagger / openapi specs
├── internal/
│   ├── adapter/
│   │   ├── config/
│   │   │   └── config.go              # typed configuration loader
│   │   ├── handler/
│   │   │   ├── auth.go                # authentication http handler
│   │   │   ├── job.go                 # job crud http handler
│   │   │   ├── middleware.go          # jwt auth & rbac middleware
│   │   │   ├── router.go              # gin router configuration
│   │   │   └── user.go                # user management http handler
│   │   ├── rabbitmq/
│   │   │   ├── client.go              # rabbitmq topology & connection
│   │   │   ├── consumer.go            # background email consumer worker
│   │   │   ├── publisher.go           # rabbitmq event publisher
│   │   │   └── smtp.go                # smtp delivery implementation
│   │   └── storage/
│   │       └── postgres/
│   │           ├── db.go              # pgxpool connection & migration
│   │           ├── repository/
│   │           │   ├── auth.go        # refresh token repository
│   │           │   ├── job.go         # job repository
│   │           │   └── user.go        # user repository
│   │           └── sqlc/              # sqlc generated queries
│   └── core/
│       ├── domain/                    # entities, errors, and dtos
│       ├── port/                      # driven and driving port interfaces
│       ├── service/                   # core business services
│       └── util/                      # jwt, bcrypt, token, and validator utilities
├── .env.example
├── .gitignore
├── Dockerfile
├── Taskfile.yml
├── docker-compose.yaml                # postgres, rabbitmq, and mailpit services
├── go.mod
└── go.sum
```

---

## API Endpoints

Base URL: `/api/v1`

### Public Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | System health check |
| `GET` | `/swagger/*any` | Interactive Swagger UI |
| `POST` | `/api/v1/register` | Register a new user |
| `GET` | `/api/v1/confirm-email` | Confirm email using token |
| `POST` | `/api/v1/resend-verification` | Resend verification email (anti-enumeration) |
| `POST` | `/api/v1/login` | Authenticate user and issue tokens |
| `POST` | `/api/v1/refresh` | Rotate access and refresh tokens |

### Authenticated Endpoints (`RoleUser: 2001`, `RoleAdmin: 5150`)

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/logout` | Bearer / Cookie | Invalidate session |
| `GET` | `/api/v1/jobs` | Bearer / Cookie | List jobs with pagination & search |
| `GET` | `/api/v1/jobs/:id` | Bearer / Cookie | Retrieve job details by UUID |

### Admin-Only Endpoints (`RoleAdmin: 5150`)

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/users` | Bearer / Cookie | List registered users |
| `GET` | `/api/v1/users/:id` | Bearer / Cookie | Retrieve user by UUID |
| `PATCH` | `/api/v1/users/:id` | Bearer / Cookie | Update user name |
| `POST` | `/api/v1/jobs` | Bearer / Cookie | Create job listing |
| `PATCH` | `/api/v1/jobs/:id` | Bearer / Cookie | Partially update job listing |
| `DELETE` | `/api/v1/jobs/:id` | Bearer / Cookie | Delete job listing |

---

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Task (optional)

### 1. Environment Setup

Copy `.env.example` to `.env`:

```bash
cp .env.example .env
```

Default configuration variables:

```env
APP_NAME=secure-go-api
APP_ENV=development

HTTP_HOST=127.0.0.1
HTTP_PORT=8080
HTTP_ALLOWED_ORIGINS=http://127.0.0.1:3000

DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=secure_go_api
DB_SSLMODE=disable

RABBITMQ_HOST=127.0.0.1
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest

SMTP_HOST=127.0.0.1
SMTP_PORT=1025
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=no-reply@secureapi.local

ACCESS_TOKEN_SECRET=your_access_token_secret_key_at_least_32_bytes_long
REFRESH_TOKEN_SECRET=your_refresh_token_secret_key_at_least_32_bytes_long
ACCESS_TOKEN_DURATION=15     # in minutes
REFRESH_TOKEN_DURATION=7     # in days
```

### 2. Start Infrastructure Services

Start PostgreSQL, RabbitMQ, and Mailpit containers:

```bash
docker compose up -d
```

- PostgreSQL: `localhost:5432`
- RabbitMQ: `localhost:5672` (Management: `http://localhost:15672`)
- Mailpit Web UI: `http://localhost:8025`

### 3. Database Migration

Migrations run automatically upon server startup. To run migrations manually:

```bash
go run cmd/migrate/main.go
```

### 4. Run the API Server

```bash
go run cmd/http/main.go
```

Or with live reload via Air:

```bash
task dev
```

The server starts on `http://127.0.0.1:8080`.
Swagger documentation is available at `http://127.0.0.1:8080/swagger/index.html`.

---

## Testing

Run unit and integration test suites with race detection:

```bash
go test -race -v ./...
```

Run static analysis:

```bash
go vet ./...
```

Regenerate Swagger documentation:

```bash
task swagger:gen
```
