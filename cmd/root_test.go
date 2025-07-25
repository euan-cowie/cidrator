package cmd

import (
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// Test that version variables have default values
	if Version == "" {
		t.Error("Version should have a default value")
	}
	if Commit == "" {
		t.Error("Commit should have a default value")
	}
	if Date == "" {
		t.Error("Date should have a default value")
	}
}

func TestRootCommand(t *testing.T) {
	// Test that root command is properly initialized
	if rootCmd == nil {
		t.Error("rootCmd should not be nil")
	}

	if rootCmd.Use != "cidrator" {
		t.Errorf("Expected rootCmd.Use to be 'cidrator', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
}
