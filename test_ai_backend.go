package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ChatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []ChatMessage   `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

var (
	port    = flag.Int("port", 9001, "Port to listen on")
	model   = flag.String("model", "gpt-4", "Model name to report")
	verbose = flag.Bool("v", false, "Verbose logging")
)

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", chatHandler)
	mux.HandleFunc("/", healthHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		Handler:      mux,
	}

	go func() {
		log.Printf("🚀 AI Backend [%s] starting on port %d", *model, *port)
		log.Fatal(server.ListenAndServe())
	}()

	// 监听关闭信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Server shutting down...")
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if *verbose {
		log.Printf("📥 Received request:\nHeaders: %v\nBody: %s", r.Header, string(body))
	}

	// 解析请求
	var req ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 检查 Header
	xAIHeader := r.Header.Get("X-AI-Model")
	log.Printf("🤖 Received model from body: '%s'", req.Model)
	if xAIHeader != "" {
		log.Printf("🤖 Received model from X-AI-Model header: '%s'", xAIHeader)
	}

	// 构建响应
	resp := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   *model, // 返回我们配置的模型名称
		Choices: []Choice{{
			Index: 0,
			Message: ChatMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("Hello! I'm %s. You sent: '%s'",
					*model, getFirstUserMessage(req.Messages)),
			},
			FinishReason: "stop",
		}},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	log.Printf("✅ Responded with model: '%s'", *model)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"model":  *model,
		"port":   *port,
	})
}

func getFirstUserMessage(messages []ChatMessage) string {
	for _, msg := range messages {
		if msg.Role == "user" {
			return msg.Content
		}
	}
	return ""
}
