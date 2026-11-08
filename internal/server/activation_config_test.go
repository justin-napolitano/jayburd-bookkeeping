package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionactivation"
)

// Finite activation configuration can restrict only an already enabled Sandbox private runtime.
func TestActivationConfigurationScope(t *testing.T) {
	base := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "jayburd-bookkeeping-dev", DatabaseURL: "synthetic", CollectionEnabled: true, CollectionActivationPlanFile: "/explicit/synthetic-approved-plan.json"}
	if err := base.Validate(); err != nil {
		t.Fatal("explicit Sandbox configuration rejected")
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.Mode = "health" },
		func(c *Config) { c.CollectionEnabled = false },
		func(c *Config) { c.Project = "unapproved-project" },
		func(c *Config) { c.ProviderEnvironment = "production" },
	} {
		c := base
		change(&c)
		if c.Validate() == nil {
			t.Fatal("activation escaped Sandbox/explicit private boundary")
		}
	}
	// A configured approval cannot silently mount an absent execution service.
	d := Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(_ context.Context) error { return nil }, Collection: &syntheticCollection{}, Labels: &syntheticLabels{}}
	if _, err := New(base, d); err == nil {
		t.Fatal("missing activation adapter accepted")
	}
}

// TestActivationInlineConfiguration applies the same Sandbox scope to either
// operator delivery and rejects ambiguous/oversized inline metadata before I/O.
func TestActivationInlineConfiguration(t *testing.T) {
	base := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "jayburd-bookkeeping-dev", DatabaseURL: "synthetic", CollectionEnabled: true, CollectionActivationPlanJSON: "{}"}
	if base.Validate() != nil || !base.activationConfigured() {
		t.Fatal("explicit inline profile rejected")
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.Mode = "health" }, func(c *Config) { c.CollectionEnabled = false }, func(c *Config) { c.Project = "unapproved-project" }, func(c *Config) { c.ProviderEnvironment = "production" },
		func(c *Config) { c.CollectionActivationPlanFile = "/explicit/synthetic-plan.json" }, func(c *Config) { c.CollectionActivationPlanJSON = strings.Repeat("x", 32769) },
		func(c *Config) { c.CollectionActivationPlanJSON = strings.Repeat("界", 10923) },
	} {
		c := base
		change(&c)
		if c.Validate() == nil {
			t.Fatal("inline profile escaped reviewed configuration boundary")
		}
	}
	atLimit := base
	atLimit.CollectionActivationPlanJSON = strings.Repeat(" ", 32768)
	if atLimit.Validate() != nil {
		t.Fatal("configuration byte bound changed")
	}
	d := Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }, Collection: &syntheticCollection{}, Labels: &syntheticLabels{}}
	if _, err := New(base, d); err == nil {
		t.Fatal("configured inline profile accepted absent execution adapter")
	}
	// A test-only empty adapter lets routing prove mount/default absence without
	// providing a database or manufacturing an authenticated end-user identity.
	d.Activation = &collectionactivation.Service{}
	for _, enabled := range []bool{false, true} {
		c := base
		if !enabled {
			c.CollectionActivationPlanJSON = ""
		}
		s, err := New(c, d)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodPost, "http://localhost:3000/v1/collection-activation/adopt", strings.NewReader(`{"plan_id":"00000000-0000-4000-8000-000000000001"}`))
		r.Header.Set("Origin", c.Origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.HTTP.Handler.ServeHTTP(w, r)
		want := http.StatusNotFound
		if enabled {
			want = http.StatusUnauthorized
		}
		if w.Code != want {
			t.Fatalf("mount default/configured mismatch: wanted%d got%d", want, w.Code)
		}
	}
}
