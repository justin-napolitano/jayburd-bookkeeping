package server

import (
	"context"
	"testing"
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
