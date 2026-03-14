package main

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type HttpRule struct {
	ID        int64     `gorm:"primary"`
	ServiceID int64     `gorm:"column:service_id"`
	RuleType  int       `gorm:"column:rule_type"`
	Rule      string    `gorm:"column:rule"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (HttpRule) TableName() string {
	return "gateway_service_http_rule"
}

type ServiceInfo struct {
	ID          int64     `gorm:"primary"`
	LoadType    int       `gorm:"column:load_type"`
	ServiceName string    `gorm:"column:service_name"`
}

func (ServiceInfo) TableName() string {
	return "gateway_service_info"
}

func main() {
	dsn := "root:jiabei880@tcp(127.0.0.1:3306)/gateway?charset=utf8mb4&parseTime=true&loc=Asia%2FChongqing"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		return
	}

	fmt.Println("=== HTTP路由规则配置 ===\n")

	var services []ServiceInfo
	db.Table("gateway_service_info").Where("is_delete = ?", 0).Find(&services)

	fmt.Println("常量定义:")
	fmt.Println("  HTTPRuleTypePrefixURL = 0 (前缀匹配)")
	fmt.Println("  HTTPRuleTypeDomain    = 1 (域名匹配)\n")

	for _, svc := range services {
		var httpRule HttpRule
		db.Table("gateway_service_http_rule").Where("service_id = ?", svc.ID).First(&httpRule)

		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("服务ID: %d\n", svc.ID)
		fmt.Printf("服务名: %s\n", svc.ServiceName)
		fmt.Printf("规则类型: %d (0=前缀, 1=域名)\n", httpRule.RuleType)
		fmt.Printf("规则值: %s\n", httpRule.Rule)

		if httpRule.RuleType == 1 && (httpRule.Rule == "" || httpRule.Rule[0] == '/') {
			fmt.Printf("⚠️  错误: 规则类型是域名匹配(1)，但规则值是路径格式 '%s'！\n", httpRule.Rule)
		}
		if httpRule.RuleType == 0 && httpRule.Rule == "" {
			fmt.Printf("⚠️  错误: 规则类型是前缀匹配(0)，但规则值为空！\n")
		}
	}
}
