package invoices_test

import (
	"testing"

	"sitewise/internal/invoices"
	"sitewise/internal/proposals"
)

// The firm directory stores proposals.NameKey; an invoice's heading must be
// reduced the same way to meet it.
func TestFirmKeyMatchesTheFirmDirectory(t *testing.T) {
	for _, name := range []string{"KARRAK CONSULTING ENGINEERS PTY LTD", "Halyard Remedial Consultants Pty Ltd", "Mecone Pty Ltd", "Smith & Sons Group (Aust) Pty Limited", "Tallow Consulting Engineers"} {
		if got, want := invoices.FirmKey(name), proposals.NameKey(name); got != want {
			t.Fatalf("%q: %q, directory %q", name, got, want)
		}
	}
}
