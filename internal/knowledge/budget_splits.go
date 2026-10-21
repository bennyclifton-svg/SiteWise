package knowledge

import (
	"fmt"
	"math/big"
	"path/filepath"
	"strconv"
)

// SplitPreliminaries is the share of preliminaries and margin. It has no
// cost line, so the spread shares it over the trade lines pro rata.
const SplitPreliminaries = "preliminaries"

// BudgetSplit is one profile's share of construction cost by building
// system, in percent (docs/evidence/2026-10-21-cost-split-research.md).
type BudgetSplit struct {
	ID         string             `yaml:"id"`
	Label      string             `yaml:"label"`
	Confidence string             `yaml:"confidence"`
	When       BudgetSplitWhen    `yaml:"when"`
	Basement   map[string]float64 `yaml:"basement"`
	Shares     map[string]float64 `yaml:"shares"`
	Basis      string             `yaml:"basis"`
}

// BudgetSplitWhen matches the profile. An empty list matches any value;
// storeys bounds are ignored when the storeys are unknown.
type BudgetSplitWhen struct {
	Class      []string `yaml:"class"`
	Subclass   []string `yaml:"subclass"`
	WorkType   []string `yaml:"work_type"`
	StoreysMin int      `yaml:"storeys_min"`
	StoreysMax int      `yaml:"storeys_max"`
}

// SplitProfile is what the profile says about the building, for choosing a
// split. Storeys is 0 when unknown.
type SplitProfile struct {
	Class     string
	Subclass  string
	WorkTypes []string
	Storeys   int
	Basement  bool
}

func (c *Catalog) BudgetSplits() []BudgetSplit { return c.budgetSplits }

// ChooseBudgetSplit returns the first split that matches the profile, with
// the basement scaling applied, as exact weights. A profile with no class
// or work type matches nothing: no split is guessed.
func (c *Catalog) ChooseBudgetSplit(p SplitProfile) (BudgetSplit, map[string]*big.Rat, bool) {
	if p.Class == "" || len(p.WorkTypes) == 0 {
		return BudgetSplit{}, nil, false
	}
	for _, s := range c.budgetSplits {
		w := s.When
		if !within(w.Class, p.Class) || len(w.Subclass) > 0 && !within(w.Subclass, p.Subclass) || !anyWithin(w.WorkType, p.WorkTypes) {
			continue
		}
		if p.Storeys > 0 && (w.StoreysMin > 0 && p.Storeys < w.StoreysMin || w.StoreysMax > 0 && p.Storeys > w.StoreysMax) {
			continue
		}
		if p.Storeys == 0 && w.StoreysMin > 0 {
			continue
		}
		weights := map[string]*big.Rat{}
		for system, share := range s.Shares {
			r, _ := new(big.Rat).SetString(strconv.FormatFloat(share, 'f', -1, 64))
			if f, ok := s.Basement[system]; ok && p.Basement {
				factor, _ := new(big.Rat).SetString(strconv.FormatFloat(f, 'f', -1, 64))
				r.Mul(r, factor)
			}
			weights[system] = r
		}
		return s, weights, true
	}
	return BudgetSplit{}, nil, false
}

func within(list []string, v string) bool {
	if len(list) == 0 {
		return true
	}
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func anyWithin(list, values []string) bool {
	if len(list) == 0 {
		return true
	}
	for _, v := range values {
		if within(list, v) {
			return true
		}
	}
	return false
}

func (c *Catalog) loadBudgetSplits(root string) error {
	path := filepath.Join(root, "costs", "budget_splits.yaml")
	if ok, e := exists(path); !ok {
		return e
	}
	var file struct {
		Version  int
		Status   string
		Verified bool
		Sources  []map[string]string
		Splits   []BudgetSplit
	}
	if e := c.unmarshal(path, &file); e != nil {
		return e
	}
	if file.Version != 1 || file.Status != "draft" && file.Status != "reviewed" || len(file.Sources) == 0 || file.Status == "draft" && file.Verified {
		return fmt.Errorf("%s: invalid metadata", path)
	}
	seen := map[string]bool{}
	for _, s := range file.Splits {
		if s.ID == "" || seen[s.ID] || s.Label == "" || s.Basis == "" || len(s.When.Class) == 0 || len(s.When.WorkType) == 0 {
			return fmt.Errorf("%s: split %q needs a unique id, label, basis, class and work type", path, s.ID)
		}
		seen[s.ID] = true
		switch s.Confidence {
		case "low", "medium", "high":
		default:
			return fmt.Errorf("%s: split %s confidence %q", path, s.ID, s.Confidence)
		}
		sum := 0.0
		for system, share := range s.Shares {
			if _, ok := c.System(system); !ok && system != SplitPreliminaries {
				return fmt.Errorf("%s: split %s names unknown system %q", path, s.ID, system)
			}
			if share < 0 {
				return fmt.Errorf("%s: split %s share for %s is negative", path, s.ID, system)
			}
			sum += share
		}
		// Shares are renormalised, but a sum far from 100 is a typo.
		if sum < 99 || sum > 101 {
			return fmt.Errorf("%s: split %s shares sum to %.2f, not 100", path, s.ID, sum)
		}
		for system, f := range s.Basement {
			if _, ok := s.Shares[system]; !ok || f <= 0 {
				return fmt.Errorf("%s: split %s basement scales %q, which has no share", path, s.ID, system)
			}
		}
	}
	c.budgetSplits = file.Splits
	return nil
}
