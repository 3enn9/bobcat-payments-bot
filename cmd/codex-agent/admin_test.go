package main

import "testing"

func TestAdministrationDisabledWithoutExplicitConfig(t *testing.T) {
	t.Setenv("CODEX_PROJECT_ADMIN", "")
	if _, err := executeSQL("DELETE FROM invoices"); err == nil {
		t.Fatal("SQL mutations enabled without admin mode")
	}
	if _, err := controlProject(map[string]any{"action": "restart", "service": "app"}); err == nil {
		t.Fatal("deployment enabled without admin mode")
	}
}
func TestAdministrationInstructionsRespectScope(t *testing.T) {
	t.Setenv("CODEX_PROJECT_ADMIN", "true")
	if !adminEnabled() {
		t.Fatal("admin mode not enabled")
	}
	if agentInstructions() == readonlyInstructions {
		t.Fatal("admin session received read-only instructions")
	}
}
