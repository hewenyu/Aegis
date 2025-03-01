package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/hewenyu/Aegis/internal/knowledge"
	"github.com/hewenyu/Aegis/internal/llm"
	"github.com/hewenyu/Aegis/internal/tool/text"
)

func main() {
	// 设置文件路径
	filePath := "C:\\Users\\boringsoft\\Downloads\\humlum-vestergaard-2024-the-unequal-adoption-of-chatgpt-exacerbates-existing-inequalities-among-workers.pdf"

	// 创建 PDF 读取器并获取文件信息
	pdfReader := text.NewPDFReader()
	fileInfo, err := pdfReader.GetFileInfo(filePath)
	if err != nil {
		log.Fatalf("Failed to get PDF info: %v", err)
	}
	fmt.Printf("PDF 文件信息：\n%+v\n\n", fileInfo)

	// 创建 LLM 服务
	llmService := llm.NewService()

	// 注册 Ollama 提供者
	ollamaProvider, err := llm.NewOllamaProvider("http://localhost:11434")
	if err != nil {
		log.Fatalf("Failed to create Ollama provider: %v", err)
	}
	err = llmService.RegisterProvider(ollamaProvider)
	if err != nil {
		log.Fatalf("Failed to register Ollama provider: %v", err)
	}

	// 创建 Ollama 适配器
	ollamaAdapter := llm.NewLLMAdapter(ollamaProvider, "deepseek-r1:14b")

	// 创建向量存储
	vectorStore, err := knowledge.NewChromaVectorStore(knowledge.DefaultVectorStoreConfig())
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer vectorStore.Close()

	// 创建向量存储适配器
	vectorAdapter := knowledge.NewVectorAdapter(vectorStore, "papers")

	// 创建文档向量化工具
	vectorizer := text.NewVectorizerTool(ollamaAdapter, vectorAdapter)

	// 创建论文总结工具
	summarizer := text.NewSummarizerTool(ollamaAdapter)

	fmt.Println("开始向量化文档...")

	// 向量化文档
	vectorizeResult, err := vectorizer.Execute(context.Background(), map[string]interface{}{
		"file_path": filePath,
		"metadata": map[string]interface{}{
			"type":       "research_paper",
			"year":       2024,
			"file_name":  filepath.Base(filePath),
			"page_count": fileInfo["page_count"],
			"title":      fileInfo["title"],
			"author":     fileInfo["author"],
			"keywords":   fileInfo["keywords"],
		},
	})
	if err != nil {
		log.Printf("Failed to vectorize document: %v", err)
	} else {
		fmt.Printf("向量化结果：\n%+v\n\n", vectorizeResult)
	}

	fmt.Println("开始总结论文...")
	// 总结论文
	summarizeResult, err := summarizer.Execute(context.Background(), map[string]interface{}{
		"file_path":  filePath,
		"max_length": 2000,
		"focus_areas": []string{
			"研究背景和目的",
			"研究方法和数据",
			"主要发现",
			"结论和建议",
			"创新点和局限性",
		},
		"language": "zh",
	})
	if err != nil {
		log.Printf("Failed to summarize paper: %v", err)
	} else {
		fmt.Printf("总结结果：\n%+v\n\n", summarizeResult)
	}
}
