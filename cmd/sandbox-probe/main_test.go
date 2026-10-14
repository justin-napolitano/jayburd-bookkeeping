package main

import (
	"bytes"
	"context"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxqualification"
	"testing"
)

// TestConfigurationGate requires opt-in and numeric pins before any credential access.
func TestConfigurationGate(t *testing.T) {
	for _, args := range [][]string{nil, {"-allow-sandbox", "-project-number=123456", "-client-id-version=latest", "-client-secret-version=1"}, {"-project-number=123456", "-client-id-version=1", "-client-secret-version=1"}} {
		var out bytes.Buffer
		if run(context.Background(), args, &out, func(context.Context, string, string, string) sandboxqualification.Receipt {
			t.Fatal("unexpected credential access")
			return sandboxqualification.Receipt{}
		}) != 2 {
			t.Fatal("bad config accepted")
		}
	}
	var out bytes.Buffer
	called := false
	run(context.Background(), []string{"-allow-sandbox", "-project-number=123456", "-client-id-version=1", "-client-secret-version=2"}, &out, func(_ context.Context, p, i, s string) sandboxqualification.Receipt {
		called = true
		if p != "123456" || i != "1" || s != "2" {
			t.Fatal("changed pins")
		}
		return sandboxqualification.Blocked("input_unavailable")
	})
	if !called {
		t.Fatal("valid config rejected")
	}
}
