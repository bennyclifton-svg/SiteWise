package consent

import (
	"os"
	"path/filepath"
	"testing"
)

// privateKeys is the owner's key; the tests that need it skip without it.
const privateKeys = "../../data/eval/consent/private"

func TestKeyLoaderAndSplitScore(t *testing.T) {
	dir := t.TempDir()
	body := `document: seven-hills-public
source: internal/consent/testdata/seven-hills-notice.md
reviewed: false
conditions:
  - {number: "6", page: 1, stage: before_cc, party: consultant.basix, system: none, evidence: true}
  - {number: "9", page: 1, stage: before_cc, party: applicant, system: none, evidence: true}
  - {number: "12", page: 1, stage: before_work, party: consultant.arborist, system: site, evidence: true}
  - {number: "18", page: 1, stage: before_occupation, party: consultant.civil, system: site, evidence: true}
  - {number: "23", page: 1, stage: during_works, party: trade.main_works, system: site, evidence: false}
  - {number: "27", page: 1, stage: during_works, party: trade.main_works, system: none, evidence: false}
  - {number: "31", page: 1, stage: before_subdivision, party: consultant.surveyor, system: none, evidence: true}
`
	if err := os.WriteFile(filepath.Join(dir, "seven-hills.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	keys, err := LoadKeys(dir)
	if err != nil || len(keys) != 1 || len(keys[0].Conditions) != 7 {
		t.Fatalf("keys %+v %v", keys, err)
	}
	if exact, problems := SplitScore(keys[0], Split(fixture(t))); !exact {
		t.Fatalf("split against the key: %v", problems)
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("document: x\nconditions:\n  - {number: \"1\", party: plumber}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeys(dir); err == nil {
		t.Fatal("an unknown party was loaded")
	}
	if keys, err := LoadKeys(filepath.Join(dir, "missing")); err != nil || keys != nil {
		t.Fatal("a missing key directory is no key")
	}
}

// TestPrivateKeySplitsExactly runs the split over the owner's key: 102 of
// 102 on Petersham, 41 of 41 on Camp Y, the modification markers. It skips
// until the owner supplies the key and the notices' text (cmd/consent-eval
// -pages writes a notice's pages beside its key).
func TestPrivateKeySplitsExactly(t *testing.T) {
	keys, err := LoadKeys(privateKeys)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Skip("no consent answer key in data/eval/consent/private")
	}
	for _, k := range keys {
		pages, err := PagesFile(filepath.Join(privateKeys, k.Document+".pages.json"))
		if err != nil {
			t.Logf("%s: no pages file (%v); run cmd/consent-eval -pages", k.Document, err)
			continue
		}
		if exact, problems := SplitScore(k, Split(pages)); !exact {
			t.Errorf("%s: %v", k.Document, problems)
		}
	}
}
