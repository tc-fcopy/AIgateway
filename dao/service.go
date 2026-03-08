package dao

import (
	"errors"
	"gateway/dto"
	"gateway/golang_common/lib"
	"gateway/public"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"sync"
)

type ServiceDetail struct {
	Info          *ServiceInfo   `json:"info" description:"鍩烘湰淇℃伅"`
	HTTPRule      *HttpRule      `json:"http_rule" description:"http_rule"`
	TCPRule       *TcpRule       `json:"tcp_rule" description:"tcp_rule"`
	GRPCRule      *GrpcRule      `json:"grpc_rule" description:"grpc_rule"`
	LoadBalance   *LoadBalance   `json:"load_balance" description:"load_balance"`
	AccessControl *AccessControl `json:"access_control" description:"access_control"`
}

var ServiceManagerHandler *ServiceManager

func init() {
	ServiceManagerHandler = NewServiceManager()
}

type ServiceManager struct {
	ServiceMap   map[string]*ServiceDetail
	ServiceSlice []*ServiceDetail
	Locker       sync.RWMutex
	init         sync.Once
	err          error
}

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		ServiceMap:   make(map[string]*ServiceDetail),
		ServiceSlice: make([]*ServiceDetail, 0),
		Locker:       sync.RWMutex{},
		init:         sync.Once{},
	}
}

func (s *ServiceManager) HTTPAccessMode(c *gin.Context) (*ServiceDetail, error) {
	// 1. Prefix match /abc => serviceSlice.rule
	// 2. Domain match www.test.com => serviceSlice.rule
	host := c.Request.Host
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}
	path := c.Request.URL.Path

	s.Locker.RLock()
	defer s.Locker.RUnlock()

	for _, serviceItem := range s.ServiceSlice {
		if serviceItem == nil || serviceItem.Info == nil || serviceItem.HTTPRule == nil {
			continue
		}
		if serviceItem.Info.LoadType != public.LoadTypeHTTP {
			continue
		}
		if serviceItem.HTTPRule.RuleType == public.HTTPRuleTypeDomain {
			if serviceItem.HTTPRule.Rule == host {
				return serviceItem, nil
			}
		}
		if serviceItem.HTTPRule.RuleType == public.HTTPRuleTypePrefixURL {
			if strings.HasPrefix(path, serviceItem.HTTPRule.Rule) {
				return serviceItem, nil
			}
		}
	}
	return nil, errors.New("not matched service")
}

func (sm *ServiceManager) LoadOnce() error {
	sm.init.Do(func() {
		sm.err = sm.Reload()
	})
	return sm.err
}

// Reload refreshes all service routing metadata from DB.
func (sm *ServiceManager) Reload() error {
	serviceInfo := &ServiceInfo{}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	tx, err := lib.GetGormPool("default")
	if err != nil {
		return err
	}
	params := &dto.ServiceListInput{PageNum: 1, PageSize: 99999}
	list, _, err := serviceInfo.PageList(c, tx, params)
	if err != nil {
		return err
	}

	nextMap := make(map[string]*ServiceDetail, len(list))
	nextSlice := make([]*ServiceDetail, 0, len(list))
	for _, listItem := range list {
		tmpItem := listItem
		serviceDetail, detailErr := tmpItem.ServiceDetail(c, tx, &tmpItem)
		if detailErr != nil {
			return detailErr
		}
		nextMap[listItem.ServiceName] = serviceDetail
		nextSlice = append(nextSlice, serviceDetail)
	}

	sm.Locker.Lock()
	sm.ServiceMap = nextMap
	sm.ServiceSlice = nextSlice
	sm.Locker.Unlock()

	return nil
}

// GetByID returns service detail by service id from in-memory runtime cache.
func (sm *ServiceManager) GetByID(id int64) (*ServiceDetail, bool) {
	if sm == nil || id <= 0 {
		return nil, false
	}

	sm.Locker.RLock()
	defer sm.Locker.RUnlock()
	for _, service := range sm.ServiceSlice {
		if service == nil || service.Info == nil {
			continue
		}
		if service.Info.ID == id {
			return service, true
		}
	}
	return nil, false
}

// GetByName returns service detail by service name from in-memory runtime cache.
func (sm *ServiceManager) GetByName(serviceName string) (*ServiceDetail, bool) {
	if sm == nil || serviceName == "" {
		return nil, false
	}

	sm.Locker.RLock()
	defer sm.Locker.RUnlock()
	service, ok := sm.ServiceMap[serviceName]
	if !ok || service == nil {
		return nil, false
	}
	return service, true
}
