package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	kite "github.com/claudiovictors/kite/core"
)

/**
 * contextKey is a private key type used to avoid collisions in the HTTP context.
 */
type contextKey string

const (
	claimsContextKey  contextKey = "auth_claims"
	sessionContextKey contextKey = "auth_session"
)

/**
 * withContext clones the HTTP request, injecting a new context that
 * carries the authentication data.
 */
func withContext(req kite.Request, ctx context.Context) kite.Request {
	req.Request = req.Request.WithContext(ctx)
	return req
}

// --- JWT Middleware ----------------------------------------------------

/**
 * RequireJWT intercepts requests, requiring a valid Bearer token in the
 * Authorization header.
 *
 * @param secret Symmetric key used to validate the JWT signature.
 * @return A MiddlewareFunc compatible with the Kite ecosystem.
 */
func RequireJWT(secret []byte) kite.MiddlewareFunc {
	return func(next kite.HandlerFunc) kite.HandlerFunc {
		return func(req kite.Request, res kite.Response) error {
			header := req.Header("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				return unauthorized(res, "missing token (expected: Authorization: Bearer <token>)")
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
 * ClaimsFromContext retrieves the decoded claims from the request context.
 */
func ClaimsFromContext(ctx context.Context) (MapClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(MapClaims)
	return claims, ok
}

// --- Session Middleware ------------------------------------------------

/**
 * RequireSession validates the existence and the expiration window of a
 * session stored via cookie.
 *
 * @param manager Pointer to the active session manager.
 * @return A MiddlewareFunc for session-based authentication.
 */
func RequireSession(manager *SessionManager) kite.MiddlewareFunc {
	return func(next kite.HandlerFunc) kite.HandlerFunc {
		return func(req kite.Request, res kite.Response) error {
			cookieValue, ok := req.Cookie(manager.CookieName)
			if !ok {
				return unauthorized(res, "missing session")
			}

			session, err := manager.Store.Get(cookieValue)
			if err != nil || session == nil {
				return unauthorized(res, "invalid session")
			}

			if time.Now().After(session.ExpiresAt) {
				_ = manager.Store.Delete(cookieValue)
				return unauthorized(res, "session expired")
			}

			ctx := context.WithValue(req.Ctx(), sessionContextKey, session)
			return next(withContext(req, ctx), res)
		}
	}
}

/**
 * SessionFromContext retrieves the active Session struct from the context.
 */
func SessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	return session, ok
}

// --- Authentication Helpers ---------------------------------------------

/**
 * Login generates a unique identifier, persists the session in the Store,
 * and writes the HTTP cookie onto the response.
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
 * Logout invalidates the session in the Store and clears the client's
 * cookie immediately.
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
 * unauthorized standardizes the 401 JSON response using Kite's error format.
 */
func unauthorized(res kite.Response, reason string) error {
	return res.Status(http.StatusUnauthorized).WithJson(kite.ErrorResponse{
		Error:  "unauthorized: " + reason,
		Status: http.StatusUnauthorized,
	})
}