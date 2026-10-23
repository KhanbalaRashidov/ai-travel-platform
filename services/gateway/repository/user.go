// Package repository is the only layer that talks to the database.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/models"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// UserRepository provides tenant-scoped access to the users table.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository constructs a UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user and returns it with the generated id/created_at.
func (r *UserRepository) Create(ctx context.Context, u models.User) (models.User, error) {
	const q = `
		INSERT INTO users (tenant_id, email, password_hash, full_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, q, u.TenantID, u.Email, u.PasswordHash, u.FullName).
		Scan(&u.ID, &u.CreatedAt)
	return u, err
}

// GetByEmail returns a user by email within a tenant.
func (r *UserRepository) GetByEmail(ctx context.Context, tenantID, email string) (models.User, error) {
	const q = `
		SELECT id, tenant_id, email, password_hash, full_name, created_at
		FROM users
		WHERE tenant_id = $1 AND email = $2`
	var u models.User
	err := r.pool.QueryRow(ctx, q, tenantID, email).
		Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

// GetByID returns a user by id within a tenant.
func (r *UserRepository) GetByID(ctx context.Context, tenantID, id string) (models.User, error) {
	const q = `
		SELECT id, tenant_id, email, password_hash, full_name, created_at
		FROM users
		WHERE tenant_id = $1 AND id = $2`
	var u models.User
	err := r.pool.QueryRow(ctx, q, tenantID, id).
		Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}
