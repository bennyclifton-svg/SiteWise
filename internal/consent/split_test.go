package consent

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"sitewise/internal/delivery"
)

func fixture(t *testing.T) []Page {
	t.Helper()
	body, err := os.ReadFile("testdata/seven-hills-notice.md")
	if err != nil {
		t.Fatal(err)
	}
	return []Page{{Page: 1, Text: string(body)}}
}

// The public fixture is a table with a No. and a When column.
func TestSplitSevenHillsTable(t *testing.T) {
	pages := fixture(t)
	if !Detect(pages) {
		t.Fatal("not detected")
	}
	n := Split(pages)
	if n.ConsentRef != "DA/2025/0733" || n.Determined != "2025-11-27" || n.Modification {
		t.Fatalf("reference %q date %q modification %v", n.ConsentRef, n.Determined, n.Modification)
	}
	var numbers, stages []string
	for _, c := range n.Conditions {
		numbers = append(numbers, c.Number)
		stages = append(stages, c.Stage)
	}
	if want := []string{"6", "9", "12", "18", "23", "27", "31"}; !reflect.DeepEqual(numbers, want) {
		t.Fatalf("numbers %v", numbers)
	}
	want := []string{"before_cc", "before_cc", "before_work", "before_occupation", "during_works", "during_works", "before_subdivision"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("stages %v", stages)
	}
	if n.Conditions[2].Text != "Tree protection installed and certified by the project arborist" {
		t.Fatalf("text %q", n.Conditions[2].Text)
	}
	// The notice prints a selection, so the numbers in between show as gaps.
	if len(n.Gaps) == 0 || n.Gaps[0] != "1–5" {
		t.Fatalf("gaps %v", n.Gaps)
	}
}

// petershamLike is a synthetic notice in the layout the plan describes for
// Petersham: upper-case stage headings, a "Reason:" line each, footers on
// every page, a cross-reference, (a) items, and advisory notes after.
func petershamLike(count int) []Page {
	stages := []string{"GENERAL CONDITIONS", "PRIOR TO THE ISSUE OF A CONSTRUCTION CERTIFICATE", "PRIOR TO COMMENCEMENT OF WORKS", "DURING CONSTRUCTION", "PRIOR TO THE ISSUE OF AN OCCUPATION CERTIFICATE", "ONGOING CONDITIONS"}
	perPage := (count + 24) / 25
	var pages []Page
	var b strings.Builder
	page := 1
	n := 0
	for s, st := range stages {
		b.WriteString(st + "\n\n")
		share := count / len(stages)
		if s == len(stages)-1 {
			share = count - n
		}
		for i := 0; i < share; i++ {
			n++
			fmt.Fprintf(&b, "%d. Condition heading %d\n", n, n)
			fmt.Fprintf(&b, "The applicant must comply with the requirements of condition %d and the following:\n", max(1, n-1))
			b.WriteString("(a) the works are to be carried out\n12 metres from the boundary at all times\n(b) as approved.\n")
			b.WriteString("Reason: To ensure compliance.\n\n")
			if n%perPage == 0 {
				fmt.Fprintf(&b, "Document Set ID: 7738%d\nPage %d of 25\n", page, page)
				pages = append(pages, Page{Page: page, Text: b.String()})
				b.Reset()
				page++
				b.WriteString("Inner West Council   DA/2016/00123\n")
			}
		}
	}
	b.WriteString("ADVISORY NOTES\n\n1. The applicant is advised to contact Sydney Water.\n2. Hours of work apply.\n3. A further note.\n")
	fmt.Fprintf(&b, "Document Set ID: 7738%d\nPage %d of 25\n", page, page)
	pages = append(pages, Page{Page: page, Text: b.String()})
	pages[0].Text = "NOTICE OF DETERMINATION\nDevelopment Application No: DA/2016/00123\nDate of determination: 18 October 2016\n\n" + pages[0].Text
	return pages
}

func TestSplitUpperCaseHeadingsReasonsAndNotes(t *testing.T) {
	pages := petershamLike(102)
	if !Detect(pages) {
		t.Fatal("not detected")
	}
	n := Split(pages)
	if len(n.Conditions) != 102 {
		t.Fatalf("got %d conditions", len(n.Conditions))
	}
	for i, c := range n.Conditions {
		if c.Number != fmt.Sprint(i+1) || c.Heading != fmt.Sprintf("Condition heading %d", i+1) {
			t.Fatalf("condition %d is %q %q", i+1, c.Number, c.Heading)
		}
		if !strings.Contains(c.Text, "Reason: To ensure compliance.") || !strings.Contains(c.Text, "(a) the works") || strings.Contains(c.Text, "Page ") || strings.Contains(c.Text, "Document Set ID") || strings.Contains(c.Text, "Inner West") {
			t.Fatalf("condition %d text %q", i+1, c.Text)
		}
		if c.Page < 1 {
			t.Fatalf("condition %d has no page", i+1)
		}
	}
	if n.Conditions[0].Stage != "" || n.Conditions[0].StageHeading != "GENERAL CONDITIONS" {
		t.Fatalf("general %+v", n.Conditions[0])
	}
	if n.Conditions[20].Stage != delivery.GateConstructionCertificate || n.Conditions[40].Stage != delivery.GateWorkStarts ||
		n.Conditions[60].Stage != delivery.GateDuringWorks || n.Conditions[80].Stage != delivery.GateOccupation || n.Conditions[101].Stage != delivery.GateOngoing {
		t.Fatalf("stages %q %q %q %q %q", n.Conditions[20].Stage, n.Conditions[40].Stage, n.Conditions[60].Stage, n.Conditions[80].Stage, n.Conditions[101].Stage)
	}
	if strings.Contains(n.Conditions[101].Text, "Sydney Water") {
		t.Fatal("advisory notes read as a condition")
	}
	if n.ConsentRef != "DA/2016/00123" || n.Determined != "2016-10-18" || len(n.Gaps) != 0 {
		t.Fatalf("ref %q date %q gaps %v", n.ConsentRef, n.Determined, n.Gaps)
	}
}

