package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ollama/ollama/api"
)

const (
	embedModel = "mxbai-embed-large"
)

// OllamaProvider 实现了Ollama的Provider接口
type OllamaProvider struct {
	embedModel string
	client     *api.Client
}

// NewOllamaProvider 创建一个新的Ollama提供者实例
func NewOllamaProvider(endpoint string) (Provider, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}
	client := api.NewClient(endpointURL, http.DefaultClient)
	return &OllamaProvider{
		embedModel: embedModel,
		client:     client,
	}, nil
}

// Name 返回提供者的名称
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// GetEmbedModel 返回嵌入模型
func (p *OllamaProvider) GetEmbedModel() string {
	return p.embedModel
}

// ListModels 返回可用的模型列表
func (p *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	models, err := p.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	var modelInfos []ModelInfo
	for _, model := range models.Models {
		modelInfos = append(modelInfos, ModelInfo{
			Name:               model.Name,
			ContextWindowSize:  4096,  // 默认值，可能需要根据模型调整
			MaxOutputTokens:    2048,  // 默认值，可能需要根据模型调整
			SupportsImageInput: false, // 根据模型能力设置
		})
	}
	return modelInfos, nil
}

// GetModel 返回指定模型的信息
func (p *OllamaProvider) GetModel(ctx context.Context, modelID string) (ModelInfo, error) {
	// 目前返回默认的ModelInfo，因为Ollama API不提供详细的模型信息
	return ModelInfo{
		Name:               modelID,
		ContextWindowSize:  4096,
		MaxOutputTokens:    2048,
		SupportsImageInput: false,
	}, nil
}

// Complete 生成文本补全
func (p *OllamaProvider) Complete(ctx context.Context, modelID string, request CompletionRequest) (CompletionResponse, error) {
	options := map[string]interface{}{
		"temperature": float32(request.Temperature),
		"top_p":       float32(request.TopP),
	}
	if len(request.Stop) > 0 {
		options["stop"] = request.Stop
	}

	generateRequest := api.GenerateRequest{
		Model:   modelID,
		Prompt:  request.Prompt,
		Options: options,
	}

	var finalResponse string
	var promptEvalCount, evalCount int

	err := p.client.Generate(ctx, &generateRequest, func(response api.GenerateResponse) error {
		finalResponse += response.Response
		promptEvalCount = response.PromptEvalCount
		evalCount = response.EvalCount
		return nil
	})

	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to generate completion: %w", err)
	}

	return CompletionResponse{
		Text: finalResponse,
		Usage: Usage{
			PromptTokens:     promptEvalCount,
			CompletionTokens: evalCount,
			TotalTokens:      promptEvalCount + evalCount,
		},
		Timestamp: 0, // Ollama API不提供创建时间戳
	}, nil
}

// Chat 处理聊天补全
func (p *OllamaProvider) Chat(ctx context.Context, modelID string, request ChatRequest) (ChatResponse, error) {
	messages := make([]api.Message, len(request.Messages))
	for i, msg := range request.Messages {
		messages[i] = api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	options := map[string]interface{}{
		"temperature": float32(request.Temperature),
		"top_p":       float32(request.TopP),
	}
	if len(request.Stop) > 0 {
		options["stop"] = request.Stop
	}

	chatRequest := api.ChatRequest{
		Model:    modelID,
		Messages: messages,
		Options:  options,
	}

	var finalResponse api.ChatResponse
	var responseContent strings.Builder

	err := p.client.Chat(ctx, &chatRequest, func(response api.ChatResponse) error {
		responseContent.WriteString(response.Message.Content)
		finalResponse = response
		return nil
	})

	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to generate chat response: %w", err)
	}

	// 使用累积的响应内容
	if finalResponse.Message.Content == "" {
		finalResponse.Message.Content = responseContent.String()
	}

	return ChatResponse{
		Message: Message{
			Role:    finalResponse.Message.Role,
			Content: finalResponse.Message.Content,
		},
		Usage: Usage{
			PromptTokens:     finalResponse.PromptEvalCount,
			CompletionTokens: finalResponse.EvalCount,
			TotalTokens:      finalResponse.PromptEvalCount + finalResponse.EvalCount,
		},
		Timestamp: 0, // Ollama API不提供创建时间戳
	}, nil
}

// Embed 生成文本的嵌入向量
func (p *OllamaProvider) Embed(ctx context.Context, modelID string, request EmbeddingRequest) (EmbeddingResponse, error) {
	embedRequest := api.EmbeddingRequest{
		Model:  modelID,
		Prompt: request.Input,
	}

	response, err := p.client.Embeddings(ctx, &embedRequest)
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	return EmbeddingResponse{
		Embedding: response.Embedding,
		Usage: Usage{
			PromptTokens: len(request.Input), // 近似的token计数
			TotalTokens:  len(request.Input),
		},
	}, nil
}
