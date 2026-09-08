package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

/**
 * Static errors returned by the session persistence flow.
 */
var ErrSessionNotFound = errors.New("auth: session not found")

/**
 * Session holds the data and expiration window of an active session.
 */
type Session struct {
	ID        string
	Data      map[string]interface{}
	ExpiresAt time.Time
}

/**
 * Expired reports whether the session has passed its valid time window.
 */
func (s *Session) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

/**
 * Store decouples the session storage layer from its consumers.
 */
type Store interface {
	Get(id string) (*Session, error)
	Save(session *Session) error
	Delete(id string) error
}

/**
 * MemoryStore provides thread-safe in-memory storage with automatic
 * cleanup of expired sessions.
 */
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	stopGC   chan struct{}
}

/**
 * NewMemoryStore creates the in-memory store and starts the periodic
 * garbage-collection goroutine.
 *
 * @param cleanupInterval How often the store scans for expired sessions.
 */
func NewMemoryStore(cleanupInterval time.Duration) *MemoryStore {
	store := &MemoryStore{
		sessions: make(map[string]*Session),
		stopGC:   make(chan struct{}),
	}

	if cleanupInterval > 0 {
		go store.startGC(cleanupInterval)
	}

	return store
}

/**
 * Close safely releases the store's resources (stops the GC goroutine).
 */
func (m *MemoryStore) Close() {
	if m.stopGC != nil {
		close(m.stopGC)
	}
}

/**
 * Get looks up an active session by ID. Returns an isolated copy to avoid
 * data races on mutation.
 */
func (m *MemoryStore) Get(id string) (*Session, error) {
	m.mu.RLock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.RUnlock()
		return nil, ErrSessionNotFound
	}

	if session.Expired() {
		m.mu.RUnlock()
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		return nil, ErrSessionNotFound
	}

	// Deep copy of session.Data to guarantee memory isolation.
	dataCopy := make(map[string]interface{}, len(session.Data))
	for k, v := range session.Data {
		dataCopy[k] = v
	}

	cp := &Session{
		ID:        session.ID,
		Data:      dataCopy,
		ExpiresAt: session.ExpiresAt,
	}
	m.mu.RUnlock()

	return cp, nil
}

/**
 * Save persists or updates the session data in the repository.
 */
func (m *MemoryStore) Save(session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dataCopy := make(map[string]interface{}, len(session.Data))
	for k, v := range session.Data {
		dataCopy[k] = v
	}

	m.sessions[session.ID] = &Session{
		ID:        session.ID,
		Data:      dataCopy,
		ExpiresAt: session.ExpiresAt,
	}
	return nil
}

/**
 * Delete explicitly removes a session from memory by its ID.
 */
func (m *MemoryStore) Delete(id string) error {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
	return nil
}

/**
 * startGC runs periodic sweeps to remove inactive keys without blocking
 * concurrent reads.
 */
func (m *MemoryStore) startGC(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for id, sess := range m.sessions {
				if now.After(sess.ExpiresAt) {
					delete(m.sessions, id)
				}
			}
			m.mu.Unlock()
		case <-m.stopGC:
			return
		}
	}
}

/**
 * generateSessionID creates a cryptographically secure 32-byte random
 * sequence, encoded as Base64 URL-safe.
 */
func generateSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

/**
 * SessionManager coordinates the integration between the Store and the
 * HTTP cookie layer.
 */
type SessionManager struct {
	Store      Store
	CookieName string
	TTL        time.Duration
	Secure     bool
	HTTPOnly   bool
	Path       string
}

/**
 * NewSessionManager builds a manager configured with safe defaults.
 */
func NewSessionManager(store Store) *SessionManager {
	return &SessionManager{
		Store:      store,
		CookieName: "session_id",
		TTL:        24 * time.Hour,
		HTTPOnly:   true,
		Path:       "/",
	}
}