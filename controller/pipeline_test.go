package controller

import (
	"net/http/httptest"
	"testing"

	"gateway/dao"
	"github.com/gin-gonic/gin"
)

func TestResolveServiceFromQueryByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orig := dao.ServiceManagerHandler
	defer func() {
		dao.ServiceManagerHandler = orig
	}()

	sm := dao.NewServiceManager()
	service := &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 42, ServiceName: "svc-42"}}
	sm.ServiceSlice = append(sm.ServiceSlice, service)
	sm.ServiceMap[service.Info.ServiceName] = service
	dao.ServiceManagerHandler = sm

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/pipeline/plan?service_id=42", nil)

	got, err := resolveServiceFromQuery(c)
	if err != nil {
		t.Fatalf("resolveServiceFromQuery returned error: %v", err)
	}
	if got == nil || got.Info == nil || got.Info.ID != 42 {
		t.Fatalf("unexpected service resolved: %#v", got)
	}
}

func TestResolveServiceFromQueryMissingParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/pipeline/plan", nil)

	if _, err := resolveServiceFromQuery(c); err == nil {
		t.Fatalf("expected error when no service identifier provided")
	}
}
