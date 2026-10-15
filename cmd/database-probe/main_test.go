package main

import "testing"

// TestOfflineAndRejectedArguments never constructs a connector or reads credentials.
func TestOfflineAndRejectedArguments(t *testing.T) {
	if run(nil) != nil {
		t.Fatal("offline default failed")
	}
	for _, args := range [][]string{{"-action=up"}, {"-action=probe"}, {"-action=verify", "-instance=invalid"}, {"-token=protected"}, {"extra"}} {
		if run(args) == nil {
			t.Fatal("invalid arguments accepted")
		}
	}
}
