package cache

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CacheResponseWriter 缓存响应Writer
type CacheResponseWriter struct {
	gin.ResponseWriter
	bodyBuffer *bytes.Buffer
	// status     int
	// size       int
}

// NewCacheResponseWriter 创建缓存响应Writer
func NewCacheResponseWriter(w gin.ResponseWriter) *CacheResponseWriter {
	return &CacheResponseWriter{
		ResponseWriter: w,
		bodyBuffer:     bytes.NewBuffer(nil),
		// status:     http.StatusOK,
		// size:       0,
	}
}

// Write 写入响应
func (w *CacheResponseWriter) Write(data []byte) (int, error) {
	// 写入原始响应
	n, err := w.ResponseWriter.Write(data)
	if err != nil {
		return n, err
	}

	// 同时写入缓存buffer
	_, err = w.bodyBuffer.Write(data)
	if err != nil {
		return n, err
	}

	return n, nil
}

// WriteString 写入字符串
func (w *CacheResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// WriteHeader 写入Header
func (w *CacheResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// GetBody 获取缓存的响应体
func (w *CacheResponseWriter) GetBody() []byte {
	return w.bodyBuffer.Bytes()
}

// GetBodyString 获取响应体字符串
func (w *CacheResponseWriter) GetBodyString() string {
	return w.bodyBuffer.String()
}

// Flush 刷新缓冲区
func (w *CacheResponseWriter) Flush() {
	w.ResponseWriter.Flush()
}

// Hijack 劫持连接（用于WebSocket）
func (w *CacheResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
}

// CloseNotify 关闭通知
func (w *CacheResponseWriter) CloseNotify() <-chan bool {
	if cn, ok := w.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}
	return nil
}

// Push 推送（HTTP/2）
func (w *CacheResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// StreamCacheResponseWriter 流式响应缓存Writer
type StreamCacheResponseWriter struct {
	gin.ResponseWriter
	chunks       []string
	firstChunk   bool
	isStreaming  bool
	contentType  string
}

// NewStreamCacheResponseWriter 创建流式响应缓存Writer
func NewStreamCacheResponseWriter(w gin.ResponseWriter) *StreamCacheResponseWriter {
	return &StreamCacheResponseWriter{
		ResponseWriter: w,
		chunks:         make([]string, 0),
		firstChunk:     true,
		isStreaming:    true,
	}
}

// Write 写入流式响应
func (w *StreamCacheResponseWriter) Write(data []byte) (int, error) {
	// 检测是否为SSE流
	if w.firstChunk {
		w.firstChunk = false
		contentType := w.ResponseWriter.Header().Get("Content-Type")
		w.contentType = contentType
		w.isStreaming = strings.Contains(contentType, "text/event-stream")
	}

	// 写入原始响应
	n, err := w.ResponseWriter.Write(data)
	if err != nil {
		return n, err
	}

	// 如果是流式响应，缓存chunk
	if w.isStreaming {
		chunk := string(data)
		w.chunks = append(w.chunks, chunk)
	}

	return n, nil
}

// GetChunks 获取缓存的chunks
func (w *StreamCacheResponseWriter) GetChunks() []string {
	return w.chunks
}

// GetFullResponse 获取完整响应
func (w *StreamCacheResponseWriter) GetFullResponse() string {
	return strings.Join(w.chunks, "")
}

// IsStreaming 检查是否为流式响应
func (w *StreamCacheResponseWriter) IsStreaming() bool {
	return w.isStreaming
}

// AddChunk 添加chunk（块）
func (w *StreamCacheResponseWriter) AddChunk(chunk string) {
	w.chunks = append(w.chunks, chunk)
}

// ClearChunks 清空chunks
func (w *StreamCacheResponseWriter) ClearChunks() {
	w.chunks = make([]string, 0)
}
