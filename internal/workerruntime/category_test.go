package workerruntime

import "testing"

// TestCategoryConsumerOnly keeps publisher identity/queue admission separate from metadata ingestion.
func TestCategoryConsumerOnly(t *testing.T) {
	c := config("consumer")
	if c.Categories || c.Validate() != nil {
		t.Fatal("default changed")
	}
	c.Categories = true
	if c.Validate() != nil {
		t.Fatal("explicit consumer rejected")
	}
	p := config("publisher")
	p.Categories = true
	if p.Validate() == nil {
		t.Fatal("publisher accepted consumer category mode")
	}
}
