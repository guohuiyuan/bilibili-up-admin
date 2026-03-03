package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	ollamaapi "github.com/ollama/ollama/api"
	"google.golang.org/genai"
)

type DeepSeekProvider struct{ *OpenAIProvider }

func NewDeepSeekProvider(cfg *Config) (*DeepSeekProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	p, err := NewOpenAIProvider(cfg)
	if err != nil {
		return nil, err
	}
	p.name = ProviderDeepSeek
	return &DeepSeekProvider{OpenAIProvider: p}, nil
}

func (p *DeepSeekProvider) Models() []string {
	return []string{"deepseek-chat", "deepseek-reasoner"}
}

type QwenProvider struct{ *OpenAIProvider }

func NewQwenProvider(cfg *Config) (*QwenProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	p, err := NewOpenAIProvider(cfg)
	if err != nil {
		return nil, err
	}
	p.name = ProviderQwen
	return &QwenProvider{OpenAIProvider: p}, nil
}

func (p *QwenProvider) Models() []string {
	return []string{"qwen-turbo", "qwen-plus", "qwen-max"}
}

type MoonshotProvider struct{ *OpenAIProvider }

func NewMoonshotProvider(cfg *Config) (*MoonshotProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.moonshot.cn/v1"
	}
	p, err := NewOpenAIProvider(cfg)
	if err != nil {
		return nil, err
	}
	p.name = ProviderMoonshot
	return &MoonshotProvider{OpenAIProvider: p}, nil
}

func (p *MoonshotProvider) Models() []string {
	return []string{"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"}
}

type ZhipuProvider struct{ *OpenAIProvider }

func NewZhipuProvider(cfg *Config) (*ZhipuProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	p, err := NewOpenAIProvider(cfg)
	if err != nil {
		return nil, err
	}
	p.name = ProviderZhipu
	return &ZhipuProvider{OpenAIProvider: p}, nil
}

func (p *ZhipuProvider) Models() []string {
	return []string{"glm-4-flash", "glm-4-air", "glm-4-plus"}
}

type ClaudeProvider struct {
	client anthropic.Client
	config *Config
}

func NewClaudeProvider(cfg *Config) (*ClaudeProvider, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("api_key is required")
	}

	opts := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	return &ClaudeProvider{
		client: anthropic.NewClient(opts...),
		config: cfg,
	}, nil
}

func (p *ClaudeProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return p.ChatWithSystem(ctx, "", messages)
}

func (p *ClaudeProvider) ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (*Response, error) {
	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(defaultModel(p.config.Model, "claude-3-5-sonnet-latest")),
		MaxTokens: int64(p.config.MaxTokens),
		System:    toAnthropicSystem(systemPrompt),
		Messages:  toAnthropicMessages(messages),
	})
	if err != nil {
		return nil, fmt.Errorf("claude chat failed: %w", err)
	}

	content := anthropicText(resp.Content)
	return &Response{
		Content:      content,
		TokensUsed:   int(resp.Usage.OutputTokens),
		PromptTokens: int(resp.Usage.InputTokens),
		TotalTokens:  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		Model:        string(resp.Model),
		FinishReason: string(resp.StopReason),
	}, nil
}

func (p *ClaudeProvider) Stream(ctx context.Context, messages []Message, callback StreamCallback) error {
	return p.StreamWithSystem(ctx, "", messages, callback)
}

func (p *ClaudeProvider) StreamWithSystem(ctx context.Context, systemPrompt string, messages []Message, callback StreamCallback) error {
	stream := p.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(defaultModel(p.config.Model, "claude-3-5-sonnet-latest")),
		MaxTokens: int64(p.config.MaxTokens),
		System:    toAnthropicSystem(systemPrompt),
		Messages:  toAnthropicMessages(messages),
	})

	for stream.Next() {
		event := stream.Current()
		switch eventVariant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			if delta, ok := eventVariant.Delta.AsAny().(anthropic.TextDelta); ok && delta.Text != "" {
				if err := callback(delta.Text); err != nil {
					return err
				}
			}
		}
	}

	return stream.Err()
}

func (p *ClaudeProvider) Name() string { return string(ProviderClaude) }

func (p *ClaudeProvider) Models() []string {
	return []string{"claude-3-5-sonnet-latest", "claude-3-7-sonnet-latest", "claude-sonnet-4-5"}
}

type GeminiProvider struct {
	client *genai.Client
	config *Config
}

func NewGeminiProvider(cfg *Config) (*GeminiProvider, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &GeminiProvider{client: client, config: cfg}, nil
}

func (p *GeminiProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return p.ChatWithSystem(ctx, "", messages)
}

