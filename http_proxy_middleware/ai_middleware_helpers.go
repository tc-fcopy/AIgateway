package http_proxy_middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

func aiReadBody(c *gin.Context) ([]byte, error) {
	if c.Request == nil || c.Request.Body == nil {
		return []byte{}, nil
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func aiResetBody(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
}

func aiEstimateTokens(body []byte) int64 {
	if len(body) == 0 {
		return 1
	}

	// Rough estimate: 1 token ~= 4 characters.
	n := int64(len(strings.TrimSpace(string(body)))) / 4
	if n <= 0 {
		return 1
	}
	return n
}

func aiIsConsumerAllowed(consumerName string, allowedConsumers []string) bool {
	if len(allowedConsumers) == 0 {
		return true
	}
	for _, allowed := range allowedConsumers {
		if allowed == "*" || allowed == consumerName {
			return true
		}
	}
	return false
}

func aiExtractPrompt(payload map[string]interface{}) string {
	if p, ok := payload["prompt"].(string); ok && p != "" {
		return p
	}

	if messages, ok := payload["messages"].([]interface{}); ok {
		for i := len(messages) - 1; i >= 0; i-- {
			msg, ok := messages[i].(map[string]interface{})
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role != "user" {
				continue
			}
			if content, ok := msg["content"].(string); ok {
				return content
			}
		}
	}

	if input, ok := payload["input"].(string); ok {
		return input
	}

	return ""
}

func aiGetModel(payload map[string]interface{}, c *gin.Context) string {
	if payload != nil {
		if m, ok := payload["model"].(string); ok && m != "" {
			return m
		}
	}
	if m, ok := c.Get("ai_model"); ok {
		if model, ok := m.(string); ok && model != "" {
			return model
		}
	}
	return c.Query("model")
}

func aiParseJSONBody(body []byte) (map[string]interface{}, error) {
	if len(body) == 0 {
		return map[string]interface{}{}, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
