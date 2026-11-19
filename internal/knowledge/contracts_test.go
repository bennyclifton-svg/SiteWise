package knowledge

import (
	"strings"
	"testing"
)

const sopFixture = `version: 1
status: draft
acts:
  - id: sop.nsw
    state: NSW
    act: An Act
    business_day_excludes: {weekdays: [sat, sun], public_holidays: true, spans: [{from: "12-27", to: "12-31"}]}
    schedule: {section: s 14(4), business_days: "10", rule: r}
    payment: {section: s 11(1A)(a), business_days: "15", rule: r}
    other_payments: []
    verified: false
    primary_source: ""
    secondary_source: guidance
    checked_on: ""
    status: draft
    notes: n
`

func TestSOPActsParse(t *testing.T) {
	acts, err := parseSOPActs([]byte(sopFixture))
	if err != nil {
		t.Fatal(err)
	}
	a := acts[0]
	if a.State != "NSW" || a.Verified || a.Schedule.BusinessDays != 10 || a.Payment.BusinessDays != 15 || a.Payment.Section != "s 11(1A)(a)" || len(a.Excludes.Spans) != 1 {
		t.Fatalf("%+v", a)
	}
}

func TestSOPActsRefuse(t *testing.T) {
	for name, edit := range map[string][2]string{
		"verified without a source": {"verified: false", "verified: true"},
		"unquoted count":            {`business_days: "10"`, `business_days: 10`},
		"unknown state":             {"id: sop.nsw\n    state: NSW", "id: sop.xx\n    state: XX"},
		"id not the state":          {"id: sop.nsw", "id: sop.qld"},
		"bad span":                  {`to: "12-31"`, `to: "12-20"`},
		"no section":                {"section: s 14(4), ", ""},
		"unknown field":             {"notes: n", "notes: n\n    extra: 1"},
	} {
		body := strings.Replace(sopFixture, edit[0], edit[1], 1)
		if body == sopFixture {
			t.Fatalf("%s: the fixture edit did not apply", name)
		}
		if _, err := parseSOPActs([]byte(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	verified := strings.Replace(strings.Replace(sopFixture, "verified: false", "verified: true", 1), `primary_source: ""`, "primary_source: https://legislation.example/act", 1)
	verified = strings.Replace(verified, `checked_on: ""`, `checked_on: "2026-11-19"`, 1)
	if _, err := parseSOPActs([]byte(verified)); err != nil {
		t.Errorf("a verified row with its source: %v", err)
	}
}

// The real file loads, and nothing in it is verified: no one on the project
// has read these Acts (owner decision 1 is open).
func TestSOPFileLoadsUnverified(t *testing.T) {
	c, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	nsw, ok := c.SOPAct("NSW")
	if !ok || nsw.Schedule == nil || nsw.Payment == nil || nsw.Schedule.BusinessDays != 10 || nsw.Payment.BusinessDays != 15 {
		t.Fatalf("NSW row: %+v", nsw)
	}
	for _, a := range c.SOPActs() {
		if a.Verified || a.Status != "draft" {
			t.Errorf("%s must stay draft and unverified until the owner reads the Act", a.ID)
		}
	}
	if _, ok := c.SOPAct("WA"); ok {
		t.Error("WA has no recorded periods")
	}
}
