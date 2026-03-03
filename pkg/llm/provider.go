package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Response struct {
	Content      string `json:"content"`
	TokensUsed   int    `json:"tokens_used"`
	PromptTokens int    `json:"prompt_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	Model        string `json:"model"`
	FinishReason string `json:"finish_reason"`
}

type StreamCallback func(chunk string) error

type Provider interface {
	Chat(ctx context.Context, messages []Message) (*Response, error)
	ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (*Response, error)
	Stream(ctx context.Context, messages []Message, callback StreamCallback) error
	StreamWithSystem(ctx context.Context, systemPrompt string, messages []Message, callback StreamCallback) error
	Name() string
	Models() []string
}

type ProviderType string

const (
	ProviderOpenAI   ProviderType = "openai"
	ProviderClaude   ProviderType = "claude"
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderQwen     ProviderType = "qwen"
	ProviderMoonshot ProviderType = "moonshot"
	ProviderZhipu    ProviderType = "zhipu"
	ProviderOllama   ProviderType = "ollama"
	ProviderGemini   ProviderType = "gemini"
)

type Config struct {
	Provider    ProviderType `json:"provider"`
	APIKey      string       `json:"api_key"`
	BaseURL     string       `json:"base_url"`
	Model       string       `json:"model"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
}
