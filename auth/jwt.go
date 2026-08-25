// Package auth implementa autenticação JWT e sessão baseada em cookie,
// usando apenas a stdlib do Go (crypto/hmac, crypto/sha256, crypto/rand,
// encoding/base64, encoding/json) — sem dependências externas.
//
// JWT (jwt.go): suporta apenas HS256 por enquanto (assinatura simétrica
// com um secret compartilhado). Suporte a RS256/ES256 fica pra v2, caso
// precise de chave pública/privada (ex: validar tokens de terceiros).
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	// ErrInvalidToken é retornado quando o token não tem o formato
	// esperado (3 partes separadas por ".") ou o JSON é inválido.
	ErrInvalidToken = errors.New("auth: token JWT inválido")

	// ErrInvalidSignature é retornado quando a assinatura não bate com
	// o secret informado (token adulterado ou secret errado).
	ErrInvalidSignature = errors.New("auth: assinatura do token inválida")

	// ErrTokenExpired é retornado quando o claim "exp" já passou.
	ErrTokenExpired = errors.New("auth: token expirado")
)

// jwtHeader é fixo por enquanto, já que só HS256 é suportado.
const jwtHeaderHS256 = `{"alg":"HS256","typ":"JWT"}`

// MapClaims é o conjunto de claims do token, no estilo do
// jwt-go/golang-jwt: um map livre, mas com helpers pros claims
// "reservados" (exp, iat, sub) mais comuns.
type MapClaims map[string]interface{}

// NewClaims monta um MapClaims já com "sub" (subject, ex: ID do
// usuário), "iat" (issued at) e "exp" (expiração, calculada a partir
// de ttl), mais qualquer claim extra que o usuário queira embutir.
//
// Exemplo:
//
//	claims := auth.NewClaims(user.ID, 24*time.Hour, auth.MapClaims{
//	    "role": "admin",
//	})
//	token, _ := auth.Sign(claims, secret)
func NewClaims(subject interface{}, ttl time.Duration, extra MapClaims) MapClaims {
	now := time.Now()
	claims := MapClaims{
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	for k, v := range extra {
		claims[k] = v
	}
	return claims
}

// base64URLEncode/Decode usam RawURLEncoding (sem "=" de padding),
// exatamente como o spec do JWT (RFC 7519) exige.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// sign calcula o HMAC-SHA256 de "header.payload" usando o secret.
func sign(headerAndPayload string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(headerAndPayload))
	return mac.Sum(nil)
}

// Sign gera um token JWT assinado (HS256) a partir dos claims e do
// secret informados. O secret deve ser mantido só no servidor (nunca
// no frontend) — recomenda-se pelo menos 32 bytes aleatórios.
//
//	token, err := auth.Sign(auth.NewClaims(user.ID, time.Hour, nil), secret)
func Sign(claims MapClaims, secret []byte) (string, error) {
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64URLEncode([]byte(jwtHeaderHS256))
	payloadB64 := base64URLEncode(payloadBytes)

	unsigned := headerB64 + "." + payloadB64
	signature := sign(unsigned, secret)

	return unsigned + "." + base64URLEncode(signature), nil
}

// Verify decodifica e valida um token JWT: checa o formato, recalcula
// a assinatura (comparação em tempo constante via hmac.Equal, contra
// timing attacks) e valida o claim "exp" quando presente.
//
//	claims, err := auth.Verify(tokenString, secret)
//	if errors.Is(err, auth.ErrTokenExpired) { ... }
func Verify(tokenString string, secret []byte) (MapClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	headerB64, payloadB64, signatureB64 := parts[0], parts[1], parts[2]

	expectedSig := sign(headerB64+"."+payloadB64, secret)
	actualSig, err := base64URLDecode(signatureB64)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, ErrInvalidSignature
	}

	payloadBytes, err := base64URLDecode(payloadB64)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims MapClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if exp, ok := claims["exp"]; ok {
		expUnix, ok := toInt64(exp)
		if ok && time.Now().Unix() > expUnix {
			return nil, ErrTokenExpired
		}
	}

	return claims, nil
}

// toInt64 normaliza o claim "exp", que volta como float64 depois do
// json.Unmarshal em interface{} (comportamento padrão do encoding/json
// pra números).
func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	default:
		return 0, false
	}
}