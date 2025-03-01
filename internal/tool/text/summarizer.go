package text

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// SummarizerTool 实现论文总结工具
type SummarizerTool struct {
	id          string
	name        string
	description string
	version     string
	splitter    *TextSplitter
	llm         LLM
}

// LLM 定义语言模型接口
type LLM interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// NewSummarizerTool 创建新的总结工具
func NewSummarizerTool(llm LLM) *SummarizerTool {
	return &SummarizerTool{
		id:          "paper-summarizer",
		name:        "Paper Summarizer",
		description: "Summarize academic papers and extract key knowledge",
		version:     "1.0.0",
		splitter:    NewTextSplitter(DefaultSplitOptions()),
		llm:         llm,
	}
}

// ID 返回工具ID
func (t *SummarizerTool) ID() string {
	return t.id
}

// Name 返回工具名称
func (t *SummarizerTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *SummarizerTool) Description() string {
	return t.description
}

// Version 返回工具版本
func (t *SummarizerTool) Version() string {
	return t.version
}

// SummarizeParams 定义总结参数
type SummarizeParams struct {
	FilePath   string   // 文件路径
	MaxLength  int      // 最大总结长度
	FocusAreas []string // 重点关注领域
	Language   string   // 输出语言
}

// Execute 执行总结
func (t *SummarizerTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 解析参数
	summarizeParams, err := t.parseParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	// 读取文件
	content, err := t.readFile(summarizeParams.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// 分割文本
	chunks := t.splitter.Split(content)

	// 处理每个块
	var chunkSummaries []string
	for i, chunk := range chunks {
		summary, err := t.summarizeChunk(ctx, chunk, summarizeParams)
		if err != nil {
			return nil, fmt.Errorf("failed to summarize chunk %d: %v", i, err)
		}
		chunkSummaries = append(chunkSummaries, summary)
	}

	// 合并所有总结
	finalSummary, err := t.mergeSummaries(ctx, chunkSummaries, summarizeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to merge summaries: %v", err)
	}

	return map[string]interface{}{
		"status":   "success",
		"summary":  finalSummary,
		"language": summarizeParams.Language,
	}, nil
}

// Validate 验证参数
func (t *SummarizerTool) Validate(params map[string]interface{}) error {
	_, err := t.parseParams(params)
	return err
}

// parseParams 解析参数
func (t *SummarizerTool) parseParams(params map[string]interface{}) (*SummarizeParams, error) {
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	maxLength := 2000 // 默认长度
	if ml, ok := params["max_length"].(int); ok {
		maxLength = ml
	}

	language := "zh" // 默认中文
	if lang, ok := params["language"].(string); ok {
		language = lang
	}

	var focusAreas []string
	if areas, ok := params["focus_areas"].([]string); ok {
		focusAreas = areas
	}

	return &SummarizeParams{
		FilePath:   filePath,
		MaxLength:  maxLength,
		FocusAreas: focusAreas,
		Language:   language,
	}, nil
}

// summarizeChunk 总结单个文本块
func (t *SummarizerTool) summarizeChunk(ctx context.Context, chunk string, params *SummarizeParams) (string, error) {
	prompt := t.buildChunkPrompt(chunk, params)
	fmt.Println("chunk prompt:", prompt)
	response, err := t.llm.Complete(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to summarize chunk: %v", err)
	}
	fmt.Println("chunk response:", response)
	return response, nil
}

// mergeSummaries 合并所有总结
func (t *SummarizerTool) mergeSummaries(ctx context.Context, summaries []string, params *SummarizeParams) (string, error) {
	prompt := t.buildMergePrompt(summaries, params)
	fmt.Println("merge prompt:", prompt)
	response, err := t.llm.Complete(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to merge summaries: %v", err)
	}
	fmt.Println("merge response:", response)
	return response, nil
}

// buildChunkPrompt 构建块总结提示
func (t *SummarizerTool) buildChunkPrompt(chunk string, params *SummarizeParams) string {
	var prompt strings.Builder

	prompt.WriteString("请总结以下学术文本段落的主要内容。")
	if len(params.FocusAreas) > 0 {
		prompt.WriteString(fmt.Sprintf("\n请特别关注以下方面：%s", strings.Join(params.FocusAreas, "、")))
	}
	prompt.WriteString("\n\n原文：\n")
	prompt.WriteString(chunk)
	prompt.WriteString("\n\n请提供一个简洁的总结，重点突出关键发现、方法和结论。")

	return prompt.String()
}

// buildMergePrompt 构建合并总结提示
func (t *SummarizerTool) buildMergePrompt(summaries []string, params *SummarizeParams) string {
	var prompt strings.Builder

	prompt.WriteString("请将以下分段总结合并成一个连贯的整体总结。")
	prompt.WriteString(fmt.Sprintf("\n要求：\n1. 总结长度控制在%d字以内", params.MaxLength))
	if len(params.FocusAreas) > 0 {
		prompt.WriteString(fmt.Sprintf("\n2. 重点关注以下方面：%s", strings.Join(params.FocusAreas, "、")))
	}
	prompt.WriteString("\n\n分段总结：\n")

	for i, summary := range summaries {
		prompt.WriteString(fmt.Sprintf("\n第%d部分：\n%s", i+1, summary))
	}

	prompt.WriteString("\n\n请提供一个完整的总结，确保内容连贯、重点突出，并保持学术性。")

	return prompt.String()
}

// readFile 读取文件内容
func (t *SummarizerTool) readFile(path string) (string, error) {
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
