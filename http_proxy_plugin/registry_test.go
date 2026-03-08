package http_proxy_plugin

import (
	"testing"
)

type testPlugin struct {
	name     string
	phase    Phase
	priority int
	requires []string
}

func (t *testPlugin) Name() string                { return t.name }
func (t *testPlugin) Phase() Phase                { return t.phase }
func (t *testPlugin) Priority() int               { return t.priority }
func (t *testPlugin) Requires() []string          { return t.requires }
func (t *testPlugin) Enabled(*ExecContext) bool   { return true }
func (t *testPlugin) Execute(*ExecContext) Result { return Continue() }

func TestRegistryRegisterAndList(t *testing.T) {
	r := NewRegistry()
	p1 := &testPlugin{name: "p1", phase: PhasePolicy, priority: 10}
	p2 := &testPlugin{name: "p2", phase: PhaseProxy, priority: 20}

	if err := r.Register(p1); err != nil {
		t.Fatalf("register p1 failed: %v", err)
	}
	if err := r.Register(p2); err != nil {
		t.Fatalf("register p2 failed: %v", err)
	}

	if r.Count() != 2 {
		t.Fatalf("expected count=2, got %d", r.Count())
	}
	list := r.List()
	if len(list) != 2 || list[0].Name() != "p1" || list[1].Name() != "p2" {
		t.Fatalf("unexpected list order: %+v", list)
	}

	meta := r.ListMeta()
	if len(meta) != 2 || meta[0].Name != "p1" || meta[0].Phase != "policy" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := NewRegistry()
	p := &testPlugin{name: "dup", phase: PhasePolicy}
	if err := r.Register(p); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := r.Register(p); err == nil {
		t.Fatalf("expected duplicate error")
	}
}

func TestRegistryValidation(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(nil); err == nil {
		t.Fatalf("expected nil plugin error")
	}
	if err := r.Register(&testPlugin{}); err == nil {
		t.Fatalf("expected empty name error")
	}
}