func (p *GeminiProvider) ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (*Response, error) {
	resp, err := p.client.Models.GenerateContent(
		ctx,
		defaultModel(p.config.Model, "gemini-2.5-flash"),
		toGeminiContents(messages),
		&genai.GenerateContentConfig{
			Temperature:       genai.Ptr(float32(p.config.Temperature)),
			MaxOutputTokens:   int32(p.config.MaxTokens),
			SystemInstruction: toGeminiSystem(systemPrompt),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini chat failed: %w", err)
	}

	var promptTokens, outputTokens, totalTokens int32
	if resp.UsageMetadata != nil {
		promptTokens = resp.UsageMetadata.PromptTokenCount
		outputTokens = resp.UsageMetadata.CandidatesTokenCount
		totalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return &Response{
		Content:      resp.Text(),
		TokensUsed:   int(outputTokens),
		PromptTokens: int(promptTokens),
		TotalTokens:  int(totalTokens),
		Model:        defaultModel(p.config.Model, "gemini-2.5-flash"),
		FinishReason: geminiFinishReason(resp),
	}, nil
}

func (p *GeminiProvider) Stream(ctx context.Context, messages []Message, callback StreamCallback) error {
	return p.StreamWithSystem(ctx, "", messages, callback)
}

func (p *GeminiProvider) StreamWithSystem(ctx context.Context, systemPrompt string, messages []Message, callback StreamCallback) error {
	for resp, err := range p.client.Models.GenerateContentStream(
		ctx,
		defaultModel(p.config.Model, "gemini-2.5-flash"),
		toGeminiContents(messages),
		&genai.GenerateContentConfig{
			Temperature:       genai.Ptr(float32(p.config.Temperature)),
			MaxOutputTokens:   int32(p.config.MaxTokens),
			SystemInstruction: toGeminiSystem(systemPrompt),
		},
	) {
		if err != nil {
			return err
		}
		text := resp.Text()
		if text != "" {
			if err := callback(text); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *GeminiProvider) Name() string { return string(ProviderGemini) }

func (p *GeminiProvider) Models() []string {
	return []string{"gemini-2.5-flash", "gemini-2.5-pro"}
}

type OllamaProvider struct {
	client *ollamaapi.Client
	config *Config
}

func NewOllamaProvider(cfg *Config) (*OllamaProvider, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &OllamaProvider{
		client: ollamaapi.NewClient(u, http.DefaultClient),
		config: cfg,
	}, nil
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return p.ChatWithSystem(ctx, "", messages)
}

func (p *OllamaProvider) ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (*Response, error) {
	var final ollamaapi.ChatResponse
	err := p.client.Chat(ctx, &ollamaapi.ChatRequest{
		Model:    defaultModel(p.config.Model, "llama3.1"),
		Messages: toOllamaMessages(systemPrompt, messages),
		Stream:   boolPtr(false),
		Options: map[string]any{
			"temperature": p.config.Temperature,
			"num_predict": p.config.MaxTokens,
		},
	}, func(resp ollamaapi.ChatResponse) error {
		final = resp
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	return &Response{
		Content:      final.Message.Content,
		Model:        final.Model,
		FinishReason: "stop",
	}, nil
}

func (p *OllamaProvider) Stream(ctx context.Context, messages []Message, callback StreamCallback) error {
	return p.StreamWithSystem(ctx, "", messages, callback)
}

func (p *OllamaProvider) StreamWithSystem(ctx context.Context, systemPrompt string, messages []Message, callback StreamCallback) error {
	return p.client.Chat(ctx, &ollamaapi.ChatRequest{
		Model:    defaultModel(p.config.Model, "llama3.1"),
		Messages: toOllamaMessages(systemPrompt, messages),
		Options: map[string]any{
			"temperature": p.config.Temperature,
			"num_predict": p.config.MaxTokens,
		},
	}, func(resp ollamaapi.ChatResponse) error {
		if resp.Message.Content == "" {
			return nil
		}
		return callback(resp.Message.Content)
	})
}

func (p *OllamaProvider) Name() string { return string(ProviderOllama) }

func (p *OllamaProvider) Models() []string {
	return []string{"llama3.1", "qwen2.5", "deepseek-r1"}
}

func toAnthropicSystem(systemPrompt string) []anthropic.TextBlockParam {
	if systemPrompt == "" {
		return nil
	}
	return []anthropic.TextBlockParam{{Text: systemPrompt}}
}

func toAnthropicMessages(messages []Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "assistant":
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		default:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	return out
}

func anthropicText(blocks []anthropic.ContentBlockUnion) string {
	var parts []string
	for _, block := range blocks {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "")
}

func toGeminiContents(messages []Message) []*genai.Content {
	out := make([]*genai.Content, 0, len(messages))
	for _, m := range messages {
		var role genai.Role = genai.RoleUser
		if m.Role == "assistant" {
			role = genai.RoleModel
		}
		out = append(out, genai.NewContentFromParts([]*genai.Part{{Text: m.Content}}, role))
	}
	return out
}

func toGeminiSystem(systemPrompt string) *genai.Content {
	if systemPrompt == "" {
		return nil
	}
	return genai.NewContentFromParts([]*genai.Part{{Text: systemPrompt}}, genai.RoleUser)
}

func geminiFinishReason(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0] == nil {
		return ""
	}
	return fmt.Sprint(resp.Candidates[0].FinishReason)
}

func toOllamaMessages(systemPrompt string, messages []Message) []ollamaapi.Message {
	out := make([]ollamaapi.Message, 0, len(messages)+1)
	if systemPrompt != "" {
		out = append(out, ollamaapi.Message{Role: "system", Content: systemPrompt})
	}
	for _, m := range messages {
		out = append(out, ollamaapi.Message{Role: m.Role, Content: m.Content})
	}
	return out
}

func boolPtr(v bool) *bool {
	return &v
}
