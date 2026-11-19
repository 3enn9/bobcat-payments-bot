package main

import "testing"

func TestQueryRestrictions(t *testing.T) {
	for _, q := range []string{"DELETE FROM invoices", "SELECT 1; DELETE FROM invoices", "SELECT SLEEP(100)", "SELECT * INTO OUTFILE '/tmp/x' FROM invoices", "SELECT LOAD_FILE('/etc/passwd')", "SELECT * FROM invoices FOR UPDATE", "SELECT 1 /* comment */"} {
		if validSQL(q) {
			t.Fatalf("unsafe query accepted: %s", q)
		}
	}
	for _, q := range []string{"SELECT COUNT(*) FROM invoices", "select number,total FROM invoices WHERE total > 100"} {
		if !validSQL(q) {
			t.Fatalf("read query rejected: %s", q)
		}
	}
}
func TestSourceRestrictions(t *testing.T) {
	for _, p := range []string{"../.env", ".env", "/etc/passwd", ".git/config", "miniapp/node_modules/x.ts", "certs/private.ts", "foo/../../secret.go"} {
		if sourceAllowed(p) {
			t.Fatalf("unsafe path %s", p)
		}
	}
	if !sourceAllowed("internal/invoice/pdf.go") {
		t.Fatal("source inaccessible")
	}
}
func TestRedact(t *testing.T) {
	if got := redact("password=abc token:xyz"); got != "password=[REDACTED] token=[REDACTED]" {
		t.Fatal(got)
	}
}
