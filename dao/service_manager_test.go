package dao

import "testing"

func TestServiceManagerGetByID(t *testing.T) {
	sm := NewServiceManager()
	service := &ServiceDetail{Info: &ServiceInfo{ID: 101, ServiceName: "svc-a"}}
	sm.ServiceSlice = append(sm.ServiceSlice, service)
	sm.ServiceMap["svc-a"] = service

	got, ok := sm.GetByID(101)
	if !ok || got == nil || got.Info == nil {
		t.Fatalf("expected service id 101 in cache")
	}
	if got.Info.ServiceName != "svc-a" {
		t.Fatalf("unexpected service name: %s", got.Info.ServiceName)
	}

	if _, ok := sm.GetByID(999); ok {
		t.Fatalf("did not expect unknown service id")
	}
}

func TestServiceManagerGetByName(t *testing.T) {
	sm := NewServiceManager()
	service := &ServiceDetail{Info: &ServiceInfo{ID: 202, ServiceName: "svc-b"}}
	sm.ServiceSlice = append(sm.ServiceSlice, service)
	sm.ServiceMap["svc-b"] = service

	got, ok := sm.GetByName("svc-b")
	if !ok || got == nil || got.Info == nil {
		t.Fatalf("expected service name svc-b in cache")
	}
	if got.Info.ID != 202 {
		t.Fatalf("unexpected service id: %d", got.Info.ID)
	}

	if _, ok := sm.GetByName("missing"); ok {
		t.Fatalf("did not expect missing service name")
	}
}
