package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

/**
 * Erros estáticos retornados no fluxo de persistência de sessão.
 */
var ErrSessionNotFound = errors.New("auth: sessão não encontrada")

/**
 * Session armazena a estrutura de dados e vigência da sessão ativa.
 */
type Session struct {
	ID        string
	Data      map[string]interface{}
	ExpiresAt time.Time
}

/**
 * Expired checa se a sessão ultrapassou a janela temporal válida.
 */
func (s *Session) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

/**
 * Interface para desacoplamento da camada de armazenamento de sessões.
 */
type Store interface {
	Get(id string) (*Session, error)
	Save(session *Session) error
	Delete(id string) error
}

/**
 * MemoryStore provê armazenamento em memória thread-safe com limpeza automática de expirados.
 */
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	stopGC   chan struct{}
}

/**
 * NewMemoryStore instancia o armazenamento em memória e inicia a goroutine de varredura periódica.
 *
 * @param cleanupInterval Frequência em que a varredura por sessões expiradas é realizada.
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
 * Close encerra com segurança os recursos da store em memória (interrompe o GC).
 */
func (m *MemoryStore) Close() {
	if m.stopGC != nil {
		close(m.stopGC)
	}
}

/**
 * Get busca uma sessão ativa por ID. Retorna cópia isolada para evitar data races em mutações.
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

	// Deep copy de session.Data para garantir isolamento de memória
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
 * Save persiste ou atualiza os dados da sessão no repositório.
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
 * Delete remove explicitamente uma sessão da memória pelo seu ID.
 */
func (m *MemoryStore) Delete(id string) error {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
	return nil
}

/**
 * startGC executa varreduras periódicas para remoção de chaves inativas sem bloquear leituras massivas.
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
 * generateSessionID cria uma sequência aleatória criptograficamente segura de 32 bytes em Base64 URL-safe.
 */
func generateSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

/**
 * SessionManager coordena a integração entre a Store e os cookies da camada HTTP.
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
 * NewSessionManager constrói o gerenciador configurado com parâmetros padrão seguros.
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