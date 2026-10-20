package main

// With -family hold_point_release it calibrates Reading C (plan 10.4): each
// labelled passage is asked the interface's release noul on the passage
// state plus `hold_point`, as the evidence fan-out asks it, and -fit
// replaces that family's cut-off only, versioned "hold-point-release-1".
// A noul has a probability and no confidence
// (https://docs.typesafe.ai/confidence), so the cut-off has no option
// shape: a proposal is written when the probability of true is at or above
// it, and an answer is right when the owner says the passage does record
// the release. Release questions are read from -knowledge. Gold set format
// (data/eval/program/private/hold-point-key.json; private because
// certificates name real projects):
//
//	{
//	  "question_version": "hold-point-release-1",
//	  "cases": [{
//	    "id": "newham-cert-01",
//	    "interface": "if.envelope-frame-before-roof",
//	    "passage": {"kind": "certificate", "discipline": "structural",
//	                "title": "Frame inspection", "section": "", "text": "…",
//	                "page": 2, "labels": ["structure"]},
//	    "expected": true   // the passage records the release
//	  }]
//	}
//
// No gold set exists yet, and no release question is authored (WP-P15):
// until both exist the family has no cut-off and nothing is proposed
// ("hold_point_release: not calibrated").

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sitewise/internal/eval"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
	"sitewise/internal/programread"
)

type releasePassage struct {
	Kind       string   `json:"kind"`
	Discipline string   `json:"discipline"`
	Title      string   `json:"title"`
	Section    string   `json:"section"`
	Context    string   `json:"context"`
	Text       string   `json:"text"`
	Page       int      `json:"page"`
	Labels     []string `json:"labels"`
}

type releaseCase struct {
	ID        string         `json:"id"`
	Interface string         `json:"interface"`
	Passage   releasePassage `json:"passage"`
	Expected  bool           `json:"expected"`
}

// releaseCases binds each gold case to its run; the catalogue supplies the
// interface's release question and evidence text.
func releaseCases(ctx context.Context, cases json.RawMessage, knowledgeDir string) ([]func(programread.Asker) ([]eval.Answer, error), []string, error) {
	var list []releaseCase
	if err := json.Unmarshal(cases, &list); err != nil {
		return nil, nil, err
	}
	cat, err := knowledge.Load(knowledgeDir)
	if err != nil {
		return nil, nil, err
	}
	evidence := map[string]string{}
	for _, iface := range cat.Interfaces {
		if iface.HoldPoint != nil && iface.HoldPoint.Release != nil {
			evidence[iface.ID] = iface.HoldPoint.Evidence
		}
	}
	var runs []func(programread.Asker) ([]eval.Answer, error)
	var ids []string
	for _, c := range list {
		text, ok := evidence[c.Interface]
		if !ok {
			return nil, nil, fmt.Errorf("%s: interface %q has no release question in %s", c.ID, c.Interface, knowledgeDir)
		}
		c := c
		ids = append(ids, c.ID)
		runs = append(runs, func(ask programread.Asker) ([]eval.Answer, error) {
			return evaluateReleaseCase(ctx, ask, cat, c, text)
		})
	}
	return runs, ids, nil
}

func evaluateReleaseCase(ctx context.Context, ask programread.Asker, cat *knowledge.Catalog, c releaseCase, evidence string) ([]eval.Answer, error) {
	p := c.Passage
	passage := jobs.Passage{Kind: p.Kind, Discipline: p.Discipline, Title: p.Title, Section: p.Section, Context: p.Context, Text: p.Text, Page: p.Page, Labels: p.Labels}
	call, ok := jobs.ReleaseCall(cat, passage, jobs.ReleaseTarget{InterfaceID: c.Interface, Evidence: evidence})
	if !ok {
		return nil, nil // the passage's labels are outside runs_on: never asked
	}
	res, err := ask.Ask(ctx, call)
	if err != nil {
		return nil, err
	}
	a, answered := res.Answers[knowledge.ReleaseQuestionID(c.Interface)]
	if !answered {
		return nil, nil
	}
	split := eval.SplitOf(c.ID, programread.HoldPointReleaseVersion, 60)
	return []eval.Answer{{Case: split + "/" + c.ID, Field: programread.HoldPointFamily, Options: 2, Confidence: a.Noul, Correct: c.Expected}}, nil
}

// writeReleaseThresholds fits the release cut-off on the calibration split
// and writes it with its n and no option shape, keeping every other
// family's cut-offs in the file.
func writeReleaseThresholds(dir string, answers []eval.Answer) error {
	var calib []eval.Answer
	for _, a := range answers {
		if a.Field == programread.HoldPointFamily && strings.HasPrefix(a.Case, eval.SplitCalibration+"/") {
			calib = append(calib, a)
		}
	}
	th, err := programread.LoadThresholds(dir)
	if err != nil {
		return err
	}
	if th.Questions == nil {
		th.Questions = map[string][]programread.Threshold{}
	}
	if th.Versions == nil {
		th.Versions = map[string]string{}
	}
	th.Versions[programread.HoldPointFamily] = programread.HoldPointReleaseVersion
	t := programread.Threshold{N: len(calib), Reason: "no calibration answers"}
	if len(calib) > 0 {
		strict := policy
		strict.MaxErrors = 0
		key := programread.HoldPointFamily
		fits := eval.Fit(calib, map[string]eval.FitPolicy{key: policy})
		greens := eval.Fit(calib, map[string]eval.FitPolicy{key: strict})
		if len(fits) > 0 {
			t.Amber, t.N, t.Reason = fits[0].Amber, fits[0].N, fits[0].Reason
		}
		if len(greens) > 0 && greens[0].Green != nil {
			t.Green = greens[0].Green
		}
		if t.Green != nil && (t.Amber == nil || *t.Amber > *t.Green) {
			t.Amber = t.Green
		}
	}
	th.Questions[programread.HoldPointFamily] = []programread.Threshold{t}
	th.Calibration = fmt.Sprintf("cmd/program-eval -fit -family %s, %d calibration answers, %s", programread.HoldPointFamily, len(calib), time.Now().UTC().Format("2006-01-02"))
	th.Policy, _ = json.Marshal(policy)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(th, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "thresholds.json"), append(out, '\n'), 0o644)
}
