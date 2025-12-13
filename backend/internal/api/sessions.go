package api

import (
	"sync"
	"time"
)

type Session struct {
	UserID string
	Role   string
	OrgID  *string
	Expires time.Time
}

type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*Session),
	}
	// Cleanup expired sessions every hour
	go store.cleanup()
	return store
}

func (s *SessionStore) Set(token string, userID string, role string, orgID *string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = &Session{
		UserID:  userID,
		Role:    role,
		OrgID:   orgID,
		Expires: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}
}

func (s *SessionStore) Get(token string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[token]
	if !ok {
		return nil, false
	}
	if time.Now().After(session.Expires) {
		delete(s.sessions, token)
		return nil, false
	}
	return session, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, session := range s.sessions {
			if now.After(session.Expires) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}
