package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	kite "github.com/claudiovictors/kite/core"
)

// contextKey evita colisão com outras chaves de context.Context vindas
// de outros pacotes/middlewares.
type contextKey string

const (
	claimsContextKey  contextKey = "auth_claims"
	sessionContextKey contextKey = "auth_session"
)

// withContext devolve uma cópia de req com o *http.Request substituído
// por um que carrega o ctx informado — é assim que dados passam de um
// middleware pro próximo/handler, já que kite.Request embute
// *http.Request por valor de ponteiro mas Request em si é passado por
// valor entre as funções.
func withContext(req kite.Request, ctx context.Context) kite.Request {
	req.Request = req.Request.WithContext(ctx)
	return req
}

// --- JWT --------------------------------------------------------------

// RequireJWT retorna um middleware que exige um Bearer token válido no
// header Authorization. Em caso de falta/token inválido/expirado,
// responde 401 em JSON e interrompe a cadeia (o handler não é chamado).
// Em caso de sucesso, os claims ficam disponíveis via ClaimsFromContext.
//
//	protected := app.Group("/api")
//	protected.Use(auth.RequireJWT(secret))
//	protected.Get("/me", meHandler)
//
//	func meHandler(req kite.Request, res kite.Response) error {
//	    claims, _ := auth.ClaimsFromContext(req.Ctx())
//	    return res.Json(map[string]interface{}{"user_id": claims["sub"]})
//	}
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

// ClaimsFromContext extrai os claims do JWT colocados no contexto por
// RequireJWT. O segundo retorno é false se não houver claims (ex: rota
// sem RequireJWT, ou chamado fora de um handler protegido).
func ClaimsFromContext(ctx context.Context) (MapClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(MapClaims)
	return claims, ok
}

// --- Session ------------------------------------------------------------

// RequireSession retorna um middleware que exige uma sessão válida via
// cookie. Se o cookie não existir, estiver expirado, ou não existir no
// Store, responde 401 em JSON. Em caso de sucesso, a sessão fica
// disponível via SessionFromContext.
//
//	protected := app.Group("/dashboard")
//	protected.Use(auth.RequireSession(sessionManager))
func RequireSession(manager *SessionManager) kite.MiddlewareFunc {
	return func(next kite.HandlerFunc) kite.HandlerFunc {
		return func(req kite.Request, res kite.Response) error {
			cookieValue, ok := req.Cookie(manager.CookieName)
			if !ok {
				return unauthorized(res, "sessão ausente")
			}

			session, err := manager.Store.Get(cookieValue)
			if err != nil {
				return unauthorized(res, "sessão inválida ou expirada")
			}

			ctx := context.WithValue(req.Ctx(), sessionContextKey, session)
			return next(withContext(req, ctx), res)
		}
	}
}

// SessionFromContext extrai a *Session colocada no contexto por
// RequireSession.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	return session, ok
}

// --- Helpers de login/logout (fora do middleware, chamados direto no handler) ---

// Login cria uma nova sessão com os dados informados, salva no Store e
// seta o cookie na resposta. Uso típico num handler de POST /login:
//
//	func loginHandler(req kite.Request, res kite.Response) error {
//	    // ... validar credenciais ...
//	    if err := manager.Login(res, map[string]interface{}{"user_id": user.ID}); err != nil {
//	        return err
//	    }
//	    return res.Json(map[string]string{"status": "ok"})
//	}
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
		HttpOnly: m.HTTPOnly,
		Secure:   m.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// Logout remove a sessão do Store (se o cookie existir) e expira o
// cookie no navegador.
func (m *SessionManager) Logout(req kite.Request, res kite.Response) error {
	if cookieValue, ok := req.Cookie(m.CookieName); ok {
		_ = m.Store.Delete(cookieValue)
	}

	res.Cookie(&http.Cookie{
		Name:     m.CookieName,
		Value:    "",
		Path:     m.Path,
		Expires:  time.Unix(0, 0), // no passado -> navegador remove o cookie
		HttpOnly: m.HTTPOnly,
		Secure:   m.Secure,
	})
	return nil
}

// unauthorized é o formato padrão de resposta 401 em JSON usado pelos
// dois middlewares (mantém consistência com kite.ErrorResponse do core).
func unauthorized(res kite.Response, reason string) error {
	return res.Status(http.StatusUnauthorized).WithJson(kite.ErrorResponse{
		Error:  "não autorizado: " + reason,
		Status: http.StatusUnauthorized,
	})
}