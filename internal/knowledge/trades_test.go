package knowledge

import (
	"strings"
	"testing"
)

func TestTradesLoadFromKnowledge(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	trades := cat.Trades()
	if len(trades) < 55 {
		t.Fatalf("got %d trades", len(trades))
	}
	byID := map[string]Trade{}
	for _, tr := range trades {
		byID[tr.ID] = tr
	}
	c, ok := byID["trade.concreting"]
	if !ok || c.Label != "Concreting" || c.Kind != "trade" || len(c.Builds) == 0 {
		t.Fatalf("concreting: %+v", c)
	}
	// The nine trades filing already knows keep their ids and labels.
	if byID["trade.main_works"].Label != "Main Works" || byID["supplier.windows_glazing"].Kind != "supply" {
		t.Fatal("existing trades changed")
	}
	if len(cat.TradeBundles()) == 0 {
		t.Fatal("no bundles")
	}
}

func TestParseTradesRejects(t *testing.T) {
	systems := func(id string) bool { return id == "structure" || id == "structure.concrete" }
	filing := map[string]string{"trade.main_works": "Main Works"}
	head := "version: 1\nstatus: draft\ntrades:\n  - {id: trade.main_works, label: Main Works, kind: trade, used_on: both, builds: [all]}\n"
	ok := head + "  - {id: trade.concreting, label: Concreting, kind: trade, used_on: both, builds: [structure.concrete]}\n"
	if _, _, err := parseTrades([]byte(ok), systems, filing); err != nil {
		t.Fatalf("valid file refused: %v", err)
	}
	cases := map[string]string{
		"duplicate id":      ok + "  - {id: trade.concreting, label: Again, kind: trade, used_on: both, builds: []}\n",
		"bad prefix":        head + "  - {id: consultant.x, label: X, kind: trade, used_on: both, builds: []}\n",
		"supply prefix":     head + "  - {id: trade.x, label: X, kind: supply, used_on: both, builds: []}\n",
		"bad used_on":       head + "  - {id: trade.x, label: X, kind: trade, used_on: sometimes, builds: []}\n",
		"unknown system":    head + "  - {id: trade.x, label: X, kind: trade, used_on: both, builds: [roof]}\n",
		"all on a trade":    head + "  - {id: trade.x, label: X, kind: trade, used_on: both, builds: [all]}\n",
		"missing label":     head + "  - {id: trade.x, kind: trade, used_on: both, builds: []}\n",
		"filing trade gone": "version: 1\nstatus: draft\ntrades:\n  - {id: trade.x, label: X, kind: trade, used_on: both, builds: []}\n",
		"bundle member":     ok + "bundles:\n  - {id: b, label: B, members: [trade.concreting, trade.nope]}\n",
		"bundle of one":     ok + "bundles:\n  - {id: b, label: B, members: [trade.concreting]}\n",
		"bundle with main":  ok + "bundles:\n  - {id: b, label: B, members: [trade.concreting, trade.main_works]}\n",
	}
	for name, body := range cases {
		if _, _, err := parseTrades([]byte(body), systems, filing); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	relabelled := map[string]string{"trade.main_works": "Head contract"}
	if _, _, err := parseTrades([]byte(ok), systems, relabelled); err == nil || !strings.Contains(err.Error(), "label") {
		t.Errorf("label drift from filing accepted: %v", err)
	}
}
