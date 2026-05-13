package llm

import (
	"context"
	"sync"
)

// Registry holds providers in priority order. v1 uses the first provider.
type Registry struct {
	mu        sync.RWMutex
	providers []Provider
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry { return &Registry{} }

// Register appends a provider to the priority list.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, p)
}

// Primary returns the highest-priority provider.
func (r *Registry) Primary() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.providers) == 0 {
		return nil, ErrNoProviders
	}
	return r.providers[0], nil
}

// Chat delegates to the primary provider.
func (r *Registry) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	p, err := r.Primary()
	if err != nil {
		return ChatResponse{}, err
	}
	return p.Chat(ctx, req)
}
