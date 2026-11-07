package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"sitewise/internal/procurement"
)

// groupAward times a combined engagement's price table and award
// (docs/plans/2026-11-07-combined-engagements-slice3.md): a five-member
// group with five bidders, each pricing every discipline's stages. The
// table counts towards price_table_group (the price_table budget); award
// and undo towards award_write_group (the award_write budget).
func (a *apiClient) groupAward(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Combined engagement award benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project
	ids := []string{}
	for _, d := range []string{"consultant.structural", "consultant.civil", "consultant.mechanical", "consultant.electrical", "consultant.hydraulic"} {
		raw, err := a.call(ctx, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"`+d+`"}`), http.StatusCreated)
		if err != nil {
			return err
		}
		var p procurement.Package
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		ids = append(ids, p.ID)
	}
	if _, err := a.call(ctx, http.MethodPost, base+"/cost-plan/prepare", []byte(`{}`), http.StatusOK); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{"package_ids": ids})
	raw, err := a.call(ctx, http.MethodPost, base+"/packages/combine", body, http.StatusCreated)
	if err != nil {
		return err
	}
	var group procurement.Package
	if err := json.Unmarshal(raw, &group); err != nil {
		return err
	}
	prices := base + "/procurement/packages/" + group.ID + "/prices"
	raw, err = a.call(ctx, http.MethodGet, prices, nil, http.StatusOK)
	if err != nil {
		return err
	}
	var t groupTableJSON
	if err := json.Unmarshal(raw, &t); err != nil {
		return err
	}
	var keys []string
	for _, it := range t.Items {
		if it.Base && !strings.HasPrefix(it.Key, "combined:") && !strings.HasPrefix(it.Key, "criterion:") {
			keys = append(keys, it.Key)
		}
	}
	if len(keys) == 0 {
		return errors.New("group award benchmark: the group has no fee rows")
	}
	// Firms are added and their fees typed, as a PM enters a bid on paper:
	// the reading has its own budget (proposal_read); this times the table
	// and the award at the group's full width.
	var subs []string
	for firm := 1; firm <= 5; firm++ {
		body, _ := json.Marshal(map[string]any{"firm": map[string]string{"name": fmt.Sprintf("Group Firm %02d Pty Ltd", firm)}})
		raw, err := a.call(ctx, http.MethodPost, base+"/procurement/packages/"+group.ID+"/firms", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var added groupTableJSON
		if err := json.Unmarshal(raw, &added); err != nil || len(added.Columns) != firm {
			return fmt.Errorf("group award benchmark: firm %d not added: %v", firm, err)
		}
		sub := added.Columns[firm-1].Submission.ID
		for j, key := range keys {
			amount := fmt.Sprintf("%d.%02d", 1000+10*j+firm, firm)
			cell := []byte(`{"state":"priced","amount":"` + amount + `"}`)
			if _, err := a.call(ctx, http.MethodPut, base+"/submissions/"+sub+"/cells/"+url.PathEscape(key), cell, http.StatusOK); err != nil {
				return err
			}
		}
		subs = append(subs, sub)
	}
	for i := 0; i < n; i++ {
		raw, err := a.timed(ctx, "price_table_group", http.MethodGet, prices, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &t); err != nil {
			return err
		}
		sub := subs[i%len(subs)]
		award := t.awardFor(sub)
		if len(award) == 0 {
			return errors.New("group award benchmark: no priced cells were read")
		}
		body, _ := json.Marshal(map[string]any{"submission_id": sub, "lines": award})
		if _, err := a.timed(ctx, "award_write_group", http.MethodPost, base+"/procurement/packages/"+group.ID+"/award", body, http.StatusCreated); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "award_write_group", http.MethodDelete, base+"/procurement/packages/"+group.ID+"/award", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}

type groupTableJSON struct {
	Items []struct {
		Key   string `json:"key"`
		Label string `json:"label"`
		Base  bool   `json:"base"`
	} `json:"items"`
	Columns []struct {
		Submission struct {
			ID string `json:"id"`
		} `json:"submission"`
		Cells map[string]struct {
			State string `json:"state"`
		} `json:"cells"`
	} `json:"columns"`
}

// awardFor is one award line per priced base cell of a submission, as the
// award dialog proposes by default.
func (t groupTableJSON) awardFor(sub string) []map[string]any {
	var out []map[string]any
	for _, c := range t.Columns {
		if c.Submission.ID != sub {
			continue
		}
		for _, it := range t.Items {
			if it.Base && c.Cells[it.Key].State == "priced" {
				out = append(out, map[string]any{"label": it.Label, "item_keys": []string{it.Key}})
			}
		}
	}
	return out
}
