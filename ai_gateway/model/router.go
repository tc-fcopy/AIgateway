package model

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	// GlobalModelRouter is shared by middleware and config loader.
	GlobalModelRouter = NewModelRouter()
)

// ModelRouter routes an input model to target model by rules.
type ModelRouter struct {
	enable       bool
	defaultModel string
	rules        []ModelRule
	lock         sync.RWMutex
}

// ModelRule defines one routing rule.
type ModelRule struct {
	Pattern     string `json:"pattern"`
	TargetModel string `json:"target_model"`
	Priority    int    `json:"priority"`
}

func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		enable:       true,
		defaultModel: "gpt-3.5-turbo",
		rules:        []ModelRule{},
	}
}

func (r *ModelRouter) SetConfig(enable bool, defaultModel string, rules []ModelRule) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.enable = enable
	if defaultModel != "" {
		r.defaultModel = defaultModel
	}

	copied := make([]ModelRule, len(rules))
	copy(copied, rules)
	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].Priority > copied[j].Priority
	})
	r.rules = copied
}

func (r *ModelRouter) Route(inputModel string) string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if !r.enable {
		return inputModel
	}

	if inputModel == "" || strings.EqualFold(inputModel, "auto") {
		return r.defaultModel
	}

	for _, rule := range r.rules {
		if r.matchPattern(rule.Pattern, inputModel) {
			return rule.TargetModel
		}
	}

	return r.defaultModel
}

func (r *ModelRouter) matchPattern(pattern, input string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// regexp match: ~<pattern>
	if strings.HasPrefix(pattern, "~") {
		regex, err := regexp.Compile(strings.TrimPrefix(pattern, "~"))
		if err != nil {
			return false
		}
		return regex.MatchString(input)
	}

	// prefix match: xxx*
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(input, prefix)
	}

	// exact match
	return pattern == input
}

func (r *ModelRouter) IsEnabled() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.enable
}

func (r *ModelRouter) GetDefaultModel() string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.defaultModel
}

func (r *ModelRouter) GetRules() []ModelRule {
	r.lock.RLock()
	defer r.lock.RUnlock()

	rules := make([]ModelRule, len(r.rules))
	copy(rules, r.rules)
	return rules
}
