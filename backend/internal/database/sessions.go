package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sherlock/service/internal/types"
)

// CreateSession stores a session in the database
func (db *DB) CreateSession(token, userID, role string, orgID *string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (token, user_id, role, org_id, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			role = EXCLUDED.role,
			org_id = EXCLUDED.org_id,
			expires_at = EXCLUDED.expires_at
	`
	_, err := db.conn.Exec(query, token, userID, role, orgID, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSession retrieves a session from the database
func (db *DB) GetSession(token string) (*types.Session, error) {
	query := `
		SELECT token, user_id, role, org_id, expires_at
		FROM sessions
		WHERE token = $1 AND expires_at > NOW()
	`

	session := &types.Session{}
	var orgID sql.NullString
	err := db.conn.QueryRow(query, token).Scan(
		&session.Token,
		&session.UserID,
		&session.Role,
		&orgID,
		&session.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if orgID.Valid {
		session.OrgID = &orgID.String
	}

	return session, nil
}

// DeleteSession removes a session from the database
func (db *DB) DeleteSession(token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := db.conn.Exec(query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (db *DB) CleanupExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`
	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}

