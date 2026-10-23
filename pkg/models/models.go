// Package models holds shared domain entities.
package models

import "time"

// Tenant represents an isolated customer workspace. The platform is
// multi-tenant-ready from day one; a single "default" tenant is used for now.
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// User is an end user, always scoped to a tenant.
type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
}
