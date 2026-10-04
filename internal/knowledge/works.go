package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// The works layer (SCHEMA.md "Works layer"): what a kind of work does to a
// system and what that raises. Code evaluates it during the profile rebuild;
// nothing here calls Jev. Signals are loaded as data only: asking them is a
// separate decision (D-12 in docs/plans/2026-10-04-next-wave-implementation-plan.md),
// so they are not added to the evidence questions.

// Action is one of the eight things works do to a system.
type Action struct {
	ID        string `yaml:"id"`
	Describes string `yaml:"describes"`
	Excludes  string `yaml:"excludes"`
}

// Proposal is what a knowledge record proposes when it applies.
type Proposal struct {
	Kind  string `yaml:"kind"`
	Label string `yaml:"label"`
}

// InterfaceConsequence applies when the works touch one side of an interface
// of Type with one of Actions ("any" for every action), and the other side
// exists on the site and is not being replaced.
type InterfaceConsequence struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Touches string   `yaml:"touches"` // from | to | either
	Actions []string `yaml:"-"`
	AnyAct  bool     `yaml:"-"`
	Propose Proposal `yaml:"propose"`
	Status  string   `yaml:"status"`
}

// Consequence (cq.*) is a trigger interfaces cannot express, chiefly a
// regulatory one on existing buildings.
type Consequence struct {
	ID             string     `yaml:"id"`
	When           any        `yaml:"when"`
	Propose        []Proposal `yaml:"propose"`
	Signals        []string   `yaml:"signals"`
	GovernedBy     []string   `yaml:"governed_by"`
	ClauseVerified bool       `yaml:"clause_verified"`
	Severity       string     `yaml:"severity"`
	Status         string     `yaml:"status"`
}

// Unforeseen (uc.*) is a state of the site or building commonly discovered
// during the works, raised by the work items that make it likely.
type Unforeseen struct {
	ID         string            `yaml:"id"`
	Kind       string            `yaml:"kind"`
	Category   string            `yaml:"category"`
	AttachesTo map[string]string `yaml:"attaches_to"`
	When       any               `yaml:"when"`
	Signals    []string          `yaml:"signals"`
	DeRisk     *Proposal         `yaml:"de_risk"`
	Contract   string            `yaml:"contract"`
	Effect     []string          `yaml:"effect"`
	Severity   string            `yaml:"severity"`
	Status     string            `yaml:"status"`
}

// Signal is a shared noul question a record cites by id. It says what a
// passage states, never whether a condition exists.
type Signal struct {
	ID           string   `yaml:"id"`
	Type         string   `yaml:"type"`
	Instructions string   `yaml:"instructions"`
	Criteria     any      `yaml:"criteria"`
	RunsOn       []string `yaml:"runs_on"`
	Status       string   `yaml:"status"`
}

// Question is the signal as a Jev question (id "sig:<id>").
func (s Signal) Question() Question {
	return Question{ID: "sig:" + s.ID, Type: s.Type, Instructions: s.Instructions, Criteria: s.Criteria, RunsOn: s.RunsOn}
}

type worksData struct {
	actions          []Action
	answers          map[string]string
	workTypeDefaults map[string]string
	conditions       []string
	interfaceCQ      []InterfaceConsequence
	consequences     []Consequence
	unforeseen       []Unforeseen
	signals          map[string]Signal
}

// Actions returns the eight works actions in file order.
func (c *Catalog) Actions() []Action { return c.works.actions }

// DefaultAction is the action a coarse work item starts with for a work type
// (actions.yaml work_type_defaults), or "".
func (c *Catalog) DefaultAction(workType string) string { return c.works.workTypeDefaults[workType] }

// ExistingConditions are the condition ids a work item may record.
func (c *Catalog) ExistingConditions() []string { return c.works.conditions }

// InterfaceConsequences, Consequences and UnforeseenConditions return the
// loaded records; Signal looks one up by id.
func (c *Catalog) InterfaceConsequences() []InterfaceConsequence { return c.works.interfaceCQ }
func (c *Catalog) Consequences() []Consequence                   { return c.works.consequences }
func (c *Catalog) UnforeseenConditions() []Unforeseen            { return c.works.unforeseen }
func (c *Catalog) Signal(id string) (Signal, bool) {
	s, ok := c.works.signals[id]
	return s, ok
}

// loadWorks reads knowledge/works and each cluster's consequences, unforeseen
// conditions and signals. Every file is optional so fixtures stay small; a
// present file must reference only known actions, systems and signals.
func (c *Catalog) loadWorks(root string, clusters []string) error {
	c.works.signals = map[string]Signal{}
	dir := filepath.Join(root, "works")
	if err := c.loadActions(filepath.Join(dir, "actions.yaml")); err != nil {
		return err
	}
	if err := c.loadSignals(filepath.Join(dir, "signals.yaml")); err != nil {
		return err
	}
	for _, cl := range clusters {
		if err := c.loadSignals(filepath.Join(cl, "signals.yaml")); err != nil {
			return err
		}
	}
	if err := c.loadInterfaceConsequences(filepath.Join(dir, "interface_consequences.yaml")); err != nil {
		return err
	}
	for _, cl := range clusters {
		if err := c.loadConsequences(filepath.Join(cl, "consequences.yaml")); err != nil {
			return err
		}
		if err := c.loadUnforeseen(filepath.Join(cl, "unforeseen.yaml")); err != nil {
			return err
		}
	}
	sort.Slice(c.works.consequences, func(i, j int) bool { return c.works.consequences[i].ID < c.works.consequences[j].ID })
	sort.Slice(c.works.unforeseen, func(i, j int) bool { return c.works.unforeseen[i].ID < c.works.unforeseen[j].ID })
	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (c *Catalog) loadActions(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version            int               `yaml:"version"`
		Actions            []Action          `yaml:"actions"`
		Answers            map[string]string `yaml:"answers"`
		WorkTypeDefaults   map[string]string `yaml:"work_type_defaults"`
		ExistingConditions struct {
			Values []struct {
				ID string `yaml:"id"`
			} `yaml:"values"`
		} `yaml:"existing_conditions"`
	}
	if err := unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", path, file.Version)
	}
	c.works.actions = file.Actions
	c.works.answers = file.Answers
	c.works.workTypeDefaults = file.WorkTypeDefaults
	for _, v := range file.ExistingConditions.Values {
		c.works.conditions = append(c.works.conditions, v.ID)
	}
	for wt, a := range file.WorkTypeDefaults {
		if !c.isAction(a) {
			return fmt.Errorf("%s: work type %s defaults to unknown action %q", path, wt, a)
		}
	}
	return nil
}

