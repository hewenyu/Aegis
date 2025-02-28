package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hewenyu/Aegis/internal/knowledge"
	"github.com/hewenyu/Aegis/internal/llm"
)

// Run 启动知识库示例
func main() {
	ctx := context.Background()

	// 1. 初始化LLM服务
	fmt.Println("初始化LLM服务...")
	llmService := llm.NewService()
	ollamaProvider, err := llm.NewOllamaProvider("http://localhost:11434")
	if err != nil {
		log.Fatalf("创建Ollama提供者失败: %v", err)
	}

	if err := llmService.RegisterProvider(ollamaProvider); err != nil {
		log.Fatalf("注册Ollama提供者失败: %v", err)
	}

	// 2. 列出可用模型
	models, err := llmService.ListModels(ctx)
	if err != nil {
		log.Fatalf("列出模型失败: %v", err)
	}

	fmt.Println("可用模型:")
	chatModel := "deepseek-r1:14b"
	embedModel := "mxbai-embed-large" // 使用完整的模型名称
	for provider, providerModels := range models {
		fmt.Printf("%s:\n", provider)
		for _, model := range providerModels {
			fmt.Printf("  - %s\n", model.Name)
		}
	}

	fmt.Printf("使用嵌入模型: %s\n", embedModel)

	// 定义默认集合名称
	defaultCollection := "default"
	// 4. 创建向量存储
	fmt.Println("创建向量存储...")
	vectorStore, err := knowledge.NewChromaVectorStore(knowledge.DefaultVectorStoreConfig())
	if err != nil {
		log.Fatalf("创建向量存储失败: %v", err)
	}

	// 5. 添加一些知识
	fmt.Println("创建知识实例...")
	knowledge1 := knowledge.Knowledge{
		ID:      uuid.New().String(),
		Type:    "concept",
		Content: "向量数据库是一种专门设计用于存储和高效检索向量（嵌入）的数据库系统。",
		Metadata: map[string]interface{}{
			"source": "definition",
			"topic":  "database",
		},
	}

	knowledge2 := knowledge.Knowledge{
		ID:      uuid.New().String(),
		Type:    "example",
		Content: "Pinecone、Milvus和Weaviate是流行的向量数据库实现。",
		Metadata: map[string]interface{}{
			"source": "examples",
			"topic":  "database",
		},
	}

	knowledge3 := knowledge.Knowledge{
		ID:      uuid.New().String(),
		Type:    "concept",
		Content: "大语言模型(LLM)是一种基于深度学习的自然语言处理系统，能够理解和生成人类语言。",
		Metadata: map[string]interface{}{
			"source": "definition",
			"topic":  "AI",
		},
	}

	knowledge4 := knowledge.Knowledge{
		ID:      uuid.New().String(),
		Type:    "concept",
		Content: "杭州2025 一月一日年天气晴朗",
		Metadata: map[string]interface{}{
			"source": "definition",
			"topic":  "weather",
		},
	}

	// 6. 将知识添加到向量存储
	fmt.Println("添加知识到向量存储...")
	startTime := time.Now()

	// 转换为文档并添加
	docs := []knowledge.Document{
		{
			ID:      knowledge1.ID,
			Content: fmt.Sprintf("%v", knowledge1.Content),
			Metadata: map[string]interface{}{
				"type":   knowledge1.Type,
				"source": knowledge1.Metadata["source"],
				"topic":  knowledge1.Metadata["topic"],
			},
		},
		{
			ID:      knowledge2.ID,
			Content: fmt.Sprintf("%v", knowledge2.Content),
			Metadata: map[string]interface{}{
				"type":   knowledge2.Type,
				"source": knowledge2.Metadata["source"],
				"topic":  knowledge2.Metadata["topic"],
			},
		},
		{
			ID:      knowledge3.ID,
			Content: fmt.Sprintf("%v", knowledge3.Content),
			Metadata: map[string]interface{}{
				"type":   knowledge3.Type,
				"source": knowledge3.Metadata["source"],
				"topic":  knowledge3.Metadata["topic"],
			},
		},
		{
			ID:      knowledge4.ID,
			Content: fmt.Sprintf("%v", knowledge4.Content),
			Metadata: map[string]interface{}{
				"type":   knowledge4.Type,
				"source": knowledge4.Metadata["source"],
				"topic":  knowledge4.Metadata["topic"],
			},
		},
	}

	if err := vectorStore.Add(ctx, defaultCollection, docs); err != nil {
		log.Fatalf("添加知识失败: %v", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("添加知识完成，耗时: %v\n", duration)

	// 7. 执行语义搜索
	fmt.Println("\n执行语义搜索...")
	searchStartTime := time.Now()

	searchResults, err := vectorStore.Search(ctx, defaultCollection, "向量数据库用来做什么的？", 5)
	if err != nil {
		log.Fatalf("语义搜索失败: %v", err)
	}

	searchDuration := time.Since(searchStartTime)
	fmt.Printf("搜索完成，耗时: %v\n", searchDuration)

	// 8. 显示搜索结果
	fmt.Println("\n搜索结果:")
	for i, result := range searchResults {
		fmt.Printf("%d. [ID: %s] %s\n", i+1, result.DocumentID, result.Content)
		fmt.Printf("   元数据: %v\n", result.Metadata)
		fmt.Printf("   相似度: %.4f\n", result.Similarity)
		fmt.Println()
	}

	// 9. 过滤搜索（使用相同的向量存储但手动过滤）
	fmt.Println("\n过滤搜索 (database主题):")
	databaseResults, err := vectorStore.Search(ctx, defaultCollection, "数据库", 5)
	if err != nil {
		log.Fatalf("过滤搜索失败: %v", err)
	}

	// 手动过滤主题为"database"的结果
	var filteredResults []knowledge.SearchResult
	for _, result := range databaseResults {
		if topic, ok := result.Metadata["topic"].(string); ok && topic == "database" {
			filteredResults = append(filteredResults, result)
		}
	}

	for i, result := range filteredResults {
		fmt.Printf("%d. %s\n", i+1, result.Content)
		fmt.Printf("   元数据: %v\n", result.Metadata)
	}

	// 10. 使用LLM服务回答问题
	chatRequest := llm.ChatRequest{
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "你是一个有帮助的AI助手。请使用以下知识库内容回答用户问题。",
			},
			{
				Role:    "user",
				Content: "什么是向量数据库？",
			},
		},
		Temperature: 0.7,
		MaxTokens:   500,
	}

	// 获取相关知识
	relevantResults, err := vectorStore.Search(ctx, defaultCollection, "向量数据库", 2)
	if err == nil && len(relevantResults) > 0 {
		knowledgeContext := "知识库内容:\n"
		for i, k := range relevantResults {
			knowledgeContext += fmt.Sprintf("%d. %s\n", i+1, k.Content)
		}

		// 将知识添加到系统提示中
		chatRequest.Messages[0].Content += "\n\n" + knowledgeContext
	}

	fmt.Println("\n使用知识库内容回答问题...")
	chatResponse, err := llmService.Chat(ctx, "ollama", chatModel, chatRequest)
	if err != nil {
		log.Printf("聊天请求失败: %v\n", err)
	} else {
		fmt.Printf("\n回答:\n%s\n", chatResponse.Message.Content)
	}
}
