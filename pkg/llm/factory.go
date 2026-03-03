package llm

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

var ErrUnsupportedProvider = errors.New("unsupported provider")

func NewProvider(cfg *Config) (Provider, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 1000
	}

	switch cfg.Provider {
	case ProviderOpenAI:
		return NewOpenAIProvider(cfg)
	case ProviderClaude:
		return NewClaudeProvider(cfg)
	case ProviderDeepSeek:
		return NewDeepSeekProvider(cfg)
	case ProviderQwen:
		return NewQwenProvider(cfg)
	case ProviderMoonshot:
		return NewMoonshotProvider(cfg)
	case ProviderZhipu:
		return NewZhipuProvider(cfg)
	case ProviderOllama:
		return NewOllamaProvider(cfg)
	case ProviderGemini:
		return NewGeminiProvider(cfg)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, cfg.Provider)
	}
}

type ProviderConstructor func(*Config) (Provider, error)

type ProviderRegistry struct {
	providers map[ProviderType]ProviderConstructor
}

func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{providers: make(map[ProviderType]ProviderConstructor)}
	r.Register(ProviderOpenAI, wrapTyped(NewOpenAIProvider))
	r.Register(ProviderClaude, wrapTyped(NewClaudeProvider))
	r.Register(ProviderDeepSeek, wrapTyped(NewDeepSeekProvider))
	r.Register(ProviderQwen, wrapTyped(NewQwenProvider))
	r.Register(ProviderMoonshot, wrapTyped(NewMoonshotProvider))
	r.Register(ProviderZhipu, wrapTyped(NewZhipuProvider))
	r.Register(ProviderOllama, wrapTyped(NewOllamaProvider))
	r.Register(ProviderGemini, wrapTyped(NewGeminiProvider))
	return r
}

func wrapTyped[T Provider](fn func(*Config) (T, error)) ProviderConstructor {
	return func(cfg *Config) (Provider, error) {
		return fn(cfg)
	}
}

func (r *ProviderRegistry) Register(pt ProviderType, constructor ProviderConstructor) {
	r.providers[pt] = constructor
}

func (r *ProviderRegistry) Create(cfg *Config) (Provider, error) {
	constructor, ok := r.providers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, cfg.Provider)
	}
	return constructor(cfg)
}

var GlobalRegistry = NewProviderRegistry()

type Manager struct {
	registry  *ProviderRegistry
	providers map[string]Provider
	defaults  string
}

func NewManager() *Manager {
	return &Manager{
		registry:  GlobalRegistry,
		providers: make(map[string]Provider),
	}
}

func (m *Manager) Register(name string, provider Provider) {
	m.providers[name] = provider
	if m.defaults == "" {
		m.defaults = name
	}
}

func (m *Manager) CreateAndRegister(name string, cfg *Config) error {
	provider, err := m.registry.Create(cfg)
	if err != nil {
		return err
	}
	m.Register(name, provider)
	return nil
}

func (m *Manager) Get(name string) (Provider, error) {
	if name == "" {
		name = m.defaults
	}
	provider, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

func (m *Manager) Default() (Provider, error) {
	return m.Get(m.defaults)
}

func (m *Manager) SetDefault(name string) {
	m.defaults = name
}

func (m *Manager) DefaultName() string {
	return m.defaults
}

func (m *Manager) Names() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *Manager) Chat(ctx context.Context, messages []Message) (*Response, error) {
	provider, err := m.Default()
	if err != nil {
		return nil, err
	}
	return provider.Chat(ctx, messages)
}

func (m *Manager) ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (*Response, error) {
	provider, err := m.Default()
	if err != nil {
		return nil, err
	}
	return provider.ChatWithSystem(ctx, systemPrompt, messages)
}
