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
 * Standard errors returned throughout the JWT lifecycle.
 */
var (
	ErrInvalidToken     = errors.New("auth: invalid JWT token")
	ErrInvalidSignature = errors.New("auth: invalid token signature")
	ErrTokenExpired     = errors.New("auth: token expired")
	ErrUnsupportedAlg   = errors.New("auth: unsupported signing algorithm")
)

/**
 * jwtHeader is the fixed internal structure for the JWT header, compliant with HS256.
 */
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

const jwtHeaderHS256 = `{"alg":"HS256","typ":"JWT"}`

/**
 * MapClaims represents the flexible set of claims carried by the token.
 */
type MapClaims map[string]interface{}

/**
 * NewClaims builds a MapClaims instance, automatically filling in the
 * standard fields ("sub", "iat", "exp").
 *
 * @param subject Subject identifier (e.g. user ID).
 * @param ttl     Token time to live.
 * @param extra   Additional claims to merge into the result.
 *
 * @return A properly configured MapClaims instance.
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
 * base64URLEncode encodes bytes into an unpadded Base64 URL-safe string.
 */
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

/**
 * base64URLDecode decodes an unpadded Base64 URL-safe string.
 */
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

/**
 * sign computes the HMAC-SHA256 signature for the given header.payload string.
 */
func sign(headerAndPayload string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(headerAndPayload))
	return mac.Sum(nil)
}

/**
 * Sign generates a JWT token signed with the given secret key using HS256.
 *
 * @param claims MapClaims structure containing the payload information.
 * @param secret Symmetric key used to sign the token.
 *
 * @return The formatted token string (header.payload.signature), or an error.
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
 * Verify parses a JWT token, validates its integrity and extracts its claims.
 *
 * @param tokenString The encoded token string.
 * @param secret      Symmetric key used to validate the HMAC signature.
 *
 * @return A MapClaims instance with the decoded token data, or a validation error.
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
 * toInt64 safely normalizes various numeric types into int64.
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