package main

import "testing"

// TestAutomaticCategoryFlag keeps opt-in publication consumer-only and default constructors unchanged.
func TestAutomaticCategoryFlag(t *testing.T) {
	base, e := parse(consumerArgs())
	if e != nil || base.Worker.Categories {
		t.Fatal("default changed", e)
	}
	enabled, e := parse(append(consumerArgs(), "-automatic-categories"))
	if e != nil || !enabled.Worker.Categories {
		t.Fatal("consumer flag missing", e)
	}
	if _, e := parse(append(publisherArgs(), "-automatic-categories")); e == nil {
		t.Fatal("publisher accepted category publication")
	}
}