// A second council's layout: mixed-case stage headings, lettered
// numbering, and "Operational Conditions" for the ongoing stage.
func TestSplitMixedCaseAndPrefixes(t *testing.T) {
	text := `Notice of Determination of a Development Application
Determination No. 0167/21
Date of determination: 8 November 2021

Prior to Construction Certificate
A1 Stormwater management
A stormwater plan is to be submitted to the Certifier.
A2 Waste
Waste must be stored on site.

Deferred Commencement
B1 Owner's consent
Evidence of the owner's consent is to be provided.

Operational Conditions
C1 Hours of operation
The amenities may operate between 7 am and 10 pm.
C2 Lighting
External lighting is to comply with AS 4282 as set out in condition A1.

Notes
1. Fees are payable.
`
	n := Split([]Page{{Page: 1, Text: text}})
	var got []string
	for _, c := range n.Conditions {
		got = append(got, c.Number+":"+c.Heading+":"+c.Stage)
	}
	want := []string{"A1:Stormwater management:before_cc", "A2:Waste:before_cc", "B1:Owner's consent:before_work", "C1:Hours of operation:ongoing", "C2:Lighting:ongoing"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if !n.Conditions[2].Deferred {
		t.Fatal("deferred commencement not flagged")
	}
	if !Detect([]Page{{Page: 1, Text: text}}) {
		t.Fatal("not detected")
	}
}

// A modification restates conditions with "Deleted" and amended markers,
// and a scan's missing numbers are gaps.
func TestSplitModificationMarkersAndGaps(t *testing.T) {
	mod := `Notice of Determination of a Modification Application
Modification of development consent DA/2016/00123 under section 4.55
Modification No. MOD/2023/0177
Date of determination: 4 August 2023

Condition 13 Deleted
Condition 46 Landscape plan (Amended MOD/2023/0177)
A landscape plan prepared by a landscape architect is to be submitted.
Condition 50 Waste storage
Waste is stored in the basement.
`
	n := Split([]Page{{Page: 1, Text: mod}})
	if !n.Modification || n.ConsentRef != "DA/2016/00123" || n.ModRef != "MOD/2023/0177" || n.Determined != "2023-08-04" {
		t.Fatalf("modification %+v", n)
	}
	if len(n.Conditions) != 3 || n.Conditions[0].Marker != "deleted" || n.Conditions[1].Marker != "changed" || n.Conditions[2].Marker != "" || len(n.Gaps) != 0 {
		t.Fatalf("conditions %+v gaps %v", n.Conditions, n.Gaps)
	}
	scan := "NOTICE OF DETERMINATION\n"
	for i := 1; i <= 30; i++ {
		if i == 14 || i == 22 {
			continue
		}
		scan += fmt.Sprintf("%d. Heading %d\nWords of condition %d.\n", i, i, i)
	}
	s := Split([]Page{{Page: 1, Text: scan}})
	if len(s.Conditions) != 28 || !reflect.DeepEqual(s.Gaps, []string{"14", "22"}) {
		t.Fatalf("scan %d conditions, gaps %v", len(s.Conditions), s.Gaps)
	}
}

func TestDetectRefusesWhatIsNotAConsent(t *testing.T) {
	for _, text := range []string{
		"Roads Act 1993 section 138 approval\n1. Traffic control\n2. Footpath\n3. Hours\n",
		"NOTICE OF DETERMINATION\nThe application is refused.\n",
		"",
	} {
		if Detect([]Page{{Page: 1, Text: text}}) {
			t.Errorf("detected %q", text)
		}
	}
}

func TestKeptCutsLongText(t *testing.T) {
	c := Condition{Text: strings.Repeat("w", delivery.MaxConditionText+500), Page: 3, EndPage: 5}
	got := Kept(c)
	if len([]rune(got)) != delivery.MaxConditionText || !strings.HasSuffix(got, "continues on page 5") {
		t.Fatalf("kept %d runes, ends %q", len([]rune(got)), got[len(got)-30:])
	}
}

// TestSplitBudget holds the split's speed row: a 25-page, 102-condition
// notice in 20 ms p50 and 50 ms p90.
func TestSplitBudget(t *testing.T) {
	pages := petershamLike(102)
	var samples []time.Duration
	for i := 0; i < 30; i++ {
		start := time.Now()
		if n := Split(pages); len(n.Conditions) != 102 {
			t.Fatal(len(n.Conditions))
		}
		samples = append(samples, time.Since(start))
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p50, p90 := samples[len(samples)/2], samples[len(samples)*9/10]
	t.Logf("split p50 %v p90 %v", p50, p90)
	if p50 > 20*time.Millisecond || p90 > 50*time.Millisecond {
		t.Fatalf("split p50 %v p90 %v over 20 / 50 ms", p50, p90)
	}
}
