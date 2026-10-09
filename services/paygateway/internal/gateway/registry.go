package gateway

import (
	"fmt"
	"strings"
)

type Registry struct {
	providers map[string]Provider
	defaultName string
}

func NewRegistry(defaultName string) *Registry {
	return &Registry{
		providers:   map[string]Provider{},
		defaultName: strings.ToLower(strings.TrimSpace(defaultName)),
	}
}

func (r *Registry) Register(p Provider) {
	if p == nil {
		return
	}
	r.providers[strings.ToLower(p.Name())] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return r.Default()
	}
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown_provider")
	}
	return p, nil
}

func (r *Registry) Default() (Provider, error) {
	if r.defaultName != "" {
		if p, ok := r.providers[r.defaultName]; ok && p.Configured() {
			return p, nil
		}
	}
	// Prefer first configured non-stub, then stub.
	var stub Provider
	for _, name := range []string{"razorpay", "cashfree"} {
		if p, ok := r.providers[name]; ok && p.Configured() {
			return p, nil
		}
	}
	for _, p := range r.providers {
		if p.Name() == "stub" {
			stub = p
			continue
		}
		if p.Configured() {
			return p, nil
		}
	}
	if stub != nil {
		return stub, nil
	}
	return nil, fmt.Errorf("no_provider")
}

func (r *Registry) List() []ProviderInfo {
	out := []ProviderInfo{}
	// Stable order
	for _, name := range []string{"stub", "razorpay", "cashfree"} {
		p, ok := r.providers[name]
		if !ok {
			continue
		}
		out = append(out, ProviderInfo{
			Name:       p.Name(),
			Configured: p.Configured(),
			IsDefault:  r.isDefault(p),
		})
	}
	for name, p := range r.providers {
		if name == "stub" || name == "razorpay" || name == "cashfree" {
			continue
		}
		out = append(out, ProviderInfo{
			Name:       p.Name(),
			Configured: p.Configured(),
			IsDefault:  r.isDefault(p),
		})
	}
	return out
}

func (r *Registry) isDefault(p Provider) bool {
	def, err := r.Default()
	if err != nil {
		return false
	}
	return def.Name() == p.Name()
}
