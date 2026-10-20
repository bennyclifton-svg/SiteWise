package knowledge_test

import (
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

// releaseFixture writes the load fixture plus one sequences interface with
// a hold point; release is the YAML under hold_point.release ("" for none).
func releaseFixture(t *testing.T, release string) string {
	t.Helper()
	root := writeFixture(t)
	body := `
version: 1
interfaces:
  - id: if.slab-before-frame
    type: sequences
    from: [structure]
    to: [fire-passive]
    status: draft
    hold_point:
      before: structure
      after: fire-passive
      evidence: Reinforcement inspection release identifying the pour.
` + release
	mustWrite(t, filepath.Join(root, "clusters", "zz", "interfaces.yaml"), body)
	return root
}

const goodRelease = `      release:
        type: noul
        instructions: >-
          Using ` + "`text`" + `, does this passage state that the inspection described in
          ` + "`hold_point`" + ` was carried out and the work was released to proceed?
        criteria:
          true: It states the inspection was completed and the work may proceed.
          false: It only requires, schedules or requests the inspection.
        runs_on: [structure]
        status: draft
`

func TestHoldPointReleaseLoads(t *testing.T) {
	cat, err := knowledge.Load(releaseFixture(t, goodRelease))
	if err != nil {
		t.Fatal(err)
	}
	var hp *knowledge.HoldPoint
	for _, iface := range cat.Interfaces {
		if iface.ID == "if.slab-before-frame" {
			hp = iface.HoldPoint
		}
	}
	if hp == nil || hp.Release == nil {
		t.Fatalf("release not parsed: %+v", hp)
	}
	r := hp.Release
	if r.ID != "hold:if.slab-before-frame" || r.Type != "noul" || r.Status != "draft" || len(r.RunsOn) != 1 || !strings.Contains(r.Instructions, "`hold_point`") {
		t.Fatalf("release = %+v", r)
	}
	if _, ok := cat.HoldPointRelease("if.slab-before-frame", []string{"structure"}); !ok {
		t.Fatal("runs_on label did not match")
	}
	if _, ok := cat.HoldPointRelease("if.slab-before-frame", []string{"fire-passive"}); ok {
		t.Fatal("a passage outside runs_on matched")
	}
	if _, ok := cat.HoldPointRelease("if.fire-water-supplies-sprinklers", []string{"fire-passive"}); ok {
		t.Fatal("an interface without a release matched")
	}
	// The release noul is not an evidence question: passages without a
	// program hold point must be asked exactly what they were before.
	for _, q := range cat.EvidenceQuestions([]string{"structure", "fire-passive"}) {
		if strings.HasPrefix(q.ID, "hold:") {
			t.Fatalf("release joined the evidence questions: %s", q.ID)
		}
	}
}

func TestHoldPointWithoutReleaseLoads(t *testing.T) {
	cat, err := knowledge.Load(releaseFixture(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.HoldPointRelease("if.slab-before-frame", []string{"structure"}); ok {
		t.Fatal("no release authored, yet one matched")
	}
}

func TestHoldPointReleaseRejectsBadShapes(t *testing.T) {
	cases := map[string]string{
		"choice":     strings.Replace(goodRelease, "type: noul", "type: choice", 1),
		"no status":  strings.Replace(goodRelease, "        status: draft\n", "", 1),
		"no runs_on": strings.Replace(goodRelease, "        runs_on: [structure]\n", "", 1),
		"unknown":    strings.Replace(goodRelease, "runs_on: [structure]", "runs_on: [nowhere]", 1),
		"criteria":   strings.Replace(goodRelease, "          false: It only requires, schedules or requests the inspection.\n", "", 1),
	}
	for name, release := range cases {
		if _, err := knowledge.Load(releaseFixture(t, release)); err == nil || !strings.Contains(err.Error(), "hold_point.release") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
