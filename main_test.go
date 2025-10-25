package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	// Test that version is set correctly
	if version == "" {
		t.Error("Version should not be empty")
	}
	
	expectedVersion := "1.0.0"
	if version != expectedVersion {
		t.Errorf("Expected version %s, got %s", expectedVersion, version)
	}
}
