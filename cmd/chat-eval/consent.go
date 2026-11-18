package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

// The house fixture's consent conditions (consent plan, "From chat"). The
// notice is read by the real route and the real background job; Jev is a
// stand-in that answers from this table, so no live call is made. These
// are synthetic picks for a synthetic notice, not an answer key.
type conditionPick struct {
	Timing, Party, System string
	Evidence              float64
	PartyP                float64 // below 0.5 the party stays blank
}

var houseConditions = map[string]conditionPick{
	"3":  {Timing: "before_cc", Party: "applicant", System: "none", Evidence: 0.9, PartyP: 0.8},
	"7":  {Timing: "before_cc", Party: "consultant.structural", System: "structure", Evidence: 0.9, PartyP: 0.8},
	"9":  {Timing: "before_cc", Party: "consultant.civil", System: "site", Evidence: 0.9, PartyP: 0.8},
	"14": {Timing: "before_work", Party: "consultant.arborist", System: "site", Evidence: 0.85, PartyP: 0.8},
	"22": {Timing: "during_works", Party: "trade.main_works", System: "none", Evidence: 0.1, PartyP: 0.8},
	"25": {Timing: "before_cc", Party: "consultant.surveyor", System: "none", Evidence: 0.9, PartyP: 0.3},
}

type fixtureAsker struct{ picks map[string]conditionPick }

func (a fixtureAsker) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	state, _ := call.State.(map[string]any)
	cond, _ := state["condition"].(map[string]any)
	number, _ := cond["number"].(string)
	p, ok := a.picks[number]
	if !ok {
		return jev.Result{Answers: map[string]jev.Answer{}}, nil
	}
	conf := func(v float64) *float64 { return &v }
	out := jev.Result{Answers: map[string]jev.Answer{
		"party":    {Type: jev.TypeChoice, Choice: p.Party, Confidence: conf(p.PartyP)},
		"system":   {Type: jev.TypeChoice, Choice: p.System, Confidence: conf(0.8)},
		"evidence": {Type: jev.TypeNoul, Noul: p.Evidence},
	}}
	if _, asked := call.Questions["timing"]; asked {
		out.Answers["timing"] = jev.Answer{Type: jev.TypeChoice, Choice: p.Timing, Confidence: conf(0.8)}
	}
	return out, nil
}

// seedConditions reads each filed notice through POST /delivery/imports and
// runs the consent job with the stand-in.
func (s *session) seedConditions(ctx context.Context, docs map[string]string, picks map[string]conditionPick) error {
	if len(picks) == 0 {
		return nil
	}
	base := "/api/projects/" + s.project + "/delivery/imports"
	for _, id := range docs {
		status, _, err := s.do(http.MethodPost, base, map[string]string{"document_id": id})
		if err != nil {
			return err
		}
		if status != http.StatusAccepted && status != http.StatusUnprocessableEntity {
			return errors.New("delivery import: unexpected status " + http.StatusText(status))
		}
	}
	w := &jobs.Worker{Store: s.l.st, Ask: fixtureAsker{picks}}
	for {
		job, err := s.l.st.ClaimJobSince(ctx, s.org, time.Minute, []string{store.JobKindConsent}, time.Time{})
		if errors.Is(err, store.ErrIdle) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := w.Perform(ctx, job); err != nil {
			return err
		}
		if err := s.l.st.CompleteJob(ctx, job.OrgID, job.ID, job.Token, store.EventWrite{Kind: "job", DocumentID: job.DocumentID, Payload: `{"kind":"consent","status":"done"}`}); err != nil {
			return err
		}
	}
}
