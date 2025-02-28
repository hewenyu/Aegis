package llm

import (
	"context"
	"math"
	"reflect"
	"testing"
)

func TestOllamaProvider(t *testing.T) {
	// 创建provider实例
	provider, err := NewOllamaProvider("http://localhost:11434")
	if err != nil {
		t.Fatalf("创建OllamaProvider失败: %v", err)
	}

	// 测试获取名称
	t.Run("测试获取提供者名称", func(t *testing.T) {
		name := provider.Name()
		if name != "ollama" {
			t.Errorf("期望名称为'ollama'，实际得到：%s", name)
		}
	})

	// 测试列出模型
	t.Run("测试列出可用模型", func(t *testing.T) {
		ctx := context.Background()
		models, err := provider.ListModels(ctx)
		if err != nil {
			t.Fatalf("列出模型失败: %v", err)
		}
		if len(models) == 0 {
			t.Error("没有找到可用模型")
		}
		// 打印找到的模型
		t.Logf("找到 %d 个模型", len(models))
		for _, model := range models {
			t.Logf("模型名称: %s", model.Name)
		}
	})

	// 获取第一个可用的模型名称
	ctx := context.Background()
	models, err := provider.ListModels(ctx)
	if err != nil || len(models) == 0 {
		t.Fatal("无法获取可用模型")
	}
	modelName := models[1].Name

	// 测试获取模型信息
	t.Run("测试获取模型信息", func(t *testing.T) {
		ctx := context.Background()
		modelInfo, err := provider.GetModel(ctx, modelName)
		if err != nil {
			t.Fatalf("获取模型信息失败: %v", err)
		}
		t.Logf("模型信息: %+v", modelInfo)
	})

	// 测试聊天功能
	t.Run("测试聊天功能", func(t *testing.T) {
		ctx := context.Background()
		request := ChatRequest{
			Messages: []Message{
				{
					Role:    "user",
					Content: "你好，请用一句话介绍自己",
				},
			},
			Temperature: 0.7,
		}

		response, err := provider.Chat(ctx, modelName, request)
		if err != nil {
			t.Fatalf("聊天请求失败: %v", err)
		}

		if response.Message.Content == "" {
			t.Error("聊天响应内容为空")
		} else {
			t.Logf("聊天响应: %s", response.Message.Content)
			t.Logf("Token使用情况: 提示词=%d, 补全=%d, 总计=%d",
				response.Usage.PromptTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens)
		}
	})

	// 测试文本补全
	t.Run("测试文本补全", func(t *testing.T) {
		ctx := context.Background()
		request := CompletionRequest{
			Prompt:      "从前有座山，山里有座庙，庙里有个",
			Temperature: 0.7,
		}

		response, err := provider.Complete(ctx, modelName, request)
		if err != nil {
			t.Fatalf("文本补全请求失败: %v", err)
		}

		if response.Text == "" {
			t.Error("补全响应内容为空")
		} else {
			t.Logf("补全响应: %s", response.Text)
			t.Logf("Token使用情况: 提示词=%d, 补全=%d, 总计=%d",
				response.Usage.PromptTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens)
		}
	})

	// 测试文本嵌入
	t.Run("测试文本嵌入", func(t *testing.T) {
		ctx := context.Background()

		testCases := []struct {
			name  string
			input string
		}{
			{
				name:  "短文本",
				input: "这是一个测试文本",
			},
			{
				name:  "长文本",
				input: "向量数据库是一种专门设计用于存储和高效检索向量（嵌入）的数据库系统。它可以用来存储文本、图像等数据的向量表示，并支持相似度搜索。",
			},
			{
				name:  "特殊字符",
				input: "测试!@#$%^&*()_+ 😊 换行\n制表符\t",
			},
			{
				name:  "多语言",
				input: "中文 English 日本語 한국어",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				request := EmbeddingRequest{
					Input: tc.input,
				}

				response, err := provider.Embed(ctx, modelName, request)
				if err != nil {
					t.Fatalf("生成嵌入向量失败: %v", err)
				}

				// 验证向量维度
				if len(response.Embedding) == 0 {
					t.Error("嵌入向量为空")
				}

				// 验证向量值范围
				for i, v := range response.Embedding {
					if math.IsNaN(v) || math.IsInf(v, 0) {
						t.Errorf("向量第%d个元素是无效值: %v", i, v)
					}
				}

				t.Logf("输入文本长度: %d", len(tc.input))
				t.Logf("嵌入向量维度: %d", len(response.Embedding))
				t.Logf("Token使用情况: 提示词=%d, 总计=%d",
					response.Usage.PromptTokens,
					response.Usage.TotalTokens)

				// 验证相同输入得到相同向量
				response2, err := provider.Embed(ctx, modelName, request)
				if err != nil {
					t.Fatalf("重复生成嵌入向量失败: %v", err)
				}

				if !reflect.DeepEqual(response.Embedding, response2.Embedding) {
					t.Error("相同输入得到不同的嵌入向量")
				}
			})
		}
	})
}

// TestOllamaProviderError 测试错误情况
func TestOllamaProviderError(t *testing.T) {
	// 测试无效的模型名称
	t.Run("测试无效的模型名称", func(t *testing.T) {
		provider, err := NewOllamaProvider("http://localhost:11434")
		if err != nil {
			t.Fatalf("创建OllamaProvider失败: %v", err)
		}

		ctx := context.Background()
		_, err = provider.Chat(ctx, "不存在的模型", ChatRequest{
			Messages: []Message{
				{
					Role:    "user",
					Content: "你好",
				},
			},
		})
		if err == nil {
			t.Error("使用无效模型名称时期望得到错误，但没有")
		}
	})
}
