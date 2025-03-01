package text

import (
	"strings"
	"unicode"
)

// SplitOptions 定义文本分割的选项
type SplitOptions struct {
	ChunkSize        int  // 每个块的目标大小（字符数）
	ChunkOverlap     int  // 块之间的重叠大小
	SplitByParagraph bool // 是否按段落分割
	SplitBySentence  bool // 是否按句子分割
}

// DefaultSplitOptions 返回默认的分割选项
func DefaultSplitOptions() SplitOptions {
	return SplitOptions{
		ChunkSize:        1000,
		ChunkOverlap:     200,
		SplitByParagraph: true,
		SplitBySentence:  true,
	}
}

// TextSplitter 文本分割器
type TextSplitter struct {
	options SplitOptions
}

// NewTextSplitter 创建新的文本分割器
func NewTextSplitter(options SplitOptions) *TextSplitter {
	return &TextSplitter{
		options: options,
	}
}

// Split 将文本分割成多个块
func (ts *TextSplitter) Split(text string) []string {
	var chunks []string

	// 首先按段落分割
	if ts.options.SplitByParagraph {
		paragraphs := ts.splitIntoParagraphs(text)
		chunks = ts.mergeParagraphsIntoChunks(paragraphs)
	} else {
		// 如果不按段落分割，直接按大小分割
		chunks = ts.splitBySize(text)
	}

	return chunks
}

// splitIntoParagraphs 将文本分割成段落
func (ts *TextSplitter) splitIntoParagraphs(text string) []string {
	// 处理不同的换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 按连续的换行符分割
	paragraphs := strings.Split(text, "\n\n")

	// 清理每个段落
	var cleaned []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			if ts.options.SplitBySentence {
				sentences := ts.splitIntoSentences(p)
				cleaned = append(cleaned, sentences...)
			} else {
				cleaned = append(cleaned, p)
			}
		}
	}

	return cleaned
}

// splitIntoSentences 将文本分割成句子
func (ts *TextSplitter) splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)

		// 检查是否是句子结束符
		if r == '.' || r == '!' || r == '?' {
			// 查看下一个字符是否是空格或结束
			if len(sentences) > 0 && unicode.IsSpace(rune(text[len(text)-1])) {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	// 添加最后一个句子
	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}

// mergeParagraphsIntoChunks 将段落合并成指定大小的块
func (ts *TextSplitter) mergeParagraphsIntoChunks(paragraphs []string) []string {
	var chunks []string
	var currentChunk strings.Builder

	for i := 0; i < len(paragraphs); i++ {
		if currentChunk.Len()+len(paragraphs[i]) > ts.options.ChunkSize {
			// 如果当前块已经足够大，保存它
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}
		}

		// 添加新段落
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(paragraphs[i])
	}

	// 添加最后一个块
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

// splitBySize 直接按大小分割文本
func (ts *TextSplitter) splitBySize(text string) []string {
	var chunks []string

	for len(text) > 0 {
		chunkSize := ts.options.ChunkSize
		if len(text) < chunkSize {
			chunkSize = len(text)
		}

		chunk := text[:chunkSize]
		chunks = append(chunks, chunk)

		// 如果还有剩余文本，考虑重叠部分
		if len(text) > chunkSize {
			text = text[chunkSize-ts.options.ChunkOverlap:]
		} else {
			break
		}
	}

	return chunks
}
