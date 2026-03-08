package consumer

import (
	"sync"
)

var (
	// ConsumerManager 全局Consumer管理器
	ConsumerManager *Manager
)

func init() {
	ConsumerManager = NewManager()
}

// Manager Consumer管理器
type Manager struct {
	consumerMap    map[string]*Consumer // name -> consumer
	credentialMap  map[string]*Consumer // credential -> consumer
	typeMap       map[string][]*Consumer // type -> consumers
	locker        sync.RWMutex
	init          sync.Once
	err           error
	loaded        bool
}

// NewManager 创建Consumer管理器
func NewManager() *Manager {
	return &Manager{
		consumerMap:   make(map[string]*Consumer),
		credentialMap: make(map[string]*Consumer),
		typeMap:      make(map[string][]*Consumer),
		locker:       sync.RWMutex{},
		init:         sync.Once{},
	}
}

// LoadOnce 加载Consumer数据（只执行一次）
// 注意：实际加载逻辑由DAO层调用，这里只标记已加载
func (m *Manager) LoadOnce() error {
	m.init.Do(func() {
		m.loaded = true
	})
	return m.err
}

// LoadConsumers 从外部加载Consumer列表（由DAO层调用）
func (m *Manager) LoadConsumers(consumers []*Consumer) {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.consumerMap = make(map[string]*Consumer)
	m.credentialMap = make(map[string]*Consumer)
	m.typeMap = make(map[string][]*Consumer)

	for _, cons := range consumers {
		if cons != nil && cons.Status == 1 {
			m.consumerMap[cons.Name] = cons
			m.credentialMap[cons.Credential] = cons
			m.typeMap[cons.Type] = append(m.typeMap[cons.Type], cons)
		}
	}

	m.loaded = true
}

// Reload 重新加载Consumer数据
func (m *Manager) Reload() error {
	// 清空当前缓存
	m.locker.Lock()
	m.consumerMap = make(map[string]*Consumer)
	m.credentialMap = make(map[string]*Consumer)
	m.typeMap = make(map[string][]*Consumer)
	m.loaded = false
	m.locker.Unlock()

	// 实际重新加载由DAO层控制
	return nil
}

// GetByName 根据名称获取Consumer
func (m *Manager) GetByName(name string) (*Consumer, bool) {
	m.locker.RLock()
	defer m.locker.RUnlock()
	c, ok := m.consumerMap[name]
	return c, ok
}

// GetByCredential 根据凭证获取Consumer
func (m *Manager) GetByCredential(credential string) (*Consumer, bool) {
	m.locker.RLock()
	defer m.locker.RUnlock()
	c, ok := m.credentialMap[credential]
	return c, ok
}

// GetByType 根据类型获取Consumer列表
func (m *Manager) GetByType(consumerType string) []*Consumer {
	m.locker.RLock()
	defer m.locker.RUnlock()
	list, ok := m.typeMap[consumerType]
	if !ok {
		return []*Consumer{}
	}
	return list
}

// GetAll 获取所有Consumer
func (m *Manager) GetAll() []*Consumer {
	m.locker.RLock()
	defer m.locker.RUnlock()

	list := make([]*Consumer, 0, len(m.consumerMap))
	for _, cons := range m.consumerMap {
		list = append(list, cons)
	}
	return list
}

// GetCount 获取Consumer总数
func (m *Manager) GetCount() int {
	m.locker.RLock()
	defer m.locker.RUnlock()
	return len(m.consumerMap)
}

// IsLoaded 检查是否已加载
func (m *Manager) IsLoaded() bool {
	m.locker.RLock()
	defer m.locker.RUnlock()
	return m.loaded
}

// Add 添加Consumer（用于动态添加）
func (m *Manager) Add(cons *Consumer) error {
	if err := m.Validate(cons); err != nil {
		return err
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	// 检查名称是否已存在
	if _, exists := m.consumerMap[cons.Name]; exists {
		return ErrorConsumerNameExists
	}

	// 检查凭证是否已存在
	if _, exists := m.credentialMap[cons.Credential]; exists {
		return ErrorCredentialExists
	}

	// 添加到maps
	m.consumerMap[cons.Name] = cons
	m.credentialMap[cons.Credential] = cons
	m.typeMap[cons.Type] = append(m.typeMap[cons.Type], cons)

	return nil
}

// Remove 移除Consumer（用于动态删除）
func (m *Manager) Remove(name string) error {
	m.locker.Lock()
	defer m.locker.Unlock()

	cons, exists := m.consumerMap[name]
	if !exists {
		return ErrorConsumerNotFound
	}

	// 从maps中删除
	delete(m.consumerMap, name)
	delete(m.credentialMap, cons.Credential)

	// 从typeMap中删除
	typeList := m.typeMap[cons.Type]
	for i, c := range typeList {
		if c.Name == name {
			m.typeMap[cons.Type] = append(typeList[:i], typeList[i+1:]...)
			break
		}
	}

	return nil
}

// Validate 验证Consumer是否有效
func (m *Manager) Validate(cons *Consumer) error {
	if cons == nil {
		return ErrorConsumerNil
	}

	if cons.Name == "" {
		return ErrorConsumerNameEmpty
	}

	if cons.Credential == "" {
		return ErrorCredentialEmpty
	}

	if cons.Type != "key" && cons.Type != "jwt" {
		return ErrorInvalidConsumerType
	}

	if cons.Status != 0 && cons.Status != 1 {
		return ErrorInvalidConsumerStatus
	}

	return nil
}

// 错误定义
var (
	ErrorConsumerNil            = NewConsumerError("consumer is nil")
	ErrorConsumerNameEmpty     = NewConsumerError("consumer name is empty")
	ErrorCredentialEmpty       = NewConsumerError("credential is empty")
	ErrorInvalidConsumerType   = NewConsumerError("invalid consumer type, must be 'key' or 'jwt'")
	ErrorInvalidConsumerStatus = NewConsumerError("invalid consumer status, must be 0 or 1")
	ErrorConsumerNotFound      = NewConsumerError("consumer not found")
	ErrorConsumerNameExists   = NewConsumerError("consumer name already exists")
	ErrorCredentialExists      = NewConsumerError("credential already exists")
)

// ConsumerError Consumer错误
type ConsumerError struct {
	Message string
}

// NewConsumerError 创建Consumer错误
func NewConsumerError(message string) *ConsumerError {
	return &ConsumerError{Message: message}
}

// Error 实现error接口
func (e *ConsumerError) Error() string {
	return e.Message
}
