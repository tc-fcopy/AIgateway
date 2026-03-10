package controller

import (
	"errors"
	"testing"
)

func TestSyncServiceRuntime_Success(t *testing.T) {
	origReload := serviceReloadFunc
	origReloadAI := reloadAIServiceRuntimeFunc
	origInvalidateService := invalidateServicePlanFunc
	origInvalidateAll := invalidateAllPlansFunc
	defer func() {
		serviceReloadFunc = origReload
		reloadAIServiceRuntimeFunc = origReloadAI
		invalidateServicePlanFunc = origInvalidateService
		invalidateAllPlansFunc = origInvalidateAll
	}()

	reloadCalled := 0
	reloadAICalled := 0
	invalidatedService := int64(0)
	invalidatedAll := 0

	serviceReloadFunc = func() error {
		reloadCalled++
		return nil
	}
	reloadAIServiceRuntimeFunc = func(serviceID int64) error {
		reloadAICalled++
		if serviceID != 88 {
			t.Fatalf("expected reload ai service id=88, got %d", serviceID)
		}
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
	if reloadAICalled != 1 {
		t.Fatalf("expected reload ai runtime called once, got %d", reloadAICalled)
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
	origReloadAI := reloadAIServiceRuntimeFunc
	origInvalidateService := invalidateServicePlanFunc
	origInvalidateAll := invalidateAllPlansFunc
	defer func() {
		serviceReloadFunc = origReload
		reloadAIServiceRuntimeFunc = origReloadAI
		invalidateServicePlanFunc = origInvalidateService
		invalidateAllPlansFunc = origInvalidateAll
	}()

	serviceReloadFunc = func() error {
		return errors.New("reload failed")
	}
	called := false
	reloadAIServiceRuntimeFunc = func(serviceID int64) error {
		called = true
		return nil
	}
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
	origReloadAI := reloadAIServiceRuntimeFunc
	origInvalidateService := invalidateServicePlanFunc
	origInvalidateAll := invalidateAllPlansFunc
	defer func() {
		reloadAIServiceRuntimeFunc = origReloadAI
		invalidateServicePlanFunc = origInvalidateService
		invalidateAllPlansFunc = origInvalidateAll
	}()

	reloadCalls := 0
	serviceCalls := 0
	allCalls := 0
	reloadAIServiceRuntimeFunc = func(serviceID int64) error {
		reloadCalls++
		if serviceID != 0 {
			t.Fatalf("expected reload all with serviceID=0, got %d", serviceID)
		}
		return nil
	}
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
	if reloadCalls != 1 {
		t.Fatalf("expected reload ai runtime called once, got %d", reloadCalls)
	}
	if allCalls != 1 {
		t.Fatalf("expected invalidateAll called once, got %d", allCalls)
	}
}

func TestSyncAIServiceConfigRuntime_ReloadFailed(t *testing.T) {
	origReloadAI := reloadAIServiceRuntimeFunc
	origInvalidateService := invalidateServicePlanFunc
	origInvalidateAll := invalidateAllPlansFunc
	defer func() {
		reloadAIServiceRuntimeFunc = origReloadAI
		invalidateServicePlanFunc = origInvalidateService
		invalidateAllPlansFunc = origInvalidateAll
	}()

	reloadAIServiceRuntimeFunc = func(serviceID int64) error {
		return errors.New("reload ai runtime failed")
	}

	invalidateCalled := false
	invalidateServicePlanFunc = func(serviceID int64) {
		invalidateCalled = true
	}
	invalidateAllPlansFunc = func() {
		invalidateCalled = true
	}

	if err := syncAIServiceConfigRuntime(123); err == nil {
		t.Fatalf("expected error when reload ai runtime fails")
	}
	if invalidateCalled {
		t.Fatalf("invalidate should not be called when reload ai runtime fails")
	}
}
