package api

import (
	"time"

	"github.com/sherlock/service/internal/database"
)

type SessionStore struct {
	db *database.DB
}

func NewSessionStore(db *database.DB) *SessionStore {
	store := &SessionStore{db: db}
	// Cleanup expired sessions every hour
	go store.cleanup()
	return store
}

func (s *SessionStore) Set(token string, userID string, role string, orgID *string) {
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	_ = s.db.CreateSession(token, userID, role, orgID, expiresAt)
}

func (s *SessionStore) Get(token string) (*Session, bool) {
	session, err := s.db.GetSession(token)
	if err != nil || session == nil {
		return nil, false
	}
	return &Session{
		UserID:  session.UserID,
		Role:    session.Role,
		OrgID:   session.OrgID,
		Expires: session.ExpiresAt,
	}, true
}

func (s *SessionStore) Delete(token string) {
	_ = s.db.DeleteSession(token)
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		_ = s.db.CleanupExpiredSessions()
	}
}

type Session struct {
	UserID  string
	Role    string
	OrgID   *string
	Expires time.Time
}
