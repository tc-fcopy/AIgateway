package token

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// StreamParser 流式响应解析器
type StreamParser struct {
	buffer      *bytes.Buffer
	usage       *TokenUsage
	content     bytes.Buffer
	mu          sync.Mutex
	contentParts []string
	onChunk     func(content string)
	onComplete   func(usage *TokenUsage, fullContent string)
}

// NewStreamParser 创建流式响应解析器
func NewStreamParser() *StreamParser {
	return &StreamParser{
		buffer:      &bytes.Buffer{},
		usage:       &TokenUsage{},
		contentParts: make([]string, 0),
	}
}

// SetChunkCallback 设置chunk回调
func (p *StreamParser) SetChunkCallback(callback func(content string)) {
	p.onChunk = callback
}

// SetCompleteCallback 设置完成回调
func (p *StreamParser) SetCompleteCallback(callback func(usage *TokenUsage, fullContent string)) {
	p.onComplete = callback
}

// ParseChunk 解析单个chunk
func (p *StreamParser) ParseChunk(chunk []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 尝试解析为 SSE 格式的 chunk
	if IsStreamChunk(chunk) {
		// 提取内容
		content := ExtractContentFromStreamChunk(chunk)
		if content != "" {
			p.content.WriteString(content)
			p.contentParts = append(p.contentParts, content)

			// 调用chunk回调
			if p.onChunk != nil {
				p.onChunk(content)
			}
		}

		// 尝试解析 Token Usage
		if usage, ok, _ := ParseStreamChunk(chunk); ok && usage != nil {
			p.usage = usage
		}
	} else {
		// 非SSE格式，直接处理为普通响应
		if usage, err := ParseOpenAIResponse(chunk); err == nil {
			p.usage = usage
		}
	}
}

// ParseStreamReader 从io.Reader解析流式响应
func (p *StreamParser) ParseStreamReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		p.ParseChunk(line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 解析完成，调用回调
	if p.onComplete != nil {
		p.onComplete(p.usage, p.content.String())
	}

	return nil
}

// GetUsage 获取Token使用量
func (p *StreamParser) GetUsage() *TokenUsage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.usage
}

// GetContent 获取完整内容
func (p *StreamParser) GetContent() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.content.String()
}

// Reset 重置解析器状态
func (p *StreamParser) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.buffer.Reset()
	p.usage = &TokenUsage{}
	p.content.Reset()
	p.contentParts = make([]string, 0)
}

// GetContentParts 获取内容分片
func (p *StreamParser) GetContentParts() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.contentParts
}

// StreamWriter 流式响应写入器
type StreamWriter struct {
	io.Writer
	parser   *StreamParser
	onWrite  func(data []byte) (int, error)
}

// NewStreamWriter 创建流式响应写入器
func NewStreamWriter(writer io.Writer, parser *StreamParser) *StreamWriter {
	return &StreamWriter{
		Writer:  writer,
		parser:   parser,
	}
}

// SetWriteCallback 设置写入回调
func (w *StreamWriter) SetWriteCallback(callback func(data []byte) (int, error)) {
	w.onWrite = callback
}

// Write 实现io.Writer接口
func (w *StreamWriter) Write(data []byte) (int, error) {
	// 先解析chunk
	w.parser.ParseChunk(data)

	// 如果有自定义回调，先调用
	if w.onWrite != nil {
		return w.onWrite(data)
	}

	// 否则直接写入底层writer
	return w.Writer.Write(data)
}

// TokenCountResponseWriter Token计数响应写入器
type TokenCountResponseWriter struct {
	gin.ResponseWriter
	parser         *StreamParser
	startTime      int64
	firstTokenTime int64
	onUsageUpdate  func(usage *TokenUsage)
}

// NewTokenCountResponseWriter 创建Token计数响应写入器
func NewTokenCountResponseWriter(writer gin.ResponseWriter, parser *StreamParser) *TokenCountResponseWriter {
	return &TokenCountResponseWriter{
		ResponseWriter: writer,
		parser:         parser,
		startTime:      getCurrentTimeMillis(),
	}
}

// SetUsageUpdateCallback 设置使用量更新回调
func (w *TokenCountResponseWriter) SetUsageUpdateCallback(callback func(usage *TokenUsage)) {
	w.onUsageUpdate = callback
}

// Write 实现gin.ResponseWriter的Write方法
func (w *TokenCountResponseWriter) Write(data []byte) (int, error) {
	// 如果是第一次写入且记录首字时间
	if w.firstTokenTime == 0 && len(data) > 0 {
		w.firstTokenTime = getCurrentTimeMillis()
	}

	// 解析数据
	w.parser.ParseChunk(data)

	// 检查是否有新的使用量
	if w.parser.GetUsage().TotalTokens > 0 && w.onUsageUpdate != nil {
		w.onUsageUpdate(w.parser.GetUsage())
	}

	// 写入底层ResponseWriter
	return w.ResponseWriter.Write(data)
}

// WriteString 实现gin.ResponseWriter的WriteString方法
func (w *TokenCountResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// GetTotalDuration 获取总时长（毫秒）
func (w *TokenCountResponseWriter) GetTotalDuration() int64 {
	return getCurrentTimeMillis() - w.startTime
}

// GetFirstTokenDuration 获取首字延迟（毫秒）
func (w *TokenCountResponseWriter) GetFirstTokenDuration() int64 {
	if w.firstTokenTime == 0 {
		return 0
	}
	return w.firstTokenTime - w.startTime
}

// getCurrentTimeMillis 获取当前时间毫秒数
func getCurrentTimeMillis() int64 {
	return int64(time.Now().UnixNano() / 1e6)
}

// getTotalTokenCount 获取总Token数
func (w *TokenCountResponseWriter) GetTotalTokenCount() int64 {
	if w.parser.GetUsage() == nil {
		return 0
	}
	return w.parser.GetUsage().TotalTokens
}
