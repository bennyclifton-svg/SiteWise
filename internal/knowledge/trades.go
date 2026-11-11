package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	intakevocab "sitewise/data/intake"
)

// Trade is one trade or supply a project can let (knowledge/works/trades.yaml,
// docs/plans/2026-11-10-trades-catalogue.md). This is the Procurement list,
// not the filing vocabulary: Jev never chooses from it, so it can grow
// without touching filing.
type Trade struct {
	ID       string   `yaml:"id" json:"id"`
	Label    string   `yaml:"label" json:"label"`
	Kind     string   `yaml:"kind" json:"kind"`
	UsedOn   string   `yaml:"used_on" json:"used_on"`
	Existing bool     `yaml:"existing" json:"-"`
	Builds   []string `yaml:"builds" json:"builds"`
	Detail   string   `yaml:"detail" json:"-"`
}

// TradeBundle is a suggested Combine: a prompt only, never a limit.
type TradeBundle struct {
	ID      string   `yaml:"id" json:"id"`
	Label   string   `yaml:"label" json:"label"`
	Members []string `yaml:"members" json:"members"`
}

func (c *Catalog) Trades() []Trade             { return c.trades }
func (c *Catalog) TradeBundles() []TradeBundle { return c.tradeBundles }

// Trade finds one trade or supply by id.
func (c *Catalog) Trade(id string) (Trade, bool) {
	for _, t := range c.trades {
		if t.ID == id {
			return t, true
		}
	}
	return Trade{}, false
}

func (c *Catalog) loadTrades(root string) error {
	path := filepath.Join(root, "works", "trades.yaml")
	if ok, err := exists(path); !ok {
		return err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := c.remember(path, body); err != nil {
		return err
	}
	filing, err := filingTrades()
	if err != nil {
		return err
	}
	system := func(id string) bool { _, ok := c.System(id); return ok }
	trades, bundles, err := parseTrades(body, system, filing)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	c.trades, c.tradeBundles = trades, bundles
	return nil
}

// filingTrades is every trade and supply the filing vocabulary holds. The
// trades list must keep each one with the same label, so a package filed
// or created under one of them reads the same in both places.
func filingTrades() (map[string]string, error) {
	var vocab struct{ Disciplines []struct{ ID, Label string } }
	if err := json.Unmarshal(intakevocab.Disciplines, &vocab); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, d := range vocab.Disciplines {
		if strings.HasPrefix(d.ID, "trade.") || strings.HasPrefix(d.ID, "supplier.") {
			out[d.ID] = d.Label
		}
	}
	return out, nil
}

func parseTrades(body []byte, system func(string) bool, filing map[string]string) ([]Trade, []TradeBundle, error) {
	var file struct {
		Version int              `yaml:"version"`
		Status  string           `yaml:"status"`
		Sources []map[string]any `yaml:"sources"`
		Trades  []Trade          `yaml:"trades"`
		Bundles []TradeBundle    `yaml:"bundles"`
	}
	if err := strictYAML(body, &file); err != nil {
		return nil, nil, err
	}
	if file.Version != 1 || (file.Status != "draft" && file.Status != "reviewed") {
		return nil, nil, fmt.Errorf("needs version 1 and status draft or reviewed")
	}
	seen := map[string]bool{}
	for _, t := range file.Trades {
		if err := checkTrade(t, system); err != nil {
			return nil, nil, err
		}
		if seen[t.ID] {
			return nil, nil, fmt.Errorf("duplicate trade %s", t.ID)
		}
		seen[t.ID] = true
		if want, ok := filing[t.ID]; ok && want != t.Label {
			return nil, nil, fmt.Errorf("trade %s: label %q must match filing label %q", t.ID, t.Label, want)
		}
	}
	for id := range filing {
		if !seen[id] {
			return nil, nil, fmt.Errorf("filing trade %s is missing", id)
		}
	}
	bundles := map[string]bool{}
	for _, b := range file.Bundles {
		if b.ID == "" || b.Label == "" || bundles[b.ID] || len(b.Members) < 2 {
			return nil, nil, fmt.Errorf("bundle %q needs an id, a label and two or more members", b.ID)
		}
		bundles[b.ID] = true
		for _, m := range b.Members {
			// Main Works already holds the whole building; combining it would
			// count work twice.
			if !seen[m] || m == "trade.main_works" {
				return nil, nil, fmt.Errorf("bundle %s: member %s", b.ID, m)
			}
		}
	}
	return file.Trades, file.Bundles, nil
}

func checkTrade(t Trade, system func(string) bool) error {
	if t.ID == "" || t.Label == "" {
		return fmt.Errorf("trade %q needs an id and a label", t.ID)
	}
	switch {
	case t.Kind == "trade" && strings.HasPrefix(t.ID, "trade."):
	case t.Kind == "supply" && strings.HasPrefix(t.ID, "supplier."):
	default:
		return fmt.Errorf("trade %s: kind trade needs trade.*, kind supply needs supplier.*", t.ID)
	}
	if t.UsedOn != "house" && t.UsedOn != "commercial" && t.UsedOn != "both" {
		return fmt.Errorf("trade %s: used_on must be house, commercial or both", t.ID)
	}
	for _, s := range t.Builds {
		if s == "all" && t.ID == "trade.main_works" {
			continue
		}
		if !system(s) {
			return fmt.Errorf("trade %s: builds unknown system %s", t.ID, s)
		}
	}
	return nil
}
