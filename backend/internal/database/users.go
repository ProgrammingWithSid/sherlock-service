package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sherlock/service/internal/types"
	"golang.org/x/crypto/bcrypt"
)

func (db *DB) CreateUser(email, password, name string, role types.Role, orgID *string) (*types.User, error) {
	id := uuid.New().String()
	now := time.Now()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users (id, email, password_hash, name, role, org_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, email, password_hash, name, role, org_id, is_active, created_at, updated_at
	`

	user := &types.User{}
	var orgIDPtr *string
	err = db.conn.QueryRow(query, id, email, string(hashedPassword), name, role, orgID, true, now, now).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &orgIDPtr, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.OrgID = orgIDPtr
	return user, nil
}

func (db *DB) GetUserByEmail(email string) (*types.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, org_id, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &types.User{}
	var orgIDPtr *string
	err := db.conn.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &orgIDPtr, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.OrgID = orgIDPtr
	return user, nil
}

func (db *DB) GetUserByID(id string) (*types.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, org_id, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &types.User{}
	var orgIDPtr *string
	err := db.conn.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &orgIDPtr, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.OrgID = orgIDPtr
	return user, nil
}

func (db *DB) ListUsers(orgID *string) ([]*types.User, error) {
	var query string
	var rows *sql.Rows
	var err error

	if orgID != nil {
		query = `
			SELECT id, email, password_hash, name, role, org_id, is_active, created_at, updated_at
			FROM users
			WHERE org_id = $1
			ORDER BY created_at DESC
		`
		rows, err = db.conn.Query(query, *orgID)
	} else {
		query = `
			SELECT id, email, password_hash, name, role, org_id, is_active, created_at, updated_at
			FROM users
			ORDER BY created_at DESC
		`
		rows, err = db.conn.Query(query)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*types.User
	for rows.Next() {
		user := &types.User{}
		var orgIDPtr *string
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &orgIDPtr, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		user.OrgID = orgIDPtr
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

func (db *DB) UpdateUser(userID string, name *string, isActive *bool) error {
	now := time.Now()

	if name != nil && isActive != nil {
		query := `UPDATE users SET name = $1, is_active = $2, updated_at = $3 WHERE id = $4`
		_, err := db.conn.Exec(query, *name, *isActive, now, userID)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	} else if name != nil {
		query := `UPDATE users SET name = $1, updated_at = $2 WHERE id = $3`
		_, err := db.conn.Exec(query, *name, now, userID)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	} else if isActive != nil {
		query := `UPDATE users SET is_active = $1, updated_at = $2 WHERE id = $3`
		_, err := db.conn.Exec(query, *isActive, now, userID)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	}

	return nil
}

func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func (db *DB) UpdateUserRole(userID string, role types.Role) error {
	now := time.Now()
	query := `UPDATE users SET role = $1, updated_at = $2 WHERE id = $3`
	_, err := db.conn.Exec(query, role, now, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}

// ListAllUsers returns all users (for super admin)
func (db *DB) ListAllUsers() ([]*types.User, error) {
	return db.ListUsers(nil)
}
