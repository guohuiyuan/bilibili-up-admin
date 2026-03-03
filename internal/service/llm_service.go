package service

import (
	"context"
	"time"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	"bilibili-up-admin/pkg/llm"
)

type LLMService struct {
	manager    *llm.Manager
	llmLogRepo *repository.LLMChatLogRepository
}

func NewLLMService(manager *llm.Manager, llmLogRepo *repository.LLMChatLogRepository) *LLMService {
	return &LLMService{
		manager:    manager,
		llmLogRepo: llmLogRepo,
	}
}

func (s *LLMService) Chat(ctx context.Context, provider string, messages []llm.Message) (*llm.Response, error) {
	if s.manager == nil {
		return nil, nil
	}
	if provider != "" {
		p, err := s.manager.Get(provider)
		if err != nil {
			return nil, err
		}
		return p.Chat(ctx, messages)
	}
	return s.manager.Chat(ctx, messages)
}

func (s *LLMService) ChatWithSystem(ctx context.Context, provider, systemPrompt string, messages []llm.Message) (*llm.Response, error) {
	if s.manager == nil {
		return nil, nil
	}
	if provider != "" {
		p, err := s.manager.Get(provider)
		if err != nil {
			return nil, err
		}
		return p.ChatWithSystem(ctx, systemPrompt, messages)
	}
	return s.manager.ChatWithSystem(ctx, systemPrompt, messages)
}

func (s *LLMService) GetProviders() []string {
	if s.manager == nil {
		return nil
	}
	return s.manager.Names()
}

func (s *LLMService) GetDefaultProvider() string {
	if s.manager == nil {
		return ""
	}
	return s.manager.DefaultName()
}

func (s *LLMService) SetDefaultProvider(name string) {
	if s.manager != nil {
		s.manager.SetDefault(name)
	}
}

func (s *LLMService) GetStats(ctx context.Context, days int) (map[string]interface{}, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -days)
	return s.llmLogRepo.GetStats(ctx, start, end)
}

func (s *LLMService) TestProvider(ctx context.Context, provider string) (bool, string) {
	if s.manager == nil {
		return false, "llm manager is not configured"
	}
	p, err := s.manager.Get(provider)
	if err != nil {
		return false, err.Error()
	}

	resp, err := p.Chat(ctx, []llm.Message{{Role: "user", Content: "hello"}})
	if err != nil {
		return false, err.Error()
	}
	return true, resp.Content
}

func (s *LLMService) LogChat(ctx context.Context, log *model.LLMChatLog) error {
	return s.llmLogRepo.Create(ctx, log)
}
