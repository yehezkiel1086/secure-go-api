# Secure Go API

A production-focused REST API written in Go, structured with Hexagonal Architecture (Ports and Adapters) and built with defense-in-depth application security practices.

---

## Architecture Overview

This project implements Hexagonal Architecture (Ports and Adapters) to decouple business domain logic from external frameworks, databases, and transport protocols.

### Layer Responsibilities

- **`internal/core/domain`**: Defines entities (`User`, `Job`), domain-level types (`Role`), and domain errors (`ErrNotFound`, `ErrTokenReuse`, `ErrInvalidCredentials`). Contains zero external dependencies.
- **`internal/core/port`**: Declares interfaces for primary drivers (services like `AuthService`, `UserService`, `JobService`) and secondary driven components (`UserRepository`, `JobRepository`).
- **`internal/core/service`**: Encapsulates core business rules, password policies, token lifecycle workflows, and email verification logic.
- **`internal/core/util`**: Cryptographic routines, bcrypt hashing, JWT issuance and verification, and secure random byte generation.
- **`internal/adapter/handler`**: Driving HTTP adapter using Gin. Handles HTTP request deserialization, boundary validation, status code mapping, and cookie orchestration.
- **`internal/adapter/storage/postgres`**: Driven adapter implementing repository ports using GORM and PostgreSQL with parameterized queries.
- **`internal/adapter/config`**: Environment variable loading and typed application configuration.

---

## Security Best Practices

The following ten security practices are implemented throughout this codebase:

### 1. Input Validation at the Boundary
Input is validated at the HTTP adapter boundary before reaching core services or persistence layers. The API relies on `go-playground/validator` struct tags to enforce field constraints, email formats, and string lengths.

```go
type UserRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name" binding:"required"`
}
```

Invalid payloads are rejected at the handler level with `400 Bad Request` prior to executing any business operations.

### 2. Parameterized Database Queries (SQL Injection Prevention)
No raw SQL string concatenation is used. All database operations in `internal/adapter/storage/postgres/repository` pass input as explicit parameters to GORM/PostgreSQL query builders:

```go
// correct: parameterized lookup
db.WithContext(ctx).Where("email = ?", email).First(&user)

// correct: parameterized ID update
db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Updates(payload)
```

### 3. Elimination of Command Injection
The application completely avoids shell execution (`os/exec`, `sh -c`, or equivalents). External operations, such as SMTP email transport (`gopkg.in/gomail.v2`) and database communication (`gorm.io/driver/postgres`), use native Go drivers and isolated network protocols rather than invoking system binaries.

### 4. Strongly-Typed Deserialization
Payloads are deserialized directly into concrete, typed structs instead of untyped `interface{}` or `map[string]any` containers:

```go
func (ah *AuthHandler) Login(c *gin.Context) {
    var req LoginUserReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
        return
    }
    // ...
}
```

This prevents unexpected type pollution, malformed type assertions, and unvalidated parameter injection.

### 5. Secrets Management and Sensitive Data Handling
- **No Hardcoded Secrets**: Secrets and database credentials are read through `internal/adapter/config` from environment variables.
- **Git Hygiene**: Local `.env` files and local database volumes (`postgres_data/`) are explicitly excluded via `.gitignore`.
- **Token Hashing at Rest**: Refresh tokens and password reset tokens are stored as SHA-256 hashes in the database (`util.HashToken(token)`), ensuring that a database dump does not expose active session tokens or reset capabilities.
- **Cryptographic Randomness**: Single-use tokens are generated using `crypto/rand` with 32 bytes of secure entropy.
- **Password Hashing**: Passwords are encrypted using `golang.org/x/crypto/bcrypt` with a secure cost factor.

### 6. Explicit Error Handling and Information Leak Prevention
Errors are wrapped with operational context using Go's `%w` formatting verb for internal tracing, while sensitive database details and stack traces are suppressed from client responses.

```go
// core service wraps error with context
if err := as.userRepo.SetPasswordResetToken(ctx, user.ID, hashedToken, expiry); err != nil {
    return fmt.Errorf("storing reset token: %w", err)
}

