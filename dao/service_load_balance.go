package dao

import (
	"fmt"
	"gateway/public"
	"gateway/reverse_proxy/load_balance"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type LoadBalance struct {
	ID            int64  `json:"id" gorm:"primary_key"`
	ServiceID     int64  `json:"service_id" gorm:"column:service_id" description:"鏈嶅姟id	"`
	CheckMethod   int    `json:"check_method" gorm:"column:check_method" description:"妫€鏌ユ柟cpchk=妫€娴嬬鍙ｆ槸鍚︽彙鎵嬫垚锟?"`
	CheckTimeout  int    `json:"check_timeout" gorm:"column:check_timeout" description:"check瓒呮椂鏃堕棿	"`
	CheckInterval int    `json:"check_interval" gorm:"column:check_interval" description:"妫€鏌ラ棿锟? 鍗曚綅s		"`
	RoundType     int    `json:"round_type" gorm:"column:round_type" description:"杞鏂瑰紡 round/weight_round/random/ip_hash"`
	IpList        string `json:"ip_list" gorm:"column:ip_list" description:"ip鍒楄〃"`
	WeightList    string `json:"weight_list" gorm:"column:weight_list" description:"weight list"`
	ForbidList    string `json:"forbid_list" gorm:"column:forbid_list" description:"绂佺敤ip鍒楄〃"`

	UpstreamConnectTimeout int `json:"upstream_connect_timeout" gorm:"column:upstream_connect_timeout" description:"涓嬫父寤虹珛杩炴帴瓒呮椂, 鍗曝綅s"`
	UpstreamHeaderTimeout  int `json:"upstream_header_timeout" gorm:"column:upstream_header_timeout" description:"涓嬫父鑾峰彇header瓒呮椂, 鍗曝綅s	"`
	UpstreamIdleTimeout    int `json:"upstream_idle_timeout" gorm:"column:upstream_idle_timeout" description:"涓嬫父閾炬帴鏈€澶х┖闂叉椂锟? 鍗曝綅s	"`
	UpstreamMaxIdle        int `json:"upstream_max_idle" gorm:"column:upstream_max_idle" description:"涓嬫父鏈€澶х┖闂查摼鎺ユ暟"`
}

func (t *LoadBalance) TableName() string {
	return "gateway_service_load_balance"
}

func (t *LoadBalance) Find(c *gin.Context, tx *gorm.DB, search *LoadBalance) (*LoadBalance, error) {
	model := &LoadBalance{}
	err := tx.WithContext(c).Where(search).Find(model).Error
	return model, err
}

func (t *LoadBalance) Save(c *gin.Context, tx *gorm.DB) error {
	if err := tx.WithContext(c).Save(t).Error; err != nil {
		return err
	}
	return nil
}

func (t *LoadBalance) GetIPListByModel() []string {
	// 瑙ｆ瀽IpList瀛楃涓诧紝鍋囪瀹冩槸浠ラ€楀彿鍒嗛殧鐨処P鍦板潃鍒楄〃
	ips := strings.Split(t.IpList, ",")
	// 杩囨护绌哄瓧绗︿覆
	var result []string
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if strings.Contains(ip, "://") {
			if parsed, err := url.Parse(ip); err == nil && parsed.Host != "" {
				ip = parsed.Host
			}
		}
		result = append(result, ip)
	}
	return result
}

func (t *LoadBalance) GetWeightListByModel() []string {
	parts := strings.Split(t.WeightList, ",")
	var result []string
	for _, w := range parts {
		w = strings.TrimSpace(w)
		if w != "" {
			result = append(result, w)
		}
	}
	return result
}

var LoadBalancerHandler *LoadBalancer

type LoadBalancer struct {
	LoadBanlanceMap   map[string]*LoadBalancerItem
	LoadBanlanceSlice []*LoadBalancerItem
	Locker            sync.RWMutex
}

type LoadBalancerItem struct {
	LoadBanlance load_balance.LoadBalance
	ServiceName  string
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		LoadBanlanceMap:   map[string]*LoadBalancerItem{},
		LoadBanlanceSlice: []*LoadBalancerItem{},
		Locker:            sync.RWMutex{},
	}
}

func init() {
	LoadBalancerHandler = NewLoadBalancer()
}

