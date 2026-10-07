package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"sitewise/internal/intake"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/works"
)

// Exercise the deployed paths without opening a database, listener or Jev
// connection. Optional catalogue files can disappear without Load failing, so
// compare its complete loaded-byte fingerprint as well as successful parsing.
func TestReleaseRuntimeAssets(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	stage := os.Getenv("SITEWISE_RELEASE_ROOT")
	if stage == "" {
		stage = t.TempDir()
		script, err := os.ReadFile(filepath.Join(repo, "deploy", "package.sh"))
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range []struct{ source, target string }{
			{"data/intake", "share/intake"}, {"data/profile", "share/profile"}, {"knowledge", "share/knowledge"},
		} {
			declaration := fmt.Sprintf("cp -r %s \"$stage/%s\"", item.source, item.target)
			if !strings.Contains(string(script), declaration) {
				t.Fatalf("package must stage runtime assets: %s", declaration)
			}
			releaseCopyTree(t, filepath.Join(repo, filepath.FromSlash(item.source)), filepath.Join(stage, filepath.FromSlash(item.target)))
		}
		releaseCopyTree(t, filepath.Join(repo, "deploy"), filepath.Join(stage, "deploy"))
	}
	if err := releaseAssetsMatch(repo, stage); err != nil {
		t.Fatal(err)
	}
	// No evaluation corpus, replay cache or private project fixtures belong in
	// the runtime share directory. Catch accidental broad data/ copies.
	entries, err := os.ReadDir(filepath.Join(stage, "share"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "intake" && entry.Name() != "profile" && entry.Name() != "knowledge" {
			t.Fatalf("unexpected release payload: share/%s", entry.Name())
		}
	}
}

func TestReleaseRuntimeAssetsRejectMissingAndWrongPolicy(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	for _, item := range []struct{ source, target string }{
		{"data/intake", "share/intake"}, {"data/profile", "share/profile"}, {"knowledge", "share/knowledge"}, {"deploy", "deploy"},
	} {
		releaseCopyTree(t, filepath.Join(repo, filepath.FromSlash(item.source)), filepath.Join(stage, filepath.FromSlash(item.target)))
	}
	for _, rel := range []string{"share/profile/thresholds.json", "share/profile/reading.json", "share/profile/proposals.json", "share/knowledge/reports/templates.yaml"} {
		t.Run(rel, func(t *testing.T) {
			path := filepath.Join(stage, filepath.FromSlash(rel))
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := releaseAssetsMatch(repo, stage); err == nil {
				t.Fatal("missing runtime asset accepted")
			}
			if err := os.WriteFile(path, body, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	path := filepath.Join(stage, "share", "profile", "proposals.json")
	if err := os.WriteFile(path, []byte(`{"show_count":999}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := releaseAssetsMatch(repo, stage); err == nil {
		t.Fatal("wrong proposal policy accepted")
	}
}

func releaseAssetsMatch(repo, stage string) error {
	unit, err := os.ReadFile(filepath.Join(stage, "deploy", "sitewise.service"))
	if err != nil {
		return err
	}
	var args []string
	for _, line := range strings.Split(string(unit), "\n") {
		if strings.HasPrefix(line, "ExecStart=") {
			args = strings.Fields(strings.TrimPrefix(line, "ExecStart="))
		}
	}
	if len(args) < 2 || args[0] != "/opt/sitewise/bin/sitewise" || args[1] != "serve" {
		return fmt.Errorf("invalid service entry point")
	}
	flags := map[string]string{}
	for i := 2; i+1 < len(args); i += 2 {
		flags[args[i]] = args[i+1]
	}
	checks := []struct {
		flag, source, deployed string
		load                   func(string) (any, error)
	}{
		{"-data", "data/intake", "share/intake", func(p string) (any, error) { return intake.LoadCatalog(p) }},
		{"-data", "data/intake", "share/intake", func(p string) (any, error) { return intake.LoadThresholds(p) }},
		{"-knowledge", "knowledge", "share/knowledge", func(p string) (any, error) {
			c, e := knowledge.Load(p)
			if e != nil {
				return nil, e
			}
			return c.Version(), nil
		}},
		{"-profile-thresholds", "data/profile/thresholds.json", "share/profile/thresholds.json", func(p string) (any, error) { return profile.LoadThresholds(p) }},
		{"-profile-reading", "data/profile/reading.json", "share/profile/reading.json", func(p string) (any, error) { return profile.LoadReadPolicy(p) }},
		{"-proposal-policy", "data/profile/proposals.json", "share/profile/proposals.json", func(p string) (any, error) { return works.LoadProposalPolicy(p) }},
	}
	for _, check := range checks {
		if flags[check.flag] != "/opt/sitewise/"+check.deployed {
			return fmt.Errorf("service %s must resolve staged %s", check.flag, check.deployed)
		}
		want, err := check.load(filepath.Join(repo, filepath.FromSlash(check.source)))
		if err != nil {
			return fmt.Errorf("source %s: %w", check.source, err)
		}
		got, err := check.load(filepath.Join(stage, filepath.FromSlash(check.deployed)))
		if err != nil {
			return fmt.Errorf("release %s: %w", check.deployed, err)
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("release %s differs from source runtime configuration", check.deployed)
		}
	}
	return nil
}

func releaseCopyTree(t *testing.T, source, target string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected non-regular asset %s", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, body, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
}