// handler returns clean, generic message to client
c.JSON(http.StatusInternalServerError, gin.H{
    "error": domain.ErrInternal.Error(),
    "code":  "INTERNAL_ERROR",
})
```

- **User Enumeration Defense**: The `RequestPasswordReset` endpoint returns a constant success response regardless of whether the requested email exists in the system.

### 7. Context and Timeout Propagation
Every service and repository port interface mandates `context.Context` as its first parameter. HTTP request contexts (`c.Request.Context()`) flow through the domain layer into database queries (`db.WithContext(ctx)`).

```go
type UserRepository interface {
    CreateUser(ctx context.Context, user *domain.User) (*domain.UserResponse, error)
    GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
    // ...
}
```

If a client disconnects or an upstream timeout expires, ongoing database operations are aborted, preventing connection pool exhaustion and runaway goroutines.

### 8. Race Condition Prevention and Concurrency Safety
- Handlers and services are designed as stateless components.
- Shared configuration parameters (such as token expiration intervals) are pre-calculated during initialization in `NewAuthHandler` to eliminate concurrent type conversions or read-write conflicts during request processing.
- The codebase is validated using Go's built-in race detector:
  ```bash
  go test -race ./...
  go build -race ./...
  ```

### 9. Dependency Hygiene and Supply Chain Auditing
- Dependencies are locked with checksum verification via `go.mod` and `go.sum`.
- The repository includes automated vulnerability checks via CI (`.github/workflows/security.yml`), scanning dependencies and code for known vulnerabilities and producing CycloneDX SBOMs.
- Local vulnerability auditing is performed via Google's official scanner:
  ```bash
  go install golang.org/x/vuln/cmd/govulncheck@latest
  govulncheck ./...
  ```

### 10. Least Privilege HTTP Routing and Scoped Middleware
Endpoints are grouped into explicit privilege tiers with middleware applied at the route group boundary rather than inside individual handlers:

```go
// 1. public routes (no authentication required)
pb := r.Group("/api/v1")
pb.POST("/login", authHandler.Login)
pb.POST("/register", userHandler.RegisterUser)

// 2. authenticated user routes (requires valid token + user/admin role)
us := pb.Group("/", auth, RoleMiddleware(domain.UserRole, domain.AdminRole))
us.GET("/jobs", jobHandler.GetJobs)
us.GET("/jobs/:id", jobHandler.GetJobById)

