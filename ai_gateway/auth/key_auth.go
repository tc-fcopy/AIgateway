package auth

import (
	"gateway/ai_gateway/config"
	"gateway/ai_gateway/consumer"
	"github.com/gin-gonic/gin"
)

// KeyAuth validates api key and resolves consumer.
type KeyAuth struct {
	configMgr   *config.ConfigManager
	consumerMgr *consumer.Manager
}

func NewKeyAuth() *KeyAuth {
	return &KeyAuth{
		configMgr:   config.AIConfManager,
		consumerMgr: consumer.ConsumerManager,
	}
}

func (k *KeyAuth) Authenticate(apiKey string) (*consumer.Consumer, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	cons, ok := k.consumerMgr.GetByCredential(apiKey)
	if !ok {
		return nil, ErrAPIKeyInvalid
	}
	if !cons.IsEnabled() {
		return nil, ErrConsumerDisabled
	}
	if !cons.IsKeyType() {
		return nil, ErrInvalidConsumerType
	}

	return cons, nil
}

func (k *KeyAuth) AuthenticateWithContext(c *gin.Context) (*consumer.Consumer, error) {
	if !k.configMgr.IsEnabled() {
		return nil, ErrAIGatewayDisabled
	}

	aiConfig := k.configMgr.GetConfig()
	if aiConfig == nil || !aiConfig.Enable || !aiConfig.DefaultService.EnableKeyAuth {
		return nil, ErrKeyAuthNotEnabled
	}

	apiKey := extractAPIKey(c)
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	return k.Authenticate(apiKey)
}

func extractAPIKey(c *gin.Context) string {
	conf := config.AIConfManager.GetConfig()
	if conf == nil {
		return ""
	}

	for _, keyName := range conf.KeyAuth.KeyNames {
		apiKey := c.GetHeader(keyName)
		if apiKey != "" {
			return apiKey
		}

		apiKey = c.Query(keyName)
		if apiKey != "" {
			return apiKey
		}

		apiKey = c.GetHeader(toLower(keyName))
		if apiKey != "" {
			return apiKey
		}
	}

	return ""
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}

var (
	ErrAPIKeyMissing       = NewAuthError("API key is missing")
	ErrAPIKeyInvalid       = NewAuthError("API key is invalid")
	ErrConsumerDisabled    = NewAuthError("consumer is disabled")
	ErrInvalidConsumerType = NewAuthError("invalid consumer type for key auth")
	ErrAIGatewayDisabled   = NewAuthError("AI gateway is disabled")
	ErrServiceNotFound     = NewAuthError("service not found")
	ErrKeyAuthNotEnabled   = NewAuthError("Key auth is not enabled for this service")
)

type AuthError struct {
	Message string
}

func NewAuthError(message string) *AuthError {
	return &AuthError{Message: message}
}

func (e *AuthError) Error() string {
	return e.Message
}
