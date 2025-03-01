package text

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPDFReader_Read(t *testing.T) {
	// 创建临时测试文件
	testPDFPath := filepath.Join(os.TempDir(), "test.pdf")
	defer os.Remove(testPDFPath)

	// 初始化 PDF 读取器
	reader := NewPDFReader()
	reader.SetLogLevel(LogLevelWarn) // 设置日志级别为警告

	// 测试文件支持检查
	if !reader.IsFileSupported(testPDFPath) {
		t.Error("Expected PDF file to be supported")
	}

	// 测试实际的 PDF 文件
	realPDFPath := "C:\\Users\\boringsoft\\Downloads\\humlum-vestergaard-2024-the-unequal-adoption-of-chatgpt-exacerbates-existing-inequalities-among-workers.pdf"
	if _, err := os.Stat(realPDFPath); err == nil {
		content, err := reader.Read(realPDFPath)
		if err != nil {
			t.Errorf("Failed to read PDF file: %v", err)
		}
		if content == "" {
			t.Error("Expected non-empty content from PDF file")
		}

		// 测试文本清理
		if strings.Contains(content, "  ") {
			t.Error("Expected no consecutive spaces in cleaned text")
		}

		// 测试文件信息获取
		info, err := reader.GetFileInfo(realPDFPath)
		if err != nil {
			t.Errorf("Failed to get file info: %v", err)
		}

		// 检查必需的字段
		requiredFields := []string{"page_count", "file_size", "format"}
		for _, field := range requiredFields {
			if info[field] == nil {
				t.Errorf("Expected %s in file info", field)
			}
		}

		// 检查元数据字段
		metadataFields := []string{"title", "author", "subject", "keywords", "creator", "producer", "encryption"}
		hasMetadata := false
		for _, field := range metadataFields {
			if info[field] != nil {
				hasMetadata = true
				break
			}
		}
		if !hasMetadata {
			t.Error("Expected at least one metadata field to be present")
		}
	} else {
		t.Skip("Test PDF file not found, skipping test")
	}
}

func TestPDFReader_LogLevels(t *testing.T) {
	reader := NewPDFReader()

	// 测试默认日志级别
	if reader.logLevel != LogLevelError {
		t.Error("Expected default log level to be LogLevelError")
	}

	// 测试设置日志级别
	reader.SetLogLevel(LogLevelInfo)
	if reader.logLevel != LogLevelInfo {
		t.Error("Expected log level to be LogLevelInfo after setting")
	}
}

func TestPDFReader_CleanText(t *testing.T) {
	reader := NewPDFReader()

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "This   is  a    test",
			expected: "This is a test",
		},
		{
			input:    "First sentence. Second sentence",
			expected: "First sentence.\nSecond sentence",
		},
		{
			input:    "Multiple.   Spaces.   Between.   Sentences",
			expected: "Multiple.\nSpaces.\nBetween.\nSentences",
		},
	}

	for _, tc := range testCases {
		result := reader.cleanText(tc.input)
		if result != tc.expected {
			t.Errorf("cleanText(%q) = %q; want %q", tc.input, result, tc.expected)
		}
	}
}
