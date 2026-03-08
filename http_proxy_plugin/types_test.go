package http_proxy_plugin

import "testing"

func TestPhaseString(t *testing.T) {
	cases := map[Phase]string{
		PhasePreflight: "preflight",
		PhaseEdgeGuard: "edge_guard",
		PhaseAuthN:     "authn",
		PhasePolicy:    "policy",
		PhaseTransform: "transform",
		PhaseTraffic:   "traffic",
		PhaseObserve:   "observe",
		PhaseProxy:     "proxy",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Fatalf("phase %v string mismatch, got=%s want=%s", in, got, want)
		}
	}
	if got := Phase(99).String(); got != "unknown" {
		t.Fatalf("expected unknown for invalid phase, got=%s", got)
	}
}
