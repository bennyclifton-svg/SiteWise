package main

import (
	"context"
	"path/filepath"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/llm"
	"sitewise/internal/tender"
)

// Every figure in these tests is invented.

func TestRowsForMainWorksHasPreliminaries(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := rowsFor(cat, []string{"main-works"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		found = found || r.Label == "Preliminaries and site establishment"
	}
	if !found || len(rows) < 5 {
		t.Fatalf("%d rows, preliminaries %v", len(rows), found)
	}
	both, err := rowsFor(cat, []string{"civil", "structural"})
	if err != nil || len(both) == 0 || both[0].Key[:6] != "civil:" {
		t.Fatalf("combined rows: %v %v", both, err)
	}
}

func c(v int64) *int64 { return &v }

func line(label string, cents int64, row string, counted bool, flags ...string) tender.Line {
	l := tender.Line{Label: label, Cents: c(cents), Row: row, Counted: counted, State: tender.StatePriced, Kind: "base"}
	for _, f := range flags {
		l.Flags = append(l.Flags, tender.Flag{Code: f})
	}
	return l
}

func TestScoreRightSum(t *testing.T) {
	if s := scoreCase(keyCase{TenderSum: "100000.00"}, tender.Reading{TenderSum: c(10000000)}); s.Sum != "right" {
		t.Fatal(s.Sum)
	}
}

func TestScoreNoSumExpected(t *testing.T) {
	if s := scoreCase(keyCase{}, tender.Reading{}); s.Sum != "right" {
		t.Fatal(s.Sum)
	}
}

func TestScoreWrongSumWithAFlagIsFlagged(t *testing.T) {
	r := tender.Reading{TenderSum: c(9000000), Flags: []tender.Flag{{Code: tender.FlagCheckTenderSum}}}
	if s := scoreCase(keyCase{TenderSum: "100000.00"}, r); s.Sum != "flagged" {
		t.Fatal(s.Sum)
	}
}

func TestScoreWrongSumWithNoFlagIsSilent(t *testing.T) {
	if s := scoreCase(keyCase{TenderSum: "100000.00"}, tender.Reading{TenderSum: c(9000000)}); s.Sum != "silent" {
		t.Fatal(s.Sum)
	}
}

var threeLines = []keyLine{{Label: "Frame", Amount: "50000.00", Row: "A", Counted: true}, {Label: "Roof", Amount: "30000.00", Row: "B", Counted: true}, {Label: "Paint", Amount: "20000.00", Row: "C", Counted: true}}

func TestScoreDollarsInTheRightRow(t *testing.T) {
	r := tender.Reading{Lines: []tender.Line{line("Frame", 5000000, "A", true), line("Roof", 3000000, "B", true), line("Paint", 2000000, "B", true)}}
	s := scoreCase(keyCase{Lines: threeLines}, r)
	if s.RowDollars != 80000_00 || s.KeyDollars != 100000_00 || s.FoundDollars != 100000_00 {
		t.Fatalf("%+v", s)
	}
}

func TestScoreAMissingLineCountsAgainstRows(t *testing.T) {
	r := tender.Reading{Lines: []tender.Line{line("Frame", 5000000, "A", true), line("Paint", 2000000, "C", true)}}
	if s := scoreCase(keyCase{Lines: threeLines}, r); s.RowDollars != 70000_00 {
		t.Fatalf("%+v", s)
	}
}

func TestScoreOutsideMoneyCounted(t *testing.T) {
	k := keyCase{Lines: []keyLine{{Label: "Option", Amount: "18000.00", Kind: "option"}}}
	if s := scoreCase(k, tender.Reading{Lines: []tender.Line{line("Option", 1800000, "", true)}}); s.OutsideCounted != 1 {
		t.Fatalf("%+v", s)
	}
	if s := scoreCase(k, tender.Reading{Lines: []tender.Line{line("Option", 1800000, "", true, tender.FlagCountedToMatch)}}); s.OutsideCounted != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestScoreLinesAreMatchedOnce(t *testing.T) {
	k := keyCase{Lines: []keyLine{{Label: "Stage 1", Amount: "10000.00", Row: "A", Counted: true}, {Label: "Stage 2", Amount: "10000.00", Row: "B", Counted: true}}}
	r := tender.Reading{Lines: []tender.Line{line("Stage 1", 1000000, "A", true), line("Stage 2", 1000000, "B", true)}}
	if s := scoreCase(k, r); s.RowDollars != 20000_00 {
		t.Fatalf("%+v", s)
	}
	one := tender.Reading{Lines: []tender.Line{line("Stage 1", 1000000, "A", true)}}
	if s := scoreCase(k, one); s.FoundDollars != 10000_00 {
		t.Fatalf("one reading line matched twice: %+v", s)
	}
}

func TestScoreAmountsKept(t *testing.T) {
	var r tender.Reading
	for i := 0; i < 9; i++ {
		r.Lines = append(r.Lines, line("Line", 100, "A", true))
	}
	lost := tender.Line{Label: "Lost", State: tender.StatePriced, Flags: []tender.Flag{{Code: tender.FlagAmountNotFound, Text: "This amount is not in the text of the page. Type it from the document."}}}
	r.Lines = append(r.Lines, tender.Line{Label: "At cost", State: tender.StatePriced, Flags: []tender.Flag{{Code: tender.FlagAmountNotFound, Text: "No amount could be read for this line. Type it from the document."}}})
	r.Lines = append(r.Lines, lost)
	if s := scoreCase(keyCase{}, r); s.AmountsKept != 9 || s.AmountsAsked != 10 {
		t.Fatalf("%+v", s)
	}
}

type fixedAsker struct{ calls int }

func (f *fixedAsker) Ask(context.Context, llm.Request) (llm.Response, error) {
	f.calls++
	return llm.Response{Text: "answer", StopReason: "end_turn", InTokens: 7, OutTokens: 9, Latency: 1500e6}, nil
}

func TestRecorderStoresAndReplays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recording.jsonl")
	live := &fixedAsker{}
	rec, err := newReplayer(path, live, false)
	if err != nil {
		t.Fatal(err)
	}
	req := llm.Request{Model: "m", System: "s", Parts: []llm.Part{{Text: "page one"}}}
	if _, err := rec.Ask(context.Background(), req); err != nil || live.calls != 1 {
		t.Fatalf("err %v calls %d", err, live.calls)
	}
	replay, err := newReplayer(path, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	res, err := replay.Ask(context.Background(), req)
	if err != nil || res.Text != "answer" || res.OutTokens != 9 || res.Latency.Milliseconds() != 1500 {
		t.Fatalf("%+v %v", res, err)
	}
	other := req
	other.Parts = []llm.Part{{Text: "page two"}}
	if _, err := replay.Ask(context.Background(), other); err != errReplayMiss {
		t.Fatalf("err %v", err)
	}
}

func TestRecordingKeyIgnoresNothingThatMatters(t *testing.T) {
	a := llm.Request{Model: "m", System: "s", Parts: []llm.Part{{Text: "one"}, {Text: "two"}}}
	b := llm.Request{Model: "m", System: "s", Parts: []llm.Part{{Text: "one"}, {Text: "two!"}}}
	if requestKey(a) == requestKey(b) || requestKey(a) != requestKey(a) {
		t.Fatal("request keys do not tell requests apart")
	}
}
