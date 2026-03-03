package llm

import (
	"context"
	"errors"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	name   ProviderType
	client *openai.Client
	config *Config
}

func NewOpenAIProvider(cfg *Config) (*OpenAIProvider, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("api_key is required")
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	return &OpenAIProvider{
		name:   ProviderOpenAI,
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}, nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return p.ChatWithSystem(ctx, "", messages)
}

func (p *OpenAIProvider) ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (*Response, error) {
	req := openai.ChatCompletionRequest{
		Model:       defaultModel(p.config.Model, "gpt-4o-mini"),
		Messages:    toOpenAIMessages(systemPrompt, messages),
		MaxTokens:   p.config.MaxTokens,
		Temperature: float32(p.config.Temperature),
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s chat failed: %w", p.name, err)
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("no response choices")
	}

	return &Response{
		Content:      resp.Choices[0].Message.Content,
		TokensUsed:   resp.Usage.CompletionTokens,
		PromptTokens: resp.Usage.PromptTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		Model:        resp.Model,
		FinishReason: string(resp.Choices[0].FinishReason),
	}, nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, messages []Message, callback StreamCallback) error {
	return p.StreamWithSystem(ctx, "", messages, callback)
}

func (p *OpenAIProvider) StreamWithSystem(ctx context.Context, systemPrompt string, messages []Message, callback StreamCallback) error {
	req := openai.ChatCompletionRequest{
		Model:       defaultModel(p.config.Model, "gpt-4o-mini"),
		Messages:    toOpenAIMessages(systemPrompt, messages),
		MaxTokens:   p.config.MaxTokens,
		Temperature: float32(p.config.Temperature),
		Stream:      true,
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("%s stream failed: %w", p.name, err)
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("stream recv failed: %w", err)
		}
		if len(response.Choices) == 0 {
			continue
		}

		chunk := response.Choices[0].Delta.Content
		if chunk != "" {
			if err := callback(chunk); err != nil {
				return err
			}
		}
	}
}

func (p *OpenAIProvider) Name() string {
	return string(p.name)
}

func (p *OpenAIProvider) Models() []string {
	return []string{"gpt-4o", "gpt-4o-mini", "o1", "o1-mini"}
}

func toOpenAIMessages(systemPrompt string, messages []Message) []openai.ChatCompletionMessage {
	out := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	if systemPrompt != "" {
		out = append(out, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}
	for _, m := range messages {
		out = append(out, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return out
}

func defaultModel(model, fallback string) string {
	if model != "" {
		return model
	}
	return fallback
}
