package commands

import (
	"strings"
	"testing"
)

func TestSnapshotCmdBasic(t *testing.T) {
	cmd := SnapshotCmd()
	
	// Test command configuration
	if cmd.Use != "snapshot" {
		t.Errorf("Expected command name 'snapshot', got '%s'", cmd.Use)
	}
	
	if cmd.Short == "" {
		t.Error("Command should have a short description")
	}
	
	if cmd.Long == "" {
		t.Error("Command should have a long description")
	}
	
	// Check that the command has the expected flags
	messageFlag := cmd.Flags().Lookup("message")
	if messageFlag == nil {
		t.Error("Command should have a 'message' flag")
	}
	
	if messageFlag.Shorthand != "m" {
		t.Errorf("Expected shorthand 'm' for message flag, got '%s'", messageFlag.Shorthand)
	}
	
	// Test that help text contains expected content
	if !strings.Contains(cmd.Long, "manual snapshot") {
		t.Error("Long description should mention 'manual snapshot'")
	}
	
	if !strings.Contains(cmd.Long, "debounce") {
		t.Error("Long description should mention debounce delays")
	}
}

func TestSnapshotCmdHelp(t *testing.T) {
	cmd := SnapshotCmd()
	
	// Test help output
	help := cmd.UsageString()
	
	if !strings.Contains(help, "snapshot") {
		t.Error("Help should contain the command name")
	}
	
	if !strings.Contains(help, "-m, --message") {
		t.Error("Help should show the message flag")
	}
}