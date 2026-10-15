// Package sample writes synthetic tax invoices as small text PDFs, for
// tests, the browser suite and the latency bench. Nothing here is a real
// supplier; the answer key of real invoices stays private.
package sample

import (
	"fmt"
	"strconv"
	"strings"

	"sitewise/internal/invoices"
)

// Invoice is one synthetic tax invoice. Amounts are cents ex GST and GST.
type Invoice struct {
	Supplier    string
	ABN         string // eleven digits; ABNFor makes a valid one
	Customer    string
	Number      string
	Date        string // as printed, e.g. "14 Oct 2026"
	Description string
	Ex, GST     int64
	Credit      bool
}

// ABNFor returns a checksum-valid ABN derived from n, so each synthetic
// supplier has its own.
func ABNFor(n int) string {
	base := fmt.Sprintf("%09d", 100000000+n*7919%899999999)
	for lead := 10; lead < 100; lead++ {
		abn := strconv.Itoa(lead) + base
		if invoices.ValidABN(abn) {
			return abn
		}
	}
	return ""
}

// PDF lays the invoice out the way common accounting packages do: labels
// above their values on the right, totals in a column at the foot.
func PDF(inv Invoice) []byte {
	var c strings.Builder
	text := func(x, y int, size int, s string) {
		fmt.Fprintf(&c, "BT\n/F1 %d Tf\n1 0 0 1 %d %d Tm\n(%s) Tj\nET\n", size, x, y, escape(s))
	}
	title := "TAX INVOICE"
	if inv.Credit {
		title = "ADJUSTMENT NOTE"
	}
	text(40, 790, 16, title)
	text(380, 790, 11, inv.Supplier)
	text(380, 776, 9, "ABN "+invoices.DisplayABN(inv.ABN))
	text(380, 762, 9, "Level 3, 10 Example Street, Sydney NSW 2000")
	text(40, 740, 10, inv.Customer)
	text(40, 726, 9, "Suite 1, 1 Client Lane, Sydney NSW 2000")
	text(380, 730, 9, "Invoice Date")
	text(380, 718, 10, inv.Date)
	text(380, 700, 9, "Invoice Number")
	text(380, 688, 10, inv.Number)
	text(40, 640, 9, "Description")
	text(470, 640, 9, "Amount AUD")
	text(40, 622, 10, inv.Description)
	text(470, 622, 10, invoices.DisplayCents(inv.Ex))
	text(380, 580, 10, "Subtotal")
	text(470, 580, 10, invoices.DisplayCents(inv.Ex))
	text(380, 566, 10, "Total GST 10%")
	text(470, 566, 10, invoices.DisplayCents(inv.GST))
	text(380, 552, 10, "Invoice Total AUD")
	text(470, 552, 10, invoices.DisplayCents(inv.Ex+inv.GST))
	text(40, 500, 8, "Payment within 14 days. Please quote the invoice number with your payment.")
	return writePDF(c.String())
}

func escape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, "(", `\(`, ")", `\)`)
	return r.Replace(s)
}

func writePDF(content string) []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, body := range objects {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objects)+1)
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return []byte(b.String())
}

// Batch returns n invoices from a few suppliers, numbered from first.
func Batch(first, n int) []Invoice {
	suppliers := []string{"Example Architects Pty Ltd", "Sample Structural Engineers Pty Ltd", "Demo Surveying Pty Ltd"}
	out := make([]Invoice, n)
	for i := range out {
		k := first + i
		ex := int64(100000 + (k%17)*12345)
		out[i] = Invoice{
			Supplier:    suppliers[k%len(suppliers)],
			ABN:         ABNFor(k % len(suppliers)),
			Customer:    "Test Client Pty Ltd",
			Number:      fmt.Sprintf("INV-%05d", k),
			Date:        fmt.Sprintf("%d Oct 2026", 1+k%14),
			Description: "Professional services for October 2026",
			Ex:          ex,
			GST:         invoices.GSTOf(ex),
		}
	}
	return out
}
