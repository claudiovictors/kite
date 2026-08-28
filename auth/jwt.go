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

/**
 * Definição dos erros padrão retornados durante o ciclo de vida do JWT.
 */
var (
	ErrInvalidToken     = errors.New("auth: token JWT inválido")
	ErrInvalidSignature = errors.New("auth: assinatura do token inválida")
	ErrTokenExpired     = errors.New("auth: token expirado")
	ErrUnsupportedAlg   = errors.New("auth: algoritmo de assinatura não suportado")
)

/**
 * Estrutura interna fixa para o cabeçalho JWT em conformidade com HS256.
 */
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

const jwtHeaderHS256 = `{"alg":"HS256","typ":"JWT"}`

/**
 * MapClaims representa o conjunto flexível de declarações (claims) do token.
 */
type MapClaims map[string]interface{}

/**
 * NewClaims instancia um MapClaims preenchendo automaticamente os campos padrão ("sub", "iat", "exp").
 *
 * @param subject Identificador do sujeito (ex.: ID do usuário).
 * @param ttl Tempo de vida do token (Time to Live).
 * @param extra Mapeamento de claims adicionais a serem incorporados.
 *
 * @return Instância de MapClaims devidamente configurada.
 */
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

/**
 * Codifica bytes para string em formato Base64 URL Safe sem preenchimento (padding).
 */
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

/**
 * Decodifica string em formato Base64 URL Safe sem preenchimento (padding).
 */
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

/**
 * Calcula a assinatura HMAC-SHA256 para o conjunto header.payload fornecido.
 */
func sign(headerAndPayload string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(headerAndPayload))
	return mac.Sum(nil)
}

/**
 * Sign gera um token JWT assinado utilizando a chave secreta e o algoritmo HS256.
 *
 * @param claims Estrutura MapClaims contendo as informações do payload.
 * @param secret Chave simétrica utilizada na assinatura.
 *
 * @return String contendo o token formatado (header.payload.signature) ou erro.
 */
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

/**
 * Verify analisa, valida a integridade e extrai as claims de um token JWT.
 *
 * @param tokenString Token codificado em formato string.
 * @param secret Chave simétrica para validação do HMAC.
 *
 * @return Instância de MapClaims com os dados do token ou erro de validação.
 */
func Verify(tokenString string, secret []byte) (MapClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	headerB64, payloadB64, signatureB64 := parts[0], parts[1], parts[2]

	headerBytes, err := base64URLDecode(headerB64)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var head jwtHeader
	if err := json.Unmarshal(headerBytes, &head); err != nil {
		return nil, ErrInvalidToken
	}

	if head.Alg != "HS256" {
		return nil, ErrUnsupportedAlg
	}

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

/**
 * Normaliza tipos numéricos variados para int64 com segurança.
 */
func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case uint32:
		return int64(n), true
	case json.Number:
		val, err := n.Int64()
		return val, err == nil
	default:
		return 0, false
	}
}