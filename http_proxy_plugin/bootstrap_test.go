package http_proxy_plugin

import "testing"

func TestRegisterBuiltinPluginsTo(t *testing.T) {
	r := NewRegistry()
	if err := RegisterBuiltinPluginsTo(r); err != nil {
		t.Fatalf("register builtins failed: %v", err)
	}
	if r.Count() == 0 {
		t.Fatalf("expected builtins in registry")
	}

	if _, ok := r.Get(PluginProxyReverseProxy); !ok {
		t.Fatalf("expected %s registered", PluginProxyReverseProxy)
	}
	if _, ok := r.Get(PluginAIAuth); !ok {
		t.Fatalf("expected %s registered", PluginAIAuth)
	}

	// duplicate register on same registry should fail by design.
	if err := RegisterBuiltinPluginsTo(r); err == nil {
		t.Fatalf("expected duplicate register error")
	}
}

func TestRegisterBuiltinPluginsToNilRegistry(t *testing.T) {
	if err := RegisterBuiltinPluginsTo(nil); err == nil {
		t.Fatalf("expected nil registry error")
	}
}
