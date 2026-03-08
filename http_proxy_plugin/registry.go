package http_proxy_plugin

import (
	"errors"
	"sort"
	"sync"
)

var (
	ErrNilPlugin       = errors.New("plugin is nil")
	ErrEmptyPluginName = errors.New("plugin name is required")
	ErrPluginExists    = errors.New("plugin already registered")
	ErrRegistryNil     = errors.New("registry is nil")
	ErrPluginNotFound  = errors.New("plugin not found")
)

// Meta describes plugin registry metadata for debug/audit output.
type Meta struct {
	Name      string   `json:"name"`
	Phase     string   `json:"phase"`
	Priority  int      `json:"priority"`
	Requires  []string `json:"requires"`
	HasEnable bool     `json:"has_enable"`
}

// Registry stores all plugins globally.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{
		plugins: map[string]Plugin{},
		order:   make([]string, 0, 32),
	}
}

var GlobalRegistry = NewRegistry()

func (r *Registry) Register(p Plugin) error {
	if r == nil {
		return ErrRegistryNil
	}
	if p == nil {
		return ErrNilPlugin
	}
	name := p.Name()
	if name == "" {
		return ErrEmptyPluginName
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.plugins[name]; exists {
		return ErrPluginExists
	}
	r.plugins[name] = p
	r.order = append(r.order, name)
	return nil
}

func (r *Registry) MustRegister(p Plugin) {
	if err := r.Register(p); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(name string) (Plugin, bool) {
	if r == nil || name == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) MustGet(name string) Plugin {
	p, ok := r.Get(name)
	if !ok {
		panic(ErrPluginNotFound)
	}
	return p
}

func (r *Registry) List() []Plugin {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Plugin, 0, len(r.order))
	for _, name := range r.order {
		if p, ok := r.plugins[name]; ok {
			out = append(out, p)
		}
	}
	return out
}

func (r *Registry) ListMeta() []Meta {
	list := r.List()
	out := make([]Meta, 0, len(list))
	for _, p := range list {
		requires := p.Requires()
		cp := make([]string, len(requires))
		copy(cp, requires)
		sort.Strings(cp)
		out = append(out, Meta{
			Name:      p.Name(),
			Phase:     p.Phase().String(),
			Priority:  p.Priority(),
			Requires:  cp,
			HasEnable: true,
		})
	}
	return out
}

func (r *Registry) Count() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plugins)
}

// Reset clears registry state. Intended for tests/bootstrap reinitialization.
func (r *Registry) Reset() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.plugins = map[string]Plugin{}
	r.order = make([]string, 0, 32)
	r.mu.Unlock()
}
