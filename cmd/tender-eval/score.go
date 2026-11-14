package main

import (
	"sort"
	"strings"

	"sitewise/internal/invoices"
	"sitewise/internal/tender"
)

type caseScore struct {
	Sum            string // "right", "flagged" or "silent"
	KeyDollars     int64  // sum of |amount| of the key's counted lines
	RowDollars     int64  // of those, matched to a counted reading line on the key's row
	FoundDollars   int64  // of those, matched to any reading line with the same amount
	OutsideCounted int    // key lines that are outside the price but counted, unflagged
	AmountsAsked   int    // reading lines the model gave an amount for
	AmountsKept    int    // of those, kept by R3
	Lines          int
	KeyLines       int
	Misses         []string // private: printed only with -show
}

// A dollar, the tolerance the reconciler itself uses.
const scoreTol = 100

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func hasFlag(flags []tender.Flag, codes ...string) bool {
	for _, f := range flags {
		for _, c := range codes {
			if f.Code == c {
				return true
			}
		}
	}
	return false
}

func sharedWords(a, b string) int {
	seen := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(a)) {
		seen[w] = true
	}
	n := 0
	for _, w := range strings.Fields(strings.ToLower(b)) {
		if seen[w] {
			n++
		}
	}
	return n
}

// scoreCase compares one reading with its key (slice 1, task 5).
func scoreCase(k keyCase, r tender.Reading) caseScore {
	s := caseScore{Lines: len(r.Lines), KeyLines: len(k.Lines)}
	// 1. The tender sum.
	var want *int64
	if strings.TrimSpace(k.TenderSum) != "" {
		if c, ok, err := invoices.ParseCents(k.TenderSum); err == nil && ok {
			want = &c
		}
	}
	switch {
	case want == nil && r.TenderSum == nil, want != nil && r.TenderSum != nil && abs(*want-*r.TenderSum) <= scoreTol:
		s.Sum = "right"
	case r.TenderSum == nil || hasFlag(r.Flags, tender.FlagCheckTenderSum, tender.FlagNoTotalPrinted, tender.FlagLinesShort, tender.FlagLinesOver, tender.FlagIncomplete):
		s.Sum = "flagged"
	default:
		s.Sum = "silent"
	}
	// 2. Match key lines to reading lines, largest first, each once.
	type keyed struct {
		line  keyLine
		cents int64
	}
	var keys []keyed
	for _, kl := range k.Lines {
		c, ok, err := invoices.ParseCents(kl.Amount)
		if err != nil || !ok {
			continue
		}
		keys = append(keys, keyed{kl, c})
	}
	sort.SliceStable(keys, func(i, j int) bool { return abs(keys[i].cents) > abs(keys[j].cents) })
	used := map[int]bool{}
	for _, kl := range keys {
		best, bestRank := -1, -1
		for i, l := range r.Lines {
			if used[i] || l.Cents == nil || abs(*l.Cents-kl.cents) > scoreTol {
				continue
			}
			rank := sharedWords(kl.line.Label, l.Label)
			if l.Row == kl.line.Row {
				rank += 1000
			}
			if rank > bestRank {
				best, bestRank = i, rank
			}
		}
		if best >= 0 {
			used[best] = true
		}
		// 3. Rows.
		if kl.line.Counted {
			s.KeyDollars += abs(kl.cents)
			switch {
			case best < 0:
				s.Misses = append(s.Misses, "missing: "+kl.line.Label+" "+kl.line.Amount)
			case r.Lines[best].Counted && r.Lines[best].Row == kl.line.Row:
				s.FoundDollars += abs(kl.cents)
				s.RowDollars += abs(kl.cents)
			default:
				s.FoundDollars += abs(kl.cents)
				why := "row " + r.Lines[best].Row + ", want " + kl.line.Row
				if !r.Lines[best].Counted {
					why = "not counted"
				}
				s.Misses = append(s.Misses, why+": "+kl.line.Label+" "+kl.line.Amount)
			}
			continue
		}
		// 4. Outside money that was counted with no flag saying so: neither
		// on the line, nor on the column as lines adding to more than the sum.
		if best >= 0 && r.Lines[best].Counted && !hasFlag(r.Lines[best].Flags, tender.FlagCountedToMatch) && !hasFlag(r.Flags, tender.FlagLinesOver) {
			s.OutsideCounted++
			s.Misses = append(s.Misses, "outside but counted: "+kl.line.Label+" "+kl.line.Amount)
		}
	}
	// 5. Amounts the model gave and R3 kept.
	// A line with wording where an amount should be ("at cost", "hourly
	// rates") was never an amount, so R3 did not throw one away.
	for _, l := range r.Lines {
		if l.State != tender.StatePriced {
			continue
		}
		thrown := false
		for _, f := range l.Flags {
			thrown = thrown || (f.Code == tender.FlagAmountNotFound && strings.HasPrefix(f.Text, "This amount is not in the text"))
		}
		if l.Cents == nil && !thrown {
			continue
		}
		s.AmountsAsked++
		if !thrown {
			s.AmountsKept++
		}
	}
	return s
}
