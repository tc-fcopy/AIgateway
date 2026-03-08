package model

import "testing"

func TestModelRouterRoute(t *testing.T) {
	r := NewModelRouter()
	r.SetConfig(true, "default-model", []ModelRule{{Pattern: "gpt-*", TargetModel: "mapped-model", Priority: 10}})

	out := r.Route("gpt-4")
	if out != "mapped-model" {
		t.Fatalf("expected mapped-model, got %s", out)
	}
}

func TestModelMapperMap(t *testing.T) {
	m := NewModelMapper()
	m.SetConfig([]ModelMapping{{Source: "gpt-*", Target: "qwen-turbo"}}, true)
	out := m.MapModel("gpt-4o")
	if out != "qwen-turbo" {
		t.Fatalf("expected qwen-turbo, got %s", out)
	}
}
