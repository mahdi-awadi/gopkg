package provider

import (
	"fmt"
	"sync"
)

// Registry is a thread-safe registry of Provider instances keyed by Code().
//
// The zero value is not usable; call NewRegistry.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	byChannel map[Channel][]Provider
}

// NewRegistry returns an empty Registry ready for use.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		byChannel: make(map[Channel][]Provider),
	}
}

// Register adds p to the registry. Returns an error if a provider with the
// same Code() is already registered.
func (r *Registry) Register(p Provider) error {
	if p == nil {
		return fmt.Errorf("provider: nil")
	}
	code := p.Code()
	if code == "" {
		return fmt.Errorf("provider: empty Code")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[code]; exists {
		return fmt.Errorf("provider %q already registered", code)
	}
	r.providers[code] = p
	for _, ch := range p.SupportedChannels() {
		r.byChannel[ch] = append(r.byChannel[ch], p)
	}
	return nil
}

// Get returns the Provider with the given code, or nil if none exists.
func (r *Registry) Get(code string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[code]
}

// ByChannel returns all registered providers that support the given channel.
// The returned slice is a copy; callers may mutate it.
func (r *Registry) ByChannel(ch Channel) []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.byChannel[ch]
	out := make([]Provider, len(src))
	copy(out, src)
	return out
}

// Codes returns the sorted list of registered provider codes.
func (r *Registry) Codes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	return out
}

// Len returns the number of registered providers.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}
