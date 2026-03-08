package jwt

import (
	"errors"

	"gateway/ai_gateway/auth"
)

// Claims is an external-facing lightweight claim object.
type Claims struct {
	Subject   string                 `json:"sub"`
	Issuer    string                 `json:"iss"`
	Audience  string                 `json:"aud"`
	ExpiresAt int64                  `json:"exp"`
	IssuedAt  int64                  `json:"iat"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

func ValidateToken(tokenString string, secret string, algorithm string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("JWT token is missing")
	}

	if algorithm == "" {
		algorithm = "HS256"
	}

	jwtAuth := auth.NewJWTAuth(secret, []string{algorithm})
	parsed, err := jwtAuth.Authenticate(tokenString)
	if err != nil {
		return nil, err
	}

	return &Claims{
		Subject:   parsed.Subject,
		Issuer:    parsed.Issuer,
		Audience:  parsed.Audience,
		ExpiresAt: parsed.ExpiresAt,
		IssuedAt:  parsed.IssuedAt,
		Custom:    parsed.Raw,
	}, nil
}
