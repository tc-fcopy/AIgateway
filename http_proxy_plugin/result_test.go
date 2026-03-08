package http_proxy_plugin

import (
	"errors"
	"testing"
)

func TestResultConstructors(t *testing.T) {
	if Continue().IsAbort() {
		t.Fatalf("continue should not abort")
	}

	err := errors.New("boom")
	a := Abort(err)
	if !a.IsAbort() {
		t.Fatalf("abort should abort")
	}
	if a.Err == nil || a.Err.Error() != "boom" {
		t.Fatalf("abort error mismatch")
	}

	as := AbortWithStatus(429, err)
	if !as.IsAbort() || as.HTTPStatus != 429 {
		t.Fatalf("abort with status mismatch: %+v", as)
	}

	ac := AbortWithCode(401, 3001, "unauthorized", err)
	if !ac.IsAbort() || ac.HTTPStatus != 401 || ac.Code != 3001 || ac.Message != "unauthorized" {
		t.Fatalf("abort with code mismatch: %+v", ac)
	}
}
