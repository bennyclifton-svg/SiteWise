package works

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProposalPolicy(t *testing.T) {
	p, err := LoadProposalPolicy("../../data/profile/proposals.json")
	if err != nil || p.ShowCount != DefaultProposalShowCount {
		t.Fatalf("default policy: %+v %v", p, err)
	}
	path := filepath.Join(t.TempDir(), "policy.json")
	for _, raw := range []string{`{}`, `null`, `{"show_count":0}`, `{"show_count":-1}`, `{"show_count":1001}`, `{"show_count":2,"unknown":true}`, `{"show_count":2} {}`} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadProposalPolicy(path); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := LoadProposalPolicy(path + "-missing"); err == nil {
		t.Fatal("accepted missing policy")
	}
}
