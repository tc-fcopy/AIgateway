package model

import (
	"regexp"
	"strings"
	"sync"
)

var (
	// GlobalModelMapper is shared by middleware and config loader.
	GlobalModelMapper = NewModelMapper()
)

// ModelMapper maps requested model names to target model names.
type ModelMapper struct {
	enable   bool
	mappings map[string]string // source -> target
	lock     sync.RWMutex
}

// ModelMapping defines one mapping rule.
type ModelMapping struct {
	Source string `mapstructure:"source"`
	Target string `mapstructure:"target"`
}

func NewModelMapper() *ModelMapper {
	return &ModelMapper{
		enable:   true,
		mappings: make(map[string]string),
	}
}

func (m *ModelMapper) SetConfig(mappings []ModelMapping, enable bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.enable = enable
	m.mappings = make(map[string]string)
	for _, mapping := range mappings {
		if mapping.Source == "" {
			continue
		}
		m.mappings[mapping.Source] = mapping.Target
	}
}

func (m *ModelMapper) MapModel(inputModel string) string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if !m.enable || inputModel == "" {
		return inputModel
	}

	// 1. exact match
	if target, ok := m.mappings[inputModel]; ok {
		return target
	}

	// 2. prefix match: gpt-* style
	for source, target := range m.mappings {
		if strings.HasSuffix(source, "*") {
			prefix := strings.TrimSuffix(source, "*")
			if strings.HasPrefix(inputModel, prefix) {
				return target
			}
		}
	}

	// 3. wildcard match
	if target, ok := m.mappings["*"]; ok {
		return target
	}

	// 4. regexp match: rule starts with "~"
	for source, target := range m.mappings {
		if strings.HasPrefix(source, "~") {
			pattern := strings.TrimPrefix(source, "~")
			re, err := regexp.Compile(pattern)
			if err == nil && re.MatchString(inputModel) {
				return target
			}
		}
	}

	return inputModel
}

func (m *ModelMapper) IsEnabled() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.enable
}

func (m *ModelMapper) GetMappings() map[string]string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	mappings := make(map[string]string, len(m.mappings))
	for k, v := range m.mappings {
		mappings[k] = v
	}
	return mappings
}

func (m *ModelMapper) AddMapping(source, target string) error {
	if source == "" {
		return nil
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	m.mappings[source] = target
	return nil
}

func (m *ModelMapper) RemoveMapping(source string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.mappings, source)
}