// 3. admin-only routes (requires valid token + admin role)
ad := pb.Group("/", auth, RoleMiddleware(domain.AdminRole))
ad.GET("/users", userHandler.GetUsers)
ad.POST("/jobs", jobHandler.CreateJob)
ad.DELETE("/jobs/:id", jobHandler.DeleteJob)
```

- **Cookie Hardening**: Authentication tokens are set as `HttpOnly` cookies. The refresh token cookie is further constrained to the `/api/v1/refresh` path.
- **Refresh Token Reuse Detection**: If a consumed or invalid refresh token is used, the system clears active cookies and revokes sessions (`domain.ErrTokenReuse`).

---

## Project Structure

```
secure-go-api/
├── cmd/
│   └── http/
│       └── main.go                    # Entry point & dependency wiring
├── internal/
│   ├── adapter/
│   │   ├── config/
│   │   │   └── config.go              # Environment configuration loader
│   │   ├── handler/
│   │   │   ├── auth.go                # Authentication endpoints & cookie handling
│   │   │   ├── job.go                 # Job CRUD HTTP handlers
│   │   │   ├── middleware.go          # Auth & Role verification middleware
│   │   │   ├── router.go              # Route grouping & CORS configuration
│   │   │   └── user.go                # User management handlers
│   │   └── storage/
│   │       └── postgres/
│   │           ├── db.go              # PostgreSQL database connection
│   │           └── repository/
│   │               ├── job.go         # Job GORM repository
│   │               └── user.go        # User GORM repository
│   └── core/
│       ├── domain/
│       │   ├── error.go               # Domain error definitions
│       │   ├── job.go                 # Job domain entity
│       │   ├── jwt.go                 # JWT claims definitions
│       │   └── user.go                # User domain model and request/response DTOs
│       ├── port/
│       │   ├── auth.go                # Auth service port
│       │   ├── job.go                 # Job repository and service ports
│       │   └── user.go                # User repository and service ports
│       ├── service/
│       │   ├── auth.go                # Authentication & token rotation service
│       │   ├── job.go                 # Job business logic service
│       │   └── user.go                # User registration & verification service
│       └── util/
│           ├── email.go               # SMTP email sender
│           ├── helper.go              # JSON serialize/deserialize helper
│           ├── jwt.go                 # Token parsing and generation
│           └── password.go            # Bcrypt and SHA-256 hashing
├── .env.example                       # Template for environment configuration
├── .gitignore
├── Taskfile.yml                       # Task runner recipes
├── docker-compose.yml                 # PostgreSQL container definition
├── go.mod
└── go.sum
```

---

## API Endpoints

Base URL: `/api/v1`

### Public Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Healthcheck endpoint |
| `POST` | `/register` | Register a new user account |
| `GET` | `/confirm-email` | Confirm user email address via token |
| `POST` | `/login` | Authenticate user and issue tokens |
| `POST` | `/logout` | Invalidate active session and clear cookies |
| `POST` | `/refresh` | Rotate access token using refresh token |
| `GET` | `/validate-token` | Validate active access token claims |
| `POST` | `/request-password-reset` | Request password reset email (enumeration-safe) |
| `POST` | `/reset-password` | Set new password using reset token |

### Protected Endpoints (User & Admin)

| Method | Endpoint | Auth Required | Minimum Role | Description |
|---|---|---|---|---|
| `GET` | `/jobs` | Yes | `User` (2001) | List job listings |
| `GET` | `/jobs/:id` | Yes | `User` (2001) | Retrieve job details by ID |

### Protected Endpoints (Admin Only)

| Method | Endpoint | Auth Required | Minimum Role | Description |
|---|---|---|---|---|
| `GET` | `/users` | Yes | `Admin` (5150) | List registered users |
| `POST` | `/jobs` | Yes | `Admin` (5150) | Create a new job listing |
| `DELETE` | `/jobs/:id` | Yes | `Admin` (5150) | Delete a job listing by ID |

---

## Getting Started

### Prerequisites

- Go 1.24+ installed
- Docker and Docker Compose
- [Task](https://taskfile.dev) (optional, standard `go` and `docker` commands work as well)

### 1. Clone and Configure Environment

Copy the example environment configuration:

```bash
cp .env.example .env
```

Adjust variables in `.env` as needed:

```env
APP_NAME=go-gin-nextjs-auth
APP_ENV=development

HTTP_HOST=127.0.0.1
HTTP_PORT=8080
HTTP_ALLOWED_ORIGINS=http://127.0.0.1:3000

DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_db_password
DB_NAME=go_auth

ACCESS_TOKEN_SECRET=your_access_token_secret_key
REFRESH_TOKEN_SECRET=your_refresh_token_secret_key
ACCESS_TOKEN_DURATION=15     # in minutes
REFRESH_TOKEN_DURATION=7     # in days

EMAIL_TOKEN_SECRET=your_email_token_secret_key
EMAIL_TOKEN_DURATION=15      # in minutes
```

### 2. Start PostgreSQL Database

Using Docker Compose:

```bash
docker compose up -d
```

Or via Task:

```bash
task compose:up
```

### 3. Run the Application

Start the API server:

```bash
go run ./cmd/http
```

For live reloading in development (requires [Air](https://github.com/air-verse/air)):

```bash
task dev
```

The server starts on `http://127.0.0.1:8080`.

---

## Testing & Quality Assurance

### Concurrency and Race Detector

Run test suites with race detection enabled:

```bash
go test -race -v ./...
```

Compile binary with race checks for staging verification:

```bash
go build -race -o server ./cmd/http
```

### Vulnerability Scanning

Scan project dependencies and symbols against known Go vulnerability databases:

```bash
govulncheck ./...
```
