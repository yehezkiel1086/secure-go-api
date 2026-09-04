package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres/repository"
	sqlc "github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres/sqlc"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

type mockDBTX struct {
	execFn     func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
	queryFn    func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, query string, args ...interface{}) pgx.Row
}

func (m *mockDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, query, args...)
	}
	return pgconn.NewCommandTag(""), nil
}

func (m *mockDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, query, args...)
	}
	return mockRow{err: pgx.ErrNoRows}
}

type mockRow struct {
	scanFn func(dest ...interface{}) error
	err    error
}

func (r mockRow) Scan(dest ...interface{}) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return r.err
}

func TestUserRepository_ImplementsPort(t *testing.T) {
	mockDB := &mockDBTX{}
	repo := repository.NewUserRepositoryWithDB(mockDB)

	var _ port.UserRepository = repo
	if repo == nil {
		t.Fatal("expected repo to not be nil")
	}
}

func TestUserRepository_NewUserRepository(t *testing.T) {
	mockDB := &mockDBTX{}
	queries := sqlc.New(mockDB)
	repo := repository.NewUserRepository(queries)

	if repo == nil {
		t.Fatal("expected repo to not be nil")
	}
}

func TestUserRepository_GetUserByID_NotFound(t *testing.T) {
	mockDB := &mockDBTX{
		queryRowFn: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return mockRow{err: pgx.ErrNoRows}
		},
	}

	repo := repository.NewUserRepositoryWithDB(mockDB)
	user, err := repo.GetUserByID(context.Background(), pgtype.UUID{})

	if user != nil {
		t.Fatalf("expected nil user, got %v", user)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepository_GetUserByEmail_NotFound(t *testing.T) {
	mockDB := &mockDBTX{
		queryRowFn: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return mockRow{err: pgx.ErrNoRows}
		},
	}

	repo := repository.NewUserRepositoryWithDB(mockDB)
	user, err := repo.GetUserByEmail(context.Background(), "unknown@example.com")

	if user != nil {
		t.Fatalf("expected nil user, got %v", user)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepository_UpdateUserName_NotFound(t *testing.T) {
	mockDB := &mockDBTX{
		queryRowFn: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return mockRow{err: pgx.ErrNoRows}
		},
	}

	repo := repository.NewUserRepositoryWithDB(mockDB)
	user, err := repo.UpdateUserName(context.Background(), pgtype.UUID{}, "New Name")

	if user != nil {
		t.Fatalf("expected nil user, got %v", user)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
