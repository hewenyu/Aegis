package text

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// VectorizerTool 实现文档向量化工具
type VectorizerTool struct {
	id          string
	name        string
	description string
	version     string
	splitter    *TextSplitter
	embedder    Embedder
	vectorStore VectorStore
}

// Embedder 定义向量嵌入接口
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// VectorStore 定义向量存储接口
type VectorStore interface {
	Store(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, vector []float32, limit int) ([]SearchResult, error)
}

// SearchResult 定义搜索结果
type SearchResult struct {
	ID       string
	Score    float32
	Metadata map[string]interface{}
}

// NewVectorizerTool 创建新的向量化工具
func NewVectorizerTool(embedder Embedder, vectorStore VectorStore) *VectorizerTool {
	return &VectorizerTool{
		id:          "document-vectorizer",
		name:        "Document Vectorizer",
		description: "Convert documents into vector embeddings and store them for RAG",
		version:     "1.0.0",
		splitter:    NewTextSplitter(DefaultSplitOptions()),
		embedder:    embedder,
		vectorStore: vectorStore,
	}
}

// ID 返回工具ID
func (t *VectorizerTool) ID() string {
	return t.id
}

// Name 返回工具名称
func (t *VectorizerTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *VectorizerTool) Description() string {
	return t.description
}

// Version 返回工具版本
func (t *VectorizerTool) Version() string {
	return t.version
}

// VectorizeParams 定义向量化参数
type VectorizeParams struct {
	FilePath string                 // 文件路径
	Metadata map[string]interface{} // 元数据
}

// Execute 执行向量化
func (t *VectorizerTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 解析参数
	vectorizeParams, err := t.parseParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	// 读取文件
	content, err := t.readFile(vectorizeParams.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// 分割文本
	chunks := t.splitter.Split(content)

	// 处理每个块
	results := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		// 生成块ID
		chunkID := fmt.Sprintf("%s_chunk_%d", filepath.Base(vectorizeParams.FilePath), i)

		// 创建块元数据
		metadata := map[string]interface{}{
			"file_path":    vectorizeParams.FilePath,
			"chunk_index":  i,
			"total_chunks": len(chunks),
		}
		// 合并用户提供的元数据
		for k, v := range vectorizeParams.Metadata {
			metadata[k] = v
		}

		// 生成向量嵌入
		vector, err := t.embedder.Embed(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for chunk %d: %v", i, err)
		}

		// 存储向量
		err = t.vectorStore.Store(ctx, chunkID, vector, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to store vector for chunk %d: %v", i, err)
		}

		results = append(results, chunkID)
	}

	return map[string]interface{}{
		"status":     "success",
		"chunk_ids":  results,
		"num_chunks": len(chunks),
	}, nil
}

// Validate 验证参数
func (t *VectorizerTool) Validate(params map[string]interface{}) error {
	_, err := t.parseParams(params)
	return err
}

// parseParams 解析参数
func (t *VectorizerTool) parseParams(params map[string]interface{}) (*VectorizeParams, error) {
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	metadata, _ := params["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &VectorizeParams{
		FilePath: filePath,
		Metadata: metadata,
	}, nil
}

// readFile 读取文件内容
func (t *VectorizerTool) readFile(path string) (string, error) {
	// 检查是否是PDF文件
	if strings.ToLower(path[len(path)-4:]) == ".pdf" {
		reader := NewPDFReader()
		content, err := reader.Read(path)
		if err != nil {
			return "", fmt.Errorf("failed to read PDF file: %w", err)
		}
		return content, nil
	}

	// 处理普通文本文件
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
