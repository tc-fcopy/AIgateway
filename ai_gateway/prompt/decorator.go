package prompt

import (
	"strings"
	"sync"
)

var (
	GlobalPromptDecorator = NewPromptDecorator()
)

// PromptDecorator decorates incoming user prompt text.
type PromptDecorator struct {
	lock         sync.RWMutex
	systemPrefix string
	systemSuffix string
	userPrefix   string
	userSuffix   string
	enable       bool
}

func NewPromptDecorator() *PromptDecorator {
	return &PromptDecorator{enable: true}
}

func (p *PromptDecorator) SetConfig(enable bool, systemPrefix, systemSuffix, userPrefix, userSuffix string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.enable = enable
	p.systemPrefix = systemPrefix
	p.systemSuffix = systemSuffix
	p.userPrefix = userPrefix
	p.userSuffix = userSuffix
}

func (p *PromptDecorator) Decorate(input, _ string) (string, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if !p.enable {
		return input, nil
	}

	var b strings.Builder
	if p.systemPrefix != "" {
		b.WriteString(p.systemPrefix)
		b.WriteString("\n")
	}
	if p.userPrefix != "" {
		b.WriteString(p.userPrefix)
	}
	b.WriteString(input)
	if p.userSuffix != "" {
		b.WriteString(p.userSuffix)
	}
	if p.systemSuffix != "" {
		b.WriteString("\n")
		b.WriteString(p.systemSuffix)
	}

	return b.String(), nil
}
