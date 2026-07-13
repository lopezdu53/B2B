package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/gosom/google-maps-scraper/admin"
)

// CreateTenant inserts a new client company.
func (s *store) CreateTenant(ctx context.Context, name string) (*admin.Tenant, error) {
	var t admin.Tenant
	if err := s.db.QueryRow(ctx,
		`INSERT INTO tenants (name) VALUES ($1) RETURNING id, name, active, created_at`,
		name,
	).Scan(&t.ID, &t.Name, &t.Active, &t.CreatedAt); err != nil {
		return nil, err
	}

	return &t, nil
}

// ListTenants returns all client companies ordered by name.
func (s *store) ListTenants(ctx context.Context) ([]admin.Tenant, error) {
	rows, err := s.db.Query(ctx, `SELECT id, name, active, created_at FROM tenants ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []admin.Tenant

	for rows.Next() {
		var t admin.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Active, &t.CreatedAt); err != nil {
			return nil, err
		}

		out = append(out, t)
	}

	return out, rows.Err()
}

// GetTenant returns a single tenant by id.
func (s *store) GetTenant(ctx context.Context, id int64) (*admin.Tenant, error) {
	var t admin.Tenant
	if err := s.db.QueryRow(ctx,
		`SELECT id, name, active, created_at FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Active, &t.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, admin.ErrResourceNotFound
		}

		return nil, err
	}

	return &t, nil
}

// TenantCounts returns the number of admin users and advisor records for a
// tenant (best-effort; errors yield zeros).
func (s *store) TenantCounts(ctx context.Context, tenantID int64) (admins, advisors int) {
	_ = s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND role = 'admin'`, tenantID).Scan(&admins)
	_ = s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM b2b_advisors WHERE tenant_id = $1`, tenantID).Scan(&advisors)

	return admins, advisors
}

// LinkUserAdvisor links a login user to their advisor record.
func (s *store) LinkUserAdvisor(ctx context.Context, userID int, advisorID int64) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET advisor_id = $1 WHERE id = $2`, advisorID, userID)

	return err
}

// CreateTenantUser creates a user (admin or advisor) scoped to a tenant.
func (s *store) CreateTenantUser(ctx context.Context, username, password, role string, tenantID int64) (*admin.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user admin.User
	err = s.db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, role, tenant_id) VALUES ($1, $2, $3, $4)
		 RETURNING id, username, password_hash, role, tenant_id, created_at, updated_at`,
		username, string(hash), role, tenantID,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.TenantID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err.Error() == `ERROR: duplicate key value violates unique constraint "users_username_key" (SQLSTATE 23505)` {
			return nil, admin.ErrUserExists
		}

		return nil, err
	}

	return &user, nil
}
