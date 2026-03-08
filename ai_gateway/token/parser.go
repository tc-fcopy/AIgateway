package token

import (
	"encoding/json"
	"fmt"
)

// TokenUsage Token使用信息
type TokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// OpenAIResponse OpenAI格式的完整响应
type OpenAIResponse struct {
	ID      string     `json:"id"`
	Object  string     `json:"object"`
	Created int64      `json:"created"`
	Model   string     `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   TokenUsage `json:"usage"`
}

// Choice OpenAI响应的选择
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	Delta        Delta    `json:"delta"`
	FinishReason string   `json:"finish_reason"`
}

// Message 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Delta 流式响应的增量
type Delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ParseOpenAIResponse 解析OpenAI格式响应
func ParseOpenAIResponse(body []byte) (*TokenUsage, error) {
	var result OpenAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}
	return &result.Usage, nil
}

// ParseOpenAIResponseFull 解析完整的OpenAI响应（包含 Usage）
func ParseOpenAIResponseFull(body []byte) (*OpenAIResponse, error) {
	var result OpenAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}
	return &result, nil
}

// IsStreamChunk 检查是否为流式响应chunk
func IsStreamChunk(body []byte) bool {
	return len(body) > 6 && string(body[:6]) == "data: "
}

// ParseStreamChunk 解析流式响应chunk
func ParseStreamChunk(body []byte) (*TokenUsage, bool, error) {
	// 检查是否为 SSE 格式
	if !IsStreamChunk(body) {
		return nil, false, nil
	}

	// 提取 JSON 部分
	jsonStr := string(body[6:]) // 跳过 "data: "
	jsonStr = trimWhitespace(jsonStr)

	// 检查是否为 [DONE]
	if jsonStr == "[DONE]" {
		return nil, false, nil
	}

	// 尝试解析为完整响应
	if fullResp, err := ParseOpenAIResponseFull(body[6:]); err == nil {
		if fullResp.Usage.TotalTokens > 0 {
			return &fullResp.Usage, true, nil
		}
	}

	// 尝试解析为仅 Usage 的响应
	var result struct {
		Usage TokenUsage `json:"usage"`
	}
	if err := json.Unmarshal(body[6:], &result); err == nil {
		if result.Usage.TotalTokens > 0 {
			return &result.Usage, true, nil
		}
	}

	// 尝试解析为包含 Usage 的流式响应
	var streamResult struct {
		ID      string    `json:"id"`
		Object  string    `json:"object"`
		Created int64     `json:"created"`
		Model   string    `json:"model"`
		Choices []struct {
			Index        int    `json:"index"`
			Message      struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Delta        Delta `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage TokenUsage `json:"usage"`
	}
	if err := json.Unmarshal(body[6:], &streamResult); err == nil {
		if streamResult.Usage.TotalTokens > 0 {
			return &streamResult.Usage, true, nil
		}
	}

	return nil, false, nil
}

// ExtractContentFromStreamChunk 从流式响应chunk中提取内容
func ExtractContentFromStreamChunk(body []byte) string {
	if !IsStreamChunk(body) {
		return ""
	}

	jsonStr := string(body[6:])
	jsonStr = trimWhitespace(jsonStr)

	// 检查是否为 [DONE]
	if jsonStr == "[DONE]" {
		return ""
	}

	var result OpenAIResponse
	if err := json.Unmarshal(body[6:], &result); err != nil {
		if len(result.Choices) > 0 {
			if result.Choices[0].Delta.Content != "" {
				return result.Choices[0].Delta.Content
			}
			if result.Choices[0].Message.Content != "" {
				return result.Choices[0].Message.Content
			}
		}
	}

	return ""
}

// trimWhitespace 去除首尾空白字符
func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	// 去除前导空白
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// 去除后导空白
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// MergeUsage 合并Token使用量
func MergeUsage(usage1, usage2 *TokenUsage) *TokenUsage {
	if usage1 == nil {
		return usage2
	}
	if usage2 == nil {
		return usage1
	}
	return &TokenUsage{
		PromptTokens:     usage1.PromptTokens + usage2.PromptTokens,
		CompletionTokens: usage1.CompletionTokens + usage2.CompletionTokens,
		TotalTokens:      usage1.TotalTokens + usage2.TotalTokens,
	}
}

// AddUsage 累加Token使用量
func AddUsage(usage *TokenUsage, deltaPrompt, deltaCompletion, deltaTotal int64) *TokenUsage {
	if usage == nil {
		return &TokenUsage{
			PromptTokens:     deltaPrompt,
			CompletionTokens: deltaCompletion,
			TotalTokens:      deltaTotal,
		}
	}
	return &TokenUsage{
		PromptTokens:     usage.PromptTokens + deltaPrompt,
		CompletionTokens: usage.CompletionTokens + deltaCompletion,
		TotalTokens:      usage.TotalTokens + deltaTotal,
	}
}
