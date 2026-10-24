package store_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

func TestSubmissionAutomaticPackageMatching(t *testing.T) {
	for _, trigger := range []string{"discipline", "package", "package_discipline", "late_read", "manual_blank", "manual_package", "ambiguous"} {
		t.Run(trigger, func(t *testing.T) {
			ctx := context.Background()
			s, _, _ := workStore(t)
			pool := rawPool(t)
			id, err := s.OpenSubmission(ctx, orgA, projectA, docA, "", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			complete := func() {
				ok, err := s.CommitSubmissionRead(ctx, orgA, id, proposals.Outcome{Decisions: []proposals.Decision{{Field: proposals.FieldKind, Value: proposals.KindFeeProposal, Band: proposals.BandGreen, DecidedBy: proposals.ByRule}}}, false, time.Now())
				if err != nil || !ok {
					t.Fatalf("complete: %v %v", ok, err)
				}
			}
			if trigger != "late_read" {
				complete()
			}
			if trigger == "manual_blank" {
				v := ""
				if err := s.EditSubmission(ctx, orgA, projectA, id, store.SubmissionEdit{PackageID: &v}); err != nil {
					t.Fatal(err)
				}
			}
			create := func(d string) procurement.Package {
				p, _ := procurement.DisciplinePackage(d, "Consultant")
				p, err := s.CreatePackage(ctx, orgA, projectA, userA, p)
				if err != nil {
					t.Fatal(err)
				}
				return p
			}
			if trigger == "package" || trigger == "late_read" {
				if err := s.CorrectDecision(ctx, orgA, docA, "discipline", "consultant.architect"); err != nil {
					t.Fatal(err)
				}
			}
			discipline := "consultant.architect"
			if trigger == "package_discipline" {
				discipline = "consultant.civil"
			}
			p := create(discipline)
			want := p.ID
			var duplicate procurement.Package
			if trigger == "ambiguous" {
				duplicate = create("consultant.architect")
				want = ""
			}
			if trigger == "manual_package" {
				other := create("consultant.civil")
				want = other.ID
				if err := s.EditSubmission(ctx, orgA, projectA, id, store.SubmissionEdit{PackageID: &other.ID}); err != nil {
					t.Fatal(err)
				}
			}
			if trigger == "manual_blank" {
				want = ""
			}
			if err := s.CorrectDecision(ctx, orgA, docA, "discipline", "consultant.architect"); err != nil {
				t.Fatal(err)
			}
			if trigger == "late_read" {
				complete()
			}
			if trigger == "package_discipline" {
				discipline = "consultant.architect"
				if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: p.Version, DisciplineID: &discipline}); err != nil {
					t.Fatal(err)
				}
			}
			check := func(want string) {
				var got string
				if err := pool.QueryRow(ctx, `SELECT COALESCE(package_id::text,'') FROM submissions WHERE org_id=$1 AND id=$2`, orgA, id).Scan(&got); err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Fatalf("package = %q, want %q", got, want)
				}
			}
			check(want)
			if trigger == "ambiguous" {
				retired := true
				if _, err := s.PatchPackage(ctx, orgA, projectA, duplicate.ID, userA, procurement.PackagePatch{Version: duplicate.Version, Retired: &retired}); err != nil {
					t.Fatal(err)
				}
				check(p.ID)
			}
			if err := s.CorrectDecision(ctx, orgB, docA, "discipline", "consultant.civil"); err == nil {
				t.Fatal("wrong org allowed")
			}
			if trigger != "ambiguous" {
				check(want)
			}
		})
	}
}

// Exercise real matching writes, not an empty-project SQL microbenchmark.
func TestSubmissionMatchingSpeedBudgets(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pool := rawPool(t)
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CommitSubmissionRead(ctx, orgA, id, proposals.Outcome{}, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	p, _ := procurement.DisciplinePackage("consultant.architect", "Architect")
	p, err = s.CreatePackage(ctx, orgA, projectA, userA, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"field_correction", "packages_write"} {
		var samples []time.Duration
		for i := 0; i < 20; i++ {
			if _, err := pool.Exec(ctx, `UPDATE submissions SET package_id=NULL WHERE org_id=$1 AND id=$2`, orgA, id); err != nil {
				t.Fatal(err)
			}
			began := time.Now()
			if path == "field_correction" {
				err = s.CorrectDecision(ctx, orgA, docA, "discipline", "consultant.architect")
			} else {
				title := "Architect"
				p, err = s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: p.Version, Title: &title})
			}
			samples = append(samples, time.Since(began))
			if err != nil {
				t.Fatal(err)
			}
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		p50, p90 := samples[9], samples[17]
		limit50, limit90 := 100*time.Millisecond, 250*time.Millisecond
		if path == "field_correction" {
			limit50, limit90 = 50*time.Millisecond, 150*time.Millisecond
		}
		t.Logf("%s n=20 p50=%v p90=%v", path, p50, p90)
		if p50 > limit50 || p90 > limit90 {
			t.Errorf("%s exceeds %v/%v", path, limit50, limit90)
		}
	}
}
