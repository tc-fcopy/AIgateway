package controller

import (
	"errors"
	"testing"
)

func TestSyncServiceRuntime_Success(t *testing.T) {
	origReload := serviceReloadFunc
	origInvalidateService := invalidateServicePlanFunc
	origInvalidateAll := invalidateAllPlansFunc
	defer func() {
		serviceReloadFunc = origReload
		invalidateServicePlanFunc = origInvalidateService
		invalidateAllPlansFunc = origInvalidateAll
	}()

	reloadCalled := 0
	invalidatedService := int64(0)
	invalidatedAll := 0

	serviceReloadFunc = func() error {
		reloadCalled++
		return nil
	}
	invalidateServicePlanFunc = func(serviceID int64) {
		invalidatedService = serviceID
	}
	invalidateAllPlansFunc = func() {
		invalidatedAll++
	}

	if err := syncServiceRuntime(nil, 88); err != nil {
		t.Fatalf("syncServiceRuntime returned error: %v", err)
	}
	if reloadCalled != 1 {
		t.Fatalf("expected reload called once, got %d", reloadCalled)
	}
	if invalidatedService != 88 {
		t.Fatalf("expected invalidated service=88, got %d", invalidatedService)
	}
	if invalidatedAll != 0 {
		t.Fatalf("expected invalidateAll not called, got %d", invalidatedAll)
	}
}

func TestSyncServiceRuntime_ReloadFailed(t *testing.T) {
	origReload := serviceReloadFunc
	origInvalidateService := invalidateServicePlanFunc
	origInvalidateAll := invalidateAllPlansFunc
	defer func() {
		serviceReloadFunc = origReload
		invalidateServicePlanFunc = origInvalidateService
		invalidateAllPlansFunc = origInvalidateAll
	}()

	serviceReloadFunc = func() error {
		return errors.New("reload failed")
	}
	called := false
	invalidateServicePlanFunc = func(serviceID int64) {
		called = true
	}
	invalidateAllPlansFunc = func() {
		called = true
	}

	if err := syncServiceRuntime(nil, 66); err == nil {
		t.Fatalf("expected error when reload fails")
	}
	if called {
		t.Fatalf("invalidate should not be called when reload fails")
	}
}

func TestSyncAIServiceConfigRuntime_InvalidateAll(t *testing.T) {
	origInvalidateService := invalidateServicePlanFunc
	origInvalidateAll := invalidateAllPlansFunc
	defer func() {
		invalidateServicePlanFunc = origInvalidateService
		invalidateAllPlansFunc = origInvalidateAll
	}()

	serviceCalls := 0
	allCalls := 0
	invalidateServicePlanFunc = func(serviceID int64) {
		serviceCalls++
	}
	invalidateAllPlansFunc = func() {
		allCalls++
	}

	if err := syncAIServiceConfigRuntime(0); err != nil {
		t.Fatalf("syncAIServiceConfigRuntime returned error: %v", err)
	}
	if serviceCalls != 0 {
		t.Fatalf("expected no service invalidation, got %d", serviceCalls)
	}
	if allCalls != 1 {
		t.Fatalf("expected invalidateAll called once, got %d", allCalls)
	}
}