func (lbr *LoadBalancer) GetLoadBalancer(service *ServiceDetail) (load_balance.LoadBalance, error) {
	if service == nil || service.Info == nil || service.LoadBalance == nil {
		return nil, fmt.Errorf("service load balance config is nil")
	}
	serviceName := service.Info.ServiceName

	lbr.Locker.RLock()
	if lbrItem, ok := lbr.LoadBanlanceMap[serviceName]; ok && lbrItem != nil {
		lbr.Locker.RUnlock()
		return lbrItem.LoadBanlance, nil
	}
	lbr.Locker.RUnlock()

	schema := "http://"
	if service.HTTPRule.NeedHttps == 1 {
		schema = "https://"
	}
	if service.Info.LoadType == public.LoadTypeTCP || service.Info.LoadType == public.LoadTypeGRPC {
		schema = ""
	}
	ipList := service.LoadBalance.GetIPListByModel()
	if len(ipList) == 0 {
		return nil, fmt.Errorf("upstream ip list is empty")
	}
	weightList := service.LoadBalance.GetWeightListByModel()
	defaultWeight := "50"
	for len(weightList) < len(ipList) {
		weightList = append(weightList, defaultWeight)
	}
	ipConf := map[string]string{}
	for ipIndex, ipItem := range ipList {
		weight := defaultWeight
		if ipIndex < len(weightList) && strings.TrimSpace(weightList[ipIndex]) != "" {
			weight = strings.TrimSpace(weightList[ipIndex])
		}
		ipConf[ipItem] = weight
	}
	// fmt.Println("ipConf", ipConf)
	mConf, err := load_balance.NewLoadBalanceCheckConf(fmt.Sprintf("%s%s", schema, "%s"), ipConf)
	if err != nil {
		return nil, err
	}
	lb := load_balance.LoadBanlanceFactorWithConf(load_balance.LbType(service.LoadBalance.RoundType), mConf)

	lbItem := &LoadBalancerItem{
		LoadBanlance: lb,
		ServiceName:  serviceName,
	}
	lbr.Locker.Lock()
	if lbrItem, ok := lbr.LoadBanlanceMap[serviceName]; ok && lbrItem != nil {
		lbr.Locker.Unlock()
		return lbrItem.LoadBanlance, nil
	}
	lbr.LoadBanlanceSlice = append(lbr.LoadBanlanceSlice, lbItem)
	lbr.LoadBanlanceMap[serviceName] = lbItem
	lbr.Locker.Unlock()
	return lb, nil
}

var TransportorHandler *Transportor

type Transportor struct {
	TransportMap   map[string]*TransportItem
	TransportSlice []*TransportItem
	Locker         sync.RWMutex
}

type TransportItem struct {
	Trans       *http.Transport
	ServiceName string
}

func NewTransportor() *Transportor {
	return &Transportor{
		TransportMap:   map[string]*TransportItem{},
		TransportSlice: []*TransportItem{},
		Locker:         sync.RWMutex{},
	}
}

func init() {
	TransportorHandler = NewTransportor()
}

func (t *Transportor) GetTrans(service *ServiceDetail) (*http.Transport, error) {
	if service == nil || service.Info == nil || service.LoadBalance == nil {
		return nil, fmt.Errorf("service transport config is nil")
	}
	serviceName := service.Info.ServiceName

	t.Locker.RLock()
	if transItem, ok := t.TransportMap[serviceName]; ok && transItem != nil {
		t.Locker.RUnlock()
		return transItem.Trans, nil
	}
	t.Locker.RUnlock()

	//todo 浼樺寲鐐?
	if service.LoadBalance.UpstreamConnectTimeout == 0 {
		service.LoadBalance.UpstreamConnectTimeout = 30
	}
	if service.LoadBalance.UpstreamMaxIdle == 0 {
		service.LoadBalance.UpstreamMaxIdle = 100
	}
	if service.LoadBalance.UpstreamIdleTimeout == 0 {
		service.LoadBalance.UpstreamIdleTimeout = 90
	}
	if service.LoadBalance.UpstreamHeaderTimeout == 0 {
		service.LoadBalance.UpstreamHeaderTimeout = 30
	}
	trans := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(service.LoadBalance.UpstreamConnectTimeout) * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          service.LoadBalance.UpstreamMaxIdle,
		IdleConnTimeout:       time.Duration(service.LoadBalance.UpstreamIdleTimeout) * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: time.Duration(service.LoadBalance.UpstreamHeaderTimeout) * time.Second,
	}

	transItem := &TransportItem{
		Trans:       trans,
		ServiceName: serviceName,
	}
	t.Locker.Lock()
	if existing, ok := t.TransportMap[serviceName]; ok && existing != nil {
		t.Locker.Unlock()
		return existing.Trans, nil
	}
	t.TransportSlice = append(t.TransportSlice, transItem)
	t.TransportMap[serviceName] = transItem
	t.Locker.Unlock()
	return trans, nil
}
