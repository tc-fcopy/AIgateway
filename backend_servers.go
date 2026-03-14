package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Response struct {
	Message   string                 `json:"message"`
	FromPort  string                 `json:"from_port"`
	Path      string                 `json:"path"`
	Method    string                 `json:"method"`
	Headers   map[string]string      `json:"headers"`
	Timestamp string                 `json:"timestamp"`
}

func createHandler(port string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 收集请求头
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		// 读取请求体
		body, _ := io.ReadAll(r.Body)

		// 构造响应
		resp := Response{
			Message:   "Hello from backend server",
			FromPort:  port,
			Path:      r.URL.Path,
			Method:    r.Method,
			Headers:   headers,
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}

		// 记录请求
		log.Printf("[%s] %s %s - %s %s", port, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())

		// 如果有请求体，也记录
		if len(body) > 0 {
			log.Printf("[%s] Request Body: %s", port, string(body))
		}

		// 根据路径返回不同的响应
		switch {
		case r.URL.Path == "/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"service":  "AI Gateway Backend",
				"port":     port,
				"message":  "Backend server is: running",
				"endpoints": []string{
					"/v1/completions",
					"/v1/chat/completions",
					"/v1/models",
					"/health",
					"/echo",
				},
			})

		case r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "ok",
				"port":   port,
			})

		case r.URL.Path == "/v1/models":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"data": []map[string]interface{}{
					{
						"id":       "gpt-3.5-turbo",
						"object":   "model",
						"created":  1677610602,
						"owned_by": "openai",
					},
					{
						"id":       "gpt-4",
						"object":   "model",
						"created":  1687882410,
						"owned_by": "openai",
					},
				},
			})

		case r.URL.Path == "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			// 模拟OpenAI响应
			chatResp := map[string]interface{}{
				"id":      "chatcmpl-" + fmt.Sprintf("%d", time.Now().UnixNano()),
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   "gpt-3.5-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": fmt.Sprintf("This is a mock response from backend server on port %s. Your request was: %s", port, string(body)),
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":     30,
				},
			}
			json.NewEncoder(w).Encode(chatResp)

		case r.URL.Path == "/echo":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)

		default:
			// 默认返回请求信息
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}
	}
}

func startServer(port string) {
	mux := http.NewServeMux()

	// 添加CORS支持
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理OPTIONS预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		createHandler(port)(w, r)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("🚀 Backend server starting on port %s...", port)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("❌ Server on port %s failed: %v", port, err)
	}
}

func main() {
	log.Println("=== Starting Backend Servers ===")
	log.Println("Serving on ports: 9001, 9002")
	log.Println("Available endpoints:")
	log.Println("  GET  /           - Server info")
	log.Println("  GET  /health     - Health check")
	log.Println("  GET  /v1/models - List models")
	log.Println("  POST /v1/chat/completions - Chat completions (mock)")
	log.Println("  GET  /echo       - Echo request info")
	log.Println()

	// 启动两个服务器
	go startServer("9001")
	go startServer("9002")

	// 等待（主goroutine）
	select {}
}
