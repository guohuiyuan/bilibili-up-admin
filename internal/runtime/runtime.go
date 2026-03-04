package runtime

import (
	"sync"

	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"
)

type Store struct {
	mu   sync.RWMutex
	bili *bilibili.Client
	llm  *llm.Manager
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Apply(bili *bilibili.Client, manager *llm.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bili = bili
	s.llm = manager
}

func (s *Store) BilibiliClient() *bilibili.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bili
}

func (s *Store) LLMManager() *llm.Manager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.llm
}
