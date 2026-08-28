package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	kite "github.com/claudiovictors/kite/core"
)

/**
 * Definição de chave privada para evitar colisões no contexto HTTP.
 */
type contextKey string

const (
	claimsContextKey  contextKey = "auth_claims"
	sessionContextKey contextKey = "auth_session"
)

/**
 * withContext clona a requisição HTTP injetando um novo contexto contendo os dados de autenticação.
 */
func withContext(req kite.Request, ctx context.Context) kite.Request {
	req.Request = req.Request.WithContext(ctx)
	return req
}

// --- JWT Middleware ----------------------------------------------------

/**
 * RequireJWT intercepta requisições exigindo um token Bearer válido no header Authorization.
 *
 * @param secret Chave simétrica utilizada para validar a assinatura do JWT.
 * @return MiddlewareFunc compatível com o ecossistema Kite.
 */
func RequireJWT(secret []byte) kite.MiddlewareFunc {
	return func(next kite.HandlerFunc) kite.HandlerFunc {
		return func(req kite.Request, res kite.Response) error {
			header := req.Header("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				return unauthorized(res, "token ausente (esperado: Authorization: Bearer <token>)")
			}
			tokenString := strings.TrimPrefix(header, "Bearer ")

			claims, err := Verify(tokenString, secret)
			if err != nil {
				return unauthorized(res, err.Error())
			}

			ctx := context.WithValue(req.Ctx(), claimsContextKey, claims)
			return next(withContext(req, ctx), res)
		}
	}
}

/**
 * ClaimsFromContext recupera os claims decodificados do contexto da requisição.
 */
func ClaimsFromContext(ctx context.Context) (MapClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(MapClaims)
	return claims, ok
}

// --- Session Middleware ------------------------------------------------

/**
 * RequireSession valida a existência e a vigência temporal de uma sessão armazenada em cookie.
 *
 * @param manager Ponteiro para o gerenciador de sessões ativo.
 * @return MiddlewareFunc de autenticação por sessão.
 */
func RequireSession(manager *SessionManager) kite.MiddlewareFunc {
	return func(next kite.HandlerFunc) kite.HandlerFunc {
		return func(req kite.Request, res kite.Response) error {
			cookieValue, ok := req.Cookie(manager.CookieName)
			if !ok {
				return unauthorized(res, "sessão ausente")
			}

			session, err := manager.Store.Get(cookieValue)
			if err != nil || session == nil {
				return unauthorized(res, "sessão inválida")
			}

			if time.Now().After(session.ExpiresAt) {
				_ = manager.Store.Delete(cookieValue)
				return unauthorized(res, "sessão expirada")
			}

			ctx := context.WithValue(req.Ctx(), sessionContextKey, session)
			return next(withContext(req, ctx), res)
		}
	}
}

/**
 * SessionFromContext recupera a estrutura de Session ativa contida no contexto.
 */
func SessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	return session, ok
}

// --- Helpers de Autenticação -------------------------------------------

/**
 * Login gera um identificador único, persiste a sessão na Store e grava o cookie HTTP na resposta.
 */
func (m *SessionManager) Login(res kite.Response, data map[string]interface{}) error {
	id, err := generateSessionID()
	if err != nil {
		return err
	}

	session := &Session{
		ID:        id,
		Data:      data,
		ExpiresAt: time.Now().Add(m.TTL),
	}
	if err := m.Store.Save(session); err != nil {
		return err
	}

	res.Cookie(&http.Cookie{
		Name:     m.CookieName,
		Value:    id,
		Path:     m.Path,
		Expires:  session.ExpiresAt,
		MaxAge:   int(m.TTL.Seconds()),
		HttpOnly: m.HTTPOnly,
		Secure:   m.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

/**
 * Logout inativa a sessão na Store e invalida o cookie do cliente imediatamente.
 */
func (m *SessionManager) Logout(req kite.Request, res kite.Response) error {
	if cookieValue, ok := req.Cookie(m.CookieName); ok {
		_ = m.Store.Delete(cookieValue)
	}

	res.Cookie(&http.Cookie{
		Name:     m.CookieName,
		Value:    "",
		Path:     m.Path,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: m.HTTPOnly,
		Secure:   m.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

/**
 * unauthorized padroniza a resposta HTTP 401 JSON utilizando o formato estipulado pelo Kite.
 */
func unauthorized(res kite.Response, reason string) error {
	return res.Status(http.StatusUnauthorized).WithJson(kite.ErrorResponse{
		Error:  "não autorizado: " + reason,
		Status: http.StatusUnauthorized,
	})
}