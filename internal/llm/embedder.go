package llm

import (
	"context"
	"fmt"
)

// LLMEmbedder 是一个使用LLM服务进行嵌入的Embedder实现
type LLMEmbedder struct {
	service     Service
	provider    string
	model       string
	dimensions  int
	maxPoolSize int
}

// NewLLMEmbedder 创建一个新的LLM嵌入器
func NewLLMEmbedder(service Service, provider, model string, dimensions int) *LLMEmbedder {
	return &LLMEmbedder{
		service:     service,
		provider:    provider,
		model:       model,
		dimensions:  dimensions,
		maxPoolSize: 10, // 默认并发池大小
	}
}

// SetMaxPoolSize 设置最大并发池大小
func (e *LLMEmbedder) SetMaxPoolSize(size int) {
	if size > 0 {
		e.maxPoolSize = size
	}
}

// Embed 将内容转换为向量
func (e *LLMEmbedder) Embed(ctx context.Context, content interface{}) ([]float64, error) {
	// 将内容转换为字符串
	var textContent string
	switch c := content.(type) {
	case string:
		textContent = c
	case []byte:
		textContent = string(c)
	case fmt.Stringer:
		textContent = c.String()
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// 创建嵌入请求
	request := EmbeddingRequest{
		Input: textContent,
	}

	// 调用LLM服务获取嵌入
	response, err := e.service.Embed(ctx, e.provider, e.model, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	// 确保嵌入维度正确
	if e.dimensions > 0 && len(response.Embedding) != e.dimensions {
		return nil, fmt.Errorf("expected embedding dimension %d, got %d", e.dimensions, len(response.Embedding))
	}

	return response.Embedding, nil
}

// BatchEmbed 批量将内容转换为向量
func (e *LLMEmbedder) BatchEmbed(ctx context.Context, contents []interface{}) ([][]float64, error) {
	// 创建结果切片
	results := make([][]float64, len(contents))
	errs := make([]error, len(contents))

	// 使用有限的goroutine池来处理批量嵌入
	semaphore := make(chan struct{}, e.maxPoolSize)
	done := make(chan int, len(contents))

	// 启动工作goroutine
	for i, content := range contents {
		go func(idx int, c interface{}) {
			// 获取信号量
			semaphore <- struct{}{}
			defer func() {
				// 释放信号量
				<-semaphore
				// 通知完成
				done <- idx
			}()

			// 执行嵌入
			embedding, err := e.Embed(ctx, c)
			if err != nil {
				errs[idx] = err
				return
			}

			results[idx] = embedding
		}(i, content)
	}

	// 等待所有工作完成
	for i := 0; i < len(contents); i++ {
		<-done
	}

	// 检查是否有错误
	for _, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("batch embedding failed: %w", err)
		}
	}

	return results, nil
}
