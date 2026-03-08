package http_proxy_plugin

// Phase declares the logical execution stage for a plugin.
type Phase int

const (
	PhasePreflight Phase = iota
	PhaseEdgeGuard
	PhaseAuthN
	PhasePolicy
	PhaseTransform
	PhaseTraffic
	PhaseObserve
	PhaseProxy
)

func (p Phase) String() string {
	switch p {
	case PhasePreflight:
		return "preflight"
	case PhaseEdgeGuard:
		return "edge_guard"
	case PhaseAuthN:
		return "authn"
	case PhasePolicy:
		return "policy"
	case PhaseTransform:
		return "transform"
	case PhaseTraffic:
		return "traffic"
	case PhaseObserve:
		return "observe"
	case PhaseProxy:
		return "proxy"
	default:
		return "unknown"
	}
}

// Plugin is the unified execution contract for all gateway capabilities.
type Plugin interface {
	Name() string
	Phase() Phase
	Priority() int
	Requires() []string
	Enabled(*ExecContext) bool
	Execute(*ExecContext) Result
}