func (c *Catalog) isAction(id string) bool {
	for _, a := range c.works.actions {
		if a.ID == id {
			return true
		}
	}
	return false
}

func (c *Catalog) loadSignals(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version int      `yaml:"version"`
		Signals []Signal `yaml:"signals"`
	}
	if err := unmarshal(path, &file); err != nil {
		return err
	}
	for _, s := range file.Signals {
		if s.ID == "" || s.Type != "noul" {
			return fmt.Errorf("%s: signal %q must be a noul with an id", path, s.ID)
		}
		if _, dup := c.works.signals[s.ID]; dup {
			return fmt.Errorf("%s: duplicate signal %s", path, s.ID)
		}
		for _, sys := range s.RunsOn {
			if _, ok := c.systems[sys]; !ok {
				return fmt.Errorf("%s: signal %s runs on unknown system %s", path, s.ID, sys)
			}
		}
		c.works.signals[s.ID] = s
	}
	return nil
}

func (c *Catalog) loadInterfaceConsequences(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version int `yaml:"version"`
		Records []struct {
			InterfaceConsequence `yaml:",inline"`
			Actions              any `yaml:"actions"`
		} `yaml:"interface_consequences"`
	}
	if err := unmarshal(path, &file); err != nil {
		return err
	}
	for _, r := range file.Records {
		ic := r.InterfaceConsequence
		switch r.Actions.(type) {
		case string:
			if r.Actions != "any" {
				return fmt.Errorf("%s: %s actions must be a list or any", path, ic.ID)
			}
			ic.AnyAct = true
		default:
			for _, a := range asList(r.Actions) {
				id := fmt.Sprint(a)
				if !c.isAction(id) {
					return fmt.Errorf("%s: %s names unknown action %q", path, ic.ID, id)
				}
				ic.Actions = append(ic.Actions, id)
			}
		}
		switch ic.Touches {
		case "from", "to", "either":
		default:
			return fmt.Errorf("%s: %s touches must be from, to or either", path, ic.ID)
		}
		c.works.interfaceCQ = append(c.works.interfaceCQ, ic)
	}
	return nil
}

func (c *Catalog) loadConsequences(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version      int           `yaml:"version"`
		Consequences []Consequence `yaml:"consequences"`
	}
	if err := unmarshal(path, &file); err != nil {
		return err
	}
	for _, r := range file.Consequences {
		if err := c.checkWorksRefs(path, r.ID, r.When, r.Signals); err != nil {
			return err
		}
		c.works.consequences = append(c.works.consequences, r)
	}
	return nil
}

func (c *Catalog) loadUnforeseen(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version    int          `yaml:"version"`
		Unforeseen []Unforeseen `yaml:"unforeseen"`
	}
	if err := unmarshal(path, &file); err != nil {
		return err
	}
	for _, r := range file.Unforeseen {
		if err := c.checkWorksRefs(path, r.ID, r.When, r.Signals); err != nil {
			return err
		}
		if sys, ok := r.AttachesTo["system"]; ok {
			if _, known := c.systems[sys]; !known {
				return fmt.Errorf("%s: %s attaches to unknown system %s", path, r.ID, sys)
			}
		}
		c.works.unforeseen = append(c.works.unforeseen, r)
	}
	return nil
}

// checkWorksRefs refuses a record whose `works` predicates name an unknown
// action or system, or which cites an unknown signal. The checker catches
// these first; this keeps a hand-edited file from loading half-understood.
func (c *Catalog) checkWorksRefs(path, id string, when any, signals []string) error {
	for _, s := range signals {
		if _, ok := c.works.signals[s]; !ok {
			return fmt.Errorf("%s: %s cites unknown signal %s", path, id, s)
		}
	}
	var bad error
	walkPredicate(when, func(m map[string]any) {
		w, ok := asMap(m["works"])
		if !ok || bad != nil {
			return
		}
		for _, a := range asList(w["action"]) {
			if !c.isAction(fmt.Sprint(a)) {
				bad = fmt.Errorf("%s: %s works names unknown action %v", path, id, a)
			}
		}
		for _, s := range asList(w["system"]) {
			if _, ok := c.systems[fmt.Sprint(s)]; !ok {
				bad = fmt.Errorf("%s: %s works names unknown system %v", path, id, s)
			}
		}
	})
	return bad
}

func walkPredicate(p any, visit func(map[string]any)) {
	if m, ok := asMap(p); ok {
		visit(m)
		for _, sub := range m {
			walkPredicate(sub, visit)
		}
		return
	}
	for _, sub := range asList(p) {
		walkPredicate(sub, visit)
	}
}
