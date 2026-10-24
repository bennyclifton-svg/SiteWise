package httpapi

import "testing"

// Every format filing accepts is read as a proposal; Markdown was filed but
// left unread, so a Markdown fee proposal never reached its package.
func TestProposalFormat(t *testing.T) {
	for name, want := range map[string]string{"a.pdf": "pdf", "b.DOCX": "docx", "c.xlsx": "xlsx", "d.md": "md", "e.markdown": "md", "f.txt": "", "noext": ""} {
		if got := proposalFormat(name); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
}
