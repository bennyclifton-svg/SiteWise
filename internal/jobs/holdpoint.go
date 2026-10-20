package jobs

import (
	"context"

	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
	"sitewise/internal/programread"
	"sitewise/internal/store"
)

// Reading C, hold-point release evidence (plan
// docs/plans/2026-10-18-program.md 10.4). The release noul joins a
// passage's existing evidence fan-out (https://docs.typesafe.ai/patterns/fan-out),
// never a call of its own, and only where all of these hold: the document
// is a certificate, report or correspondence; the project's program holds
// that hold point; and the passage is labelled with a system in the
// question's runs_on. Everywhere else the evidence call is exactly what it
// was. The answer is split off before the passage's own answers are
// applied, so their questions, version and thresholds are unchanged.

// releaseKinds are the document kinds that can record an inspection
// release.
var releaseKinds = map[string]bool{"certificate": true, "report": true, "correspondence": true}

// ReleaseTarget is a live program hold point whose interface has an
// authored release question. Evidence is the hold point's evidence text,
// the `hold_point` state field.
type ReleaseTarget struct {
	ProgramID, ActivityID, InterfaceID, Evidence string
}

// ReleaseTargets keeps the program hold points whose interface authors a
// release question, in the given order.
func ReleaseTargets(cat *knowledge.Catalog, holds []store.HoldPointTarget) []ReleaseTarget {
	if cat == nil || len(holds) == 0 {
		return nil
	}
	byID := map[string]*knowledge.HoldPoint{}
	for _, iface := range cat.Interfaces {
		if hp := iface.HoldPoint; hp != nil && hp.Release != nil {
			byID[iface.ID] = hp
		}
	}
	var out []ReleaseTarget
	for _, h := range holds {
		if hp := byID[h.InterfaceID]; hp != nil {
			out = append(out, ReleaseTarget{ProgramID: h.ProgramID, ActivityID: h.ActivityID, InterfaceID: h.InterfaceID, Evidence: hp.Evidence})
		}
	}
	return out
}

// releaseTargets reads the hold points this document's passages may
// evidence. Nil for any other document kind, and without a database read.
func (w *Worker) releaseTargets(ctx context.Context, job store.ClaimedJob, passages []storedPassage) ([]ReleaseTarget, error) {
	if w.Catalog == nil || len(passages) == 0 || !releaseKinds[passages[0].Kind] {
		return nil, nil
	}
	holds, err := w.Store.DocumentHoldPoints(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return nil, err
	}
	return ReleaseTargets(w.Catalog, holds), nil
}

// releaseFor is the first target, by interface id, whose release question
// runs on the passage's labels. One per passage: the state has one
// `hold_point` field, so two hold points cannot share one request.
func releaseFor(cat *knowledge.Catalog, passage Passage, targets []ReleaseTarget) (ReleaseTarget, knowledge.Question, bool) {
	if cat == nil || !releaseKinds[passage.Kind] {
		return ReleaseTarget{}, knowledge.Question{}, false
	}
	for _, t := range targets {
		if q, ok := cat.HoldPointRelease(t.InterfaceID, passage.Labels); ok {
			return t, q, true
		}
	}
	return ReleaseTarget{}, knowledge.Question{}, false
}

// WithRelease adds one hold point's release noul to the passage's
// evidence call: the same state plus `hold_point`, the same questions plus
// the release, and the question version extended with the release map's
// own version so the cache and the record name both. When no target
// applies, or the joined request would be too large, the call is returned
// unchanged and joined is false: the passage's own questions come first.
func WithRelease(cat *knowledge.Catalog, call jev.Call, passage Passage, targets []ReleaseTarget) (joined jev.Call, target ReleaseTarget, ok bool) {
	target, q, found := releaseFor(cat, passage, targets)
	base, isMap := call.State.(map[string]any)
	if !found || !isMap || len(call.Questions) == 0 {
		return call, ReleaseTarget{}, false
	}
	state := make(map[string]any, len(base)+1)
	for k, v := range base {
		state[k] = v
	}
	state[programread.HoldPointStateKey] = target.Evidence
	questions := make(map[string]jev.Question, len(call.Questions)+1)
	for k, v := range call.Questions {
		questions[k] = v
	}
	questions[q.ID] = programread.ReleaseQuestion(q)
	joined = call
	joined.State, joined.Questions = state, questions
	joined.QuestionVersion = call.QuestionVersion + "+" + programread.HoldPointReleaseVersion
	if _, err := jev.CheckRequestSize(joined); err != nil {
		return call, ReleaseTarget{}, false
	}
	return joined, target, true
}

// ReleaseCall is the release noul alone on the passage state plus
// `hold_point`, for calibration (cmd/program-eval): each question in a
// fan-out is answered on the same state, so the answer it calibrates is
// the one the joined call gets.
func ReleaseCall(cat *knowledge.Catalog, passage Passage, target ReleaseTarget) (jev.Call, bool) {
	q, ok := cat.HoldPointRelease(target.InterfaceID, passage.Labels)
	if !ok {
		return jev.Call{}, false
	}
	state := passageState(passage)
	state[programread.HoldPointStateKey] = target.Evidence
	return jev.Call{
		State:           state,
		Questions:       map[string]jev.Question{q.ID: programread.ReleaseQuestion(q)},
		Priority:        jev.PriorityBackground,
		QuestionVersion: programread.HoldPointReleaseVersion,
	}, true
}

// splitRelease takes the release answer out of a joined result, leaving
// the passage's own answers and unresolved keys exactly as one evidence
// call's would be.
func splitRelease(r jev.Result, id string) (jev.Result, jev.Answer, bool) {
	answer, answered := r.Answers[id]
	answers := make(map[string]jev.Answer, len(r.Answers))
	for k, v := range r.Answers {
		if k != id {
			answers[k] = v
		}
	}
	var unresolved []string
	for _, k := range r.Unresolved {
		if k != id {
			unresolved = append(unresolved, k)
		}
	}
	r.Answers, r.Unresolved = answers, unresolved
	return r, answer, answered && answer.Type == jev.TypeNoul
}

// releaseReading bands one release answer. A noul carries a probability,
// never a confidence (https://docs.typesafe.ai/confidence); without a
// calibrated cut-off nothing is proposed. A proposal never releases the
// hold point: only a user's click does (store.ReleaseHoldPoint).
func (w *Worker) releaseReading(t ReleaseTarget, passageID string, a jev.Answer) (store.HoldPointReading, bool) {
	band, apply := w.ProgramThresholds.BandRelease(a.Noul)
	if !apply {
		return store.HoldPointReading{}, false
	}
	return store.HoldPointReading{ProgramID: t.ProgramID, ActivityID: t.ActivityID, InterfaceID: t.InterfaceID,
		PassageID: passageID, Band: band, Probability: a.Noul}, true
}
