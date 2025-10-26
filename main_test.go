package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	// Test that version is set correctly
	// Version will be "dev" in tests unless overridden by ldflags
	if version == "" {
		t.Error("Version should not be empty")
	}
	
	// In test environment, version is "dev"
	// When built with ldflags, it will be set to the actual version
	if version != "dev" && version != "1.0.0" && version != "1.1.0" && version != "1.2.0" {
		t.Logf("Version is %s (this is OK for development builds)", version)
	}
}
