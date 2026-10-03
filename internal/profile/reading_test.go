package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"sitewise/internal/profile"
)

func repoReadPolicy(t *testing.T) profile.ReadPolicy {
	t.Helper()
	p, err := profile.LoadReadPolicy(filepath.Join("..", "..", "data", "profile", "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadPolicyFollowsKindUntilTheUserChooses(t *testing.T) {
	p := repoReadPolicy(t)
	cases := []struct {
		kind, setting string
		want          bool
	}{
		{"drawing", "auto", false},
		{"photo", "auto", false},
		{"correspondence", "auto", false},
		{"", "auto", false},
		{"unknown", "auto", false},
		{"specification", "auto", true},
		{"design_brief", "auto", true},
		{"report", "", true},
		{"drawing", "read", true},
		{"specification", "skip", false},
	}
	for _, c := range cases {
		if got := p.Reads(c.kind, c.setting); got != c.want {
			t.Errorf("Reads(%q, %q) = %v, want %v", c.kind, c.setting, got, c.want)
		}
	}
}

func TestLoadReadPolicyRejectsAnEmptyList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"read_kinds":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := profile.LoadReadPolicy(path); err == nil {
		t.Fatal("an empty read list would silently read nothing")
	}
}

func TestValidReadSetting(t *testing.T) {
	for _, s := range []string{"auto", "read", "skip"} {
		if !profile.ValidReadSetting(s) {
			t.Errorf("%q rejected", s)
		}
	}
	for _, s := range []string{"", "Read", "none"} {
		if profile.ValidReadSetting(s) {
			t.Errorf("%q accepted", s)
		}
	}
}

func TestBuildIgnoresDocumentsTheProfileDoesNotRead(t *testing.T) {
	cat := repoCatalog(t)
	drawing := jevFact("hdr.scale.storeys", "1", "drawing-1", 0.9)
	drawing.DocumentKind = "drawing"
	spec := jevFact("hdr.scale.storeys", "2", "spec-1", 0.9)
	spec.DocumentKind = "specification"

	in := input(drawing, spec)
	in.Read = repoReadPolicy(t)
	r := row(t, profile.Build(in, cat), whole, "hdr.scale.storeys")
	if r.Value != "2" || r.Band != "amber" || len(r.Sources) != 1 {
		t.Fatalf("an automatic drawing must not count: %+v", r)
	}

	drawing.ReadSetting = "read"
	in = input(drawing, spec)
	in.Read = repoReadPolicy(t)
	if r := row(t, profile.Build(in, cat), whole, "hdr.scale.storeys"); r.Band != "red" {
		t.Fatalf("a drawing the user chose to read must count: %+v", r)
	}

	spec.ReadSetting = "skip"
	in = input(spec)
	in.Read = repoReadPolicy(t)
	noRow(t, profile.Build(in, cat), whole, "hdr.scale.storeys")
}

func TestBuildWithoutAPolicyReadsEverything(t *testing.T) {
	drawing := jevFact("hdr.scale.storeys", "1", "drawing-1", 0.9)
	drawing.DocumentKind = "drawing"
	if r := row(t, profile.Build(input(drawing), repoCatalog(t)), whole, "hdr.scale.storeys"); r.Value != "1" {
		t.Fatalf("%+v", r)
	}
}
