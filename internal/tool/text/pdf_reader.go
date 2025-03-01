package text

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gen2brain/go-fitz"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	// LogLevelError 只记录错误
	LogLevelError LogLevel = iota
	// LogLevelWarn 记录警告和错误
	LogLevelWarn
	// LogLevelInfo 记录所有信息
	LogLevelInfo
)

// PDFReader PDF文件读取器
type PDFReader struct {
	logLevel LogLevel
}

// NewPDFReader 创建新的PDF读取器
func NewPDFReader() *PDFReader {
	return &PDFReader{
		logLevel: LogLevelError, // 默认只记录错误
	}
}

// SetLogLevel 设置日志级别
func (r *PDFReader) SetLogLevel(level LogLevel) {
	r.logLevel = level
}

// logf 根据日志级别记录日志
func (r *PDFReader) logf(level LogLevel, format string, v ...interface{}) {
	if level <= r.logLevel {
		log.Printf(format, v...)
	}
}

// cleanText 清理提取的文本
func (r *PDFReader) cleanText(text string) string {
	// 移除多余的空白字符
	text = strings.Join(strings.Fields(text), " ")
	// 确保段落之间有适当的分隔
	text = strings.ReplaceAll(text, ". ", ".\n")
	return text
}

// Read 读取PDF文件内容
func (r *PDFReader) Read(filePath string) (string, error) {
	r.logf(LogLevelInfo, "开始读取PDF文件: %s", filePath)

	// 打开PDF文件
	doc, err := fitz.New(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer doc.Close()

	// 获取页数
	numPages := doc.NumPage()
	r.logf(LogLevelInfo, "PDF页数: %d", numPages)

	var content strings.Builder
	var warnings []string

	// 逐页读取内容
	for i := 0; i < numPages; i++ {
		r.logf(LogLevelInfo, "正在处理第 %d 页", i+1)

		// 捕获标准错误输出以处理警告
		oldStderr := os.Stderr
		rErr, wErr, _ := os.Pipe()
		os.Stderr = wErr

		// 提取文本
		text, err := doc.Text(i)

		// 恢复标准错误输出并读取警告
		wErr.Close()
		os.Stderr = oldStderr
		warningOutput, _ := io.ReadAll(rErr)
		rErr.Close()

		if len(warningOutput) > 0 {
			warning := strings.TrimSpace(string(warningOutput))
			if strings.Contains(warning, "invalid marked content and clip nesting") {
				r.logf(LogLevelWarn, "第 %d 页存在标记内容和剪裁嵌套警告，不影响文本提取", i+1)
			} else {
				warnings = append(warnings, fmt.Sprintf("第 %d 页: %s", i+1, warning))
			}
		}

		if err != nil {
			r.logf(LogLevelError, "提取第 %d 页文本失败: %v", i+1, err)
			continue
		}

		// 清理和处理文本
		text = r.cleanText(text)
		textLen := len(text)

		if textLen > 0 {
			r.logf(LogLevelInfo, "第 %d 页成功提取文本，长度: %d 字符", i+1, textLen)
			if textLen > 100 {
				r.logf(LogLevelInfo, "文本预览: %s...", text[:100])
			} else {
				r.logf(LogLevelInfo, "文本预览: %s", text)
			}
			content.WriteString(text)
			content.WriteString("\n\n")
		} else {
			r.logf(LogLevelWarn, "第 %d 页提取的文本为空", i+1)
		}
	}

	// 报告所有非标准警告
	if len(warnings) > 0 {
		r.logf(LogLevelWarn, "PDF处理警告:\n%s", strings.Join(warnings, "\n"))
	}

	result := content.String()
	r.logf(LogLevelInfo, "PDF文本提取完成，总长度: %d 字符", len(result))

	return result, nil
}

// IsFileSupported 检查文件是否是支持的PDF文件
func (r *PDFReader) IsFileSupported(filePath string) bool {
	return strings.ToLower(filePath[len(filePath)-4:]) == ".pdf"
}

// GetFileInfo 获取PDF文件信息
func (r *PDFReader) GetFileInfo(filePath string) (map[string]interface{}, error) {
	r.logf(LogLevelInfo, "开始获取PDF文件信息: %s", filePath)

	// 打开PDF文件
	doc, err := fitz.New(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer doc.Close()

	info := make(map[string]interface{})

	// 获取页数
	numPages := doc.NumPage()
	info["page_count"] = numPages

	// 获取文件大小
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		info["file_size"] = fileInfo.Size()
	}

	r.logf(LogLevelInfo, "PDF基本信息: 页数=%d, 大小=%d", info["page_count"], info["file_size"])

	// 获取PDF元数据
	metadata := doc.Metadata()
	if metadata != nil {
		r.logf(LogLevelInfo, "开始读取PDF元数据")
		if title, ok := metadata["format"]; ok && title != "" {
			info["format"] = title
			r.logf(LogLevelInfo, "格式: %s", info["format"])
		}
		if title, ok := metadata["title"]; ok && title != "" {
			info["title"] = title
			r.logf(LogLevelInfo, "标题: %s", info["title"])
		}
		if author, ok := metadata["author"]; ok && author != "" {
			info["author"] = author
			r.logf(LogLevelInfo, "作者: %s", info["author"])
		}
		if subject, ok := metadata["subject"]; ok && subject != "" {
			info["subject"] = subject
			r.logf(LogLevelInfo, "主题: %s", info["subject"])
		}
		if keywords, ok := metadata["keywords"]; ok && keywords != "" {
			info["keywords"] = keywords
			r.logf(LogLevelInfo, "关键词: %s", info["keywords"])
		}
		if creator, ok := metadata["creator"]; ok && creator != "" {
			info["creator"] = creator
			r.logf(LogLevelInfo, "创建工具: %s", info["creator"])
		}
		if producer, ok := metadata["producer"]; ok && producer != "" {
			info["producer"] = producer
			r.logf(LogLevelInfo, "生成工具: %s", info["producer"])
		}
		if encryption, ok := metadata["encryption"]; ok && encryption != "" {
			info["encryption"] = encryption
			r.logf(LogLevelInfo, "加密方式: %s", info["encryption"])
		}
	} else {
		r.logf(LogLevelWarn, "PDF文件没有元数据")
	}

	return info, nil
}
