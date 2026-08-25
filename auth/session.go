package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

// ErrSessionNotFound é retornado pelo Store quando o ID não existe ou
// já expirou.
var ErrSessionNotFound = errors.New("auth: sessão não encontrada")

// Session representa os dados de uma sessão autenticada. Data é livre
// pra guardar o que precisar (ex: "user_id", "role").
type Session struct {
	ID        string
	Data      map[string]interface{}
	ExpiresAt time.Time
}

// Expired verifica se a sessão já passou do prazo de validade.
func (s *Session) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Store é a interface de persistência de sessões. MemoryStore
// (abaixo) cobre o caso comum de single-instance; pra multi-instância
// (vários processos atrás de um load balancer), implemente Store sobre
// Redis/DB e passe pro NewSessionManager — a assinatura não muda.
type Store interface {
	Get(id string) (*Session, error)
	Save(session *Session) error
	Delete(id string) error
}

// MemoryStore é uma implementação de Store em memória, protegida por
// mutex. Simples e rápida, mas não sobrevive a restart nem escala pra
// múltiplas instâncias do processo.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStore cria um MemoryStore vazio.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]*Session)}
}

func (m *MemoryStore) Get(id string) (*Session, error) {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrSessionNotFound
	}
	if session.Expired() {
		// Limpeza preguiçosa: remove no primeiro acesso após expirar.
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (m *MemoryStore) Save(session *Session) error {
	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) Delete(id string) error {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
	return nil
}

// generateSessionID gera um ID aleatório de 32 bytes (256 bits) via
// crypto/rand, codificado em base64 URL-safe — imprevisível o
// suficiente pra usar como cookie de sessão.
func generateSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// SessionManager amarra um Store a um cookie HTTP, cuidando de gerar
// ID, setar/ler/expirar o cookie e persistir no Store.
type SessionManager struct {
	Store      Store
	CookieName string
	TTL        time.Duration

	// Atributos do cookie. Secure=true exige HTTPS — deixa false só em
	// desenvolvimento local.
	Secure   bool
	HTTPOnly bool
	Path     string
}

// NewSessionManager cria um manager com defaults sensatos: cookie
// "session_id", TTL de 24h, HttpOnly=true (protege contra XSS lendo o
// cookie via JS), Secure=false (ajuste pra true em produção com HTTPS).
func NewSessionManager(store Store) *SessionManager {
	return &SessionManager{
		Store:      store,
		CookieName: "session_id",
		TTL:        24 * time.Hour,
		HTTPOnly:   true,
		Path:       "/",
	}
}