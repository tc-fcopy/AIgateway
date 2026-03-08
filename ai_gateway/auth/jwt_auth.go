package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"hash"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// JWTAuth validates JWT tokens for configured algorithms.
type JWTAuth struct {
	secret     string
	algorithms map[string]struct{}
}

func NewJWTAuth(secret string, algorithms []string) *JWTAuth {
	if len(algorithms) == 0 {
		algorithms = []string{"HS256"}
	}

	allowed := make(map[string]struct{}, len(algorithms))
	for _, alg := range algorithms {
		allowed[strings.ToUpper(strings.TrimSpace(alg))] = struct{}{}
	}

	return &JWTAuth{secret: secret, algorithms: allowed}
}

// JWTClaims represents supported claims.
type JWTClaims struct {
	Subject   string                 `json:"sub"`
	Issuer    string                 `json:"iss"`
	Audience  string                 `json:"aud"`
	ExpiresAt int64                  `json:"exp"`
	IssuedAt  int64                  `json:"iat"`
	Raw       map[string]interface{} `json:"-"`
}

func (j *JWTAuth) Authenticate(tokenString string) (*JWTClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimPrefix(tokenString, "bearer ")
	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return nil, ErrJWTTokenMissing
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidJWTToken
	}

	headerJSON, err := decodeBase64URL(parts[0])
	if err != nil {
		return nil, ErrInvalidJWTToken
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalidJWTToken
	}

	alg, _ := header["alg"].(string)
	alg = strings.ToUpper(strings.TrimSpace(alg))
	if alg == "" {
		return nil, ErrInvalidAlgorithm
	}
	if _, ok := j.algorithms[alg]; !ok {
		return nil, ErrInvalidAlgorithm
	}

	if err := j.verifySignature(parts[0], parts[1], parts[2], alg); err != nil {
		return nil, err
	}

	payloadJSON, err := decodeBase64URL(parts[1])
	if err != nil {
		return nil, ErrInvalidJWTToken
	}

	raw := map[string]interface{}{}
	if err := json.Unmarshal(payloadJSON, &raw); err != nil {
		return nil, ErrInvalidJWTToken
	}

	claims := &JWTClaims{Raw: raw}
	claims.Subject = asString(raw["sub"])
	claims.Issuer = asString(raw["iss"])
	claims.Audience = asString(raw["aud"])
	claims.ExpiresAt = asInt64(raw["exp"])
	claims.IssuedAt = asInt64(raw["iat"])

	if claims.Subject == "" {
		return nil, ErrMissingSubjectClaim
	}

	if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrJWTTokenExpired
	}

	return claims, nil
}

func (j *JWTAuth) AuthenticateWithContext(_ *gin.Context, tokenString string) (*JWTClaims, error) {
	return j.Authenticate(tokenString)
}

func (j *JWTAuth) verifySignature(headerPart, payloadPart, sigPart, alg string) error {
	signingInput := headerPart + "." + payloadPart

	switch alg {
	case "HS256", "HS384", "HS512":
		if j.secret == "" {
			return ErrInvalidJWTToken
		}

		var h func() hash.Hash
		switch alg {
		case "HS256":
			h = sha256.New
		case "HS384":
			h = sha512.New384
		default:
			h = sha512.New
		}

		mac := hmac.New(h, []byte(j.secret))
		_, _ = mac.Write([]byte(signingInput))
		expected := mac.Sum(nil)

		provided, err := decodeBase64URL(sigPart)
		if err != nil {
			return ErrInvalidJWTToken
		}

		if !hmac.Equal(expected, provided) {
			return ErrInvalidJWTToken
		}
		return nil

	case "RS256":
		return ErrRS256NotImplemented
	default:
		return ErrUnsupportedAlgorithm
	}
}

func decodeBase64URL(v string) ([]byte, error) {
	out, err := base64.RawURLEncoding.DecodeString(v)
	if err == nil {
		return out, nil
	}
	return nil, err
}

func asString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []interface{}:
		if len(x) > 0 {
			if first, ok := x[0].(string); ok {
				return first
			}
		}
	}
	return ""
}

func asInt64(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	}
	return 0
}

func (j *JWTAuth) GetConsumerName(claims *JWTClaims) string {
	if claims == nil {
		return ""
	}
	return claims.Subject
}

var (
	ErrJWTTokenMissing      = NewAuthError("JWT token is missing")
	ErrInvalidJWTToken      = NewAuthError("invalid JWT token")
	ErrJWTTokenExpired      = NewAuthError("JWT token is expired")
	ErrInvalidAlgorithm     = NewAuthError("invalid JWT algorithm")
	ErrUnsupportedAlgorithm = NewAuthError("unsupported JWT algorithm")
	ErrMissingSubjectClaim  = NewAuthError("missing 'sub' claim in JWT token")
	ErrInvalidClaims        = NewAuthError("invalid JWT claims")
	ErrRS256NotImplemented  = NewAuthError("RS256 algorithm not implemented yet")
)
