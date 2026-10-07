package knowledge

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"math/big"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type CostBenchmark struct {
	ID          string              `yaml:"id" json:"id"`
	Version     int                 `yaml:"version" json:"version"`
	Basis       string              `yaml:"basis" json:"basis"`
	Unit        string              `yaml:"unit" json:"unit"`
	Amount      string              `yaml:"amount" json:"amount"`
	Currency    string              `yaml:"currency" json:"currency"`
	TaxBasis    string              `yaml:"tax_basis" json:"tax_basis"`
	PriceDate   string              `yaml:"price_date" json:"price_date"`
	Geography   string              `yaml:"geography" json:"geography"`
	Quality     string              `yaml:"quality" json:"quality"`
	Inclusions  []string            `yaml:"inclusions" json:"inclusions"`
	Exclusions  []string            `yaml:"exclusions" json:"exclusions"`
	AppliesWhen any                 `yaml:"applies_when" json:"applies_when"`
	Status      string              `yaml:"status" json:"status"`
	Sources     []map[string]string `yaml:"sources" json:"sources"`
}

func (b *CostBenchmark) UnmarshalYAML(n *yaml.Node) error {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == "amount" && n.Content[i+1].Tag != "!!str" {
			return fmt.Errorf("benchmark amount must be a decimal string")
		}
	}
	type plain CostBenchmark
	return n.Decode((*plain)(b))
}
func (c *Catalog) loadCostBenchmarks(root string) error {
	path := filepath.Join(root, "costs", "benchmarks.yaml")
	if ok, e := exists(path); !ok {
		return e
	}
	var file struct {
		Version    int             `yaml:"version"`
		Benchmarks []CostBenchmark `yaml:"benchmarks"`
	}
	if e := c.unmarshal(path, &file); e != nil {
		return e
	}
	if file.Version != 1 {
		return fmt.Errorf("unsupported cost benchmark version")
	}
	c.costBenchmarks = map[string]CostBenchmark{}
	for _, b := range file.Benchmarks {
		if e := validateCostBenchmark(b); e != nil {
			return fmt.Errorf("%s: %w", path, e)
		}
		if e := c.checkWorksRefs(path, b.ID, b.AppliesWhen, nil); e != nil {
			return e
		}
		if e := c.validateCostPredicate(b.AppliesWhen, 0); e != nil {
			return fmt.Errorf("%s: %s: %w", path, b.ID, e)
		}
		if _, ok := c.costBenchmarks[b.ID]; ok {
			return fmt.Errorf("duplicate cost benchmark %s", b.ID)
		}
		c.costBenchmarks[b.ID] = b
	}
	return nil
}
func validateCostBenchmark(b CostBenchmark) error {
	if !regexp.MustCompile(`^bm\.[a-z0-9-]+$`).MatchString(b.ID) || b.Version < 1 || (b.Basis != "lump_sum" && b.Basis != "rate") || (b.Basis == "rate" && strings.TrimSpace(b.Unit) == "") || !regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,4})?$`).MatchString(b.Amount) || len(b.Amount) > 24 {
		return fmt.Errorf("invalid benchmark %s", b.ID)
	}
	r, ok := new(big.Rat).SetString(b.Amount)
	if !ok || r.Sign() < 0 {
		return fmt.Errorf("invalid benchmark amount")
	}
	if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(b.Currency) || (b.TaxBasis != "ex_tax" && b.TaxBasis != "inc_tax") || strings.TrimSpace(b.Geography) == "" || strings.TrimSpace(b.Quality) == "" || b.Inclusions == nil || b.Exclusions == nil || b.AppliesWhen == nil || len(b.Sources) == 0 || (b.Status != "draft" && b.Status != "reviewed") {
		return fmt.Errorf("incomplete benchmark %s", b.ID)
	}
	if _, e := time.Parse("2006-01-02", b.PriceDate); e != nil {
		return e
	}
	for _, source := range b.Sources {
		valid := false
		for _, key := range []string{"design", "clerk_file", "document", "dataset", "seed"} {
			if strings.TrimSpace(source[key]) != "" {
				valid = true
				if (key == "document" || key == "seed") && strings.TrimSpace(source["anchor"]) == "" {
					return fmt.Errorf("benchmark source needs anchor")
				}
				if key == "dataset" && strings.TrimSpace(source["row"]) == "" {
					return fmt.Errorf("benchmark dataset needs row")
				}
			}
		}
		if !valid {
			return fmt.Errorf("unrecognised benchmark source")
		}
	}
	return nil
}

// CostBenchmarks returns only owner-reviewed catalogue records. Applicability
// must also resolve true before code may suggest an amount.
func (c *Catalog) CostBenchmarks() []CostBenchmark {
	out := []CostBenchmark{}
	for _, b := range c.costBenchmarks {
		if b.Status == "reviewed" {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (c *Catalog) EligibleCostBenchmark(b CostBenchmark, env WorksEnv, currency, tax, date, geography, quality string) bool {
	return b.Status == "reviewed" && b.Currency == currency && b.TaxBasis == tax && b.PriceDate == date && b.Geography == geography && b.Quality == quality && c.Holds(b.AppliesWhen, env) == True
}

func (c *Catalog) validateCostPredicate(p any, depth int) error {
	if depth > 50 {
		return fmt.Errorf("predicate too deep")
	}
	m, ok := asMap(p)
	if !ok {
		return fmt.Errorf("predicate must be a mapping")
	}
	if det, ok := m["det"]; ok {
		if _, known := c.Determinant(fmt.Sprint(det)); !known {
			return fmt.Errorf("unknown determinant %v", det)
		}
		if len(m) != 2 {
			return fmt.Errorf("determinant needs exactly one operator")
		}
		for op, v := range m {
			switch op {
			case "det":
			case "any_of":
				if len(asList(v)) == 0 {
					return fmt.Errorf("any_of needs values")
				}
			case "eq", "is", "gt", "gte", "lt", "lte":
				if v == nil {
					return fmt.Errorf("empty comparison")
				}
			default:
				return fmt.Errorf("unknown predicate operator %s", op)
			}
		}
		return nil
	}
	for op, v := range m {
		switch op {
		case "all", "any":
			list, ok := v.([]any)
			if !ok {
				return fmt.Errorf("%s needs a list", op)
			}
			for _, child := range list {
				if e := c.validateCostPredicate(child, depth+1); e != nil {
					return e
				}
			}
		case "not":
			if e := c.validateCostPredicate(v, depth+1); e != nil {
				return e
			}
		case "system_present", "system_existing":
			if _, ok := c.System(fmt.Sprint(v)); !ok {
				return fmt.Errorf("unknown system %v", v)
			}
		case "works":
			if e := c.checkWorksRefs("benchmark", "applies_when", map[string]any{"works": v}, nil); e != nil {
				return e
			}
		default:
			return fmt.Errorf("unknown predicate operator %s", op)
		}
	}
	return nil
}
