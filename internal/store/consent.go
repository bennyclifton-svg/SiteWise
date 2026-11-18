package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/delivery"
	"sitewise/internal/invoices"
	engine "sitewise/internal/program"
)

// Consent conditions in the register (docs/plans/2026-11-17-consent-conditions.md).
// The background read writes proposed rows (UpsertConditions); the PM's
// edits accept them; evidence is linked here. Nothing on these paths calls
// Jev or the model.

// addEvidence links a filed document of this project to a condition as its
// evidence. An open condition becomes evidence filed; nothing here ever
// makes one satisfied (owner decision 6).
func addEvidence(ctx context.Context, tx pgx.Tx, org, project, actor string, c *delivery.Content, document string) error {
	if c.Kind != "condition" {
		return fmt.Errorf("%w: only a consent condition takes evidence", ErrInvalidDelivery)
	}
	if err := projectDocument(ctx, tx, org, project, document); err != nil {
		return err
	}
	d := delivery.Condition(*c)
	if !slices.ContainsFunc(d.Evidence, func(e delivery.ConditionEvidence) bool { return e.DocumentID == document }) {
		d.Evidence = append(d.Evidence, delivery.ConditionEvidence{DocumentID: document, LinkedBy: actor,
			LinkedAt: time.Now().In(invoices.Sydney).Format("2006-01-02")})
	}
	if len(d.Evidence) > 50 {
		return fmt.Errorf("%w: a condition keeps at most 50 evidence documents", ErrInvalidDelivery)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	c.Details = raw
	if c.Status == "open" {
		c.Status = "evidence_filed"
	}
	return nil
}

// ConditionScope is the RFP line Add to RFP wrote.
type ConditionScope struct {
	ReportID, Stage, LineID, Text string
}

// ScopeStage is the RFP stage a condition's line goes to (owner decision
// 5): conditions met before the construction certificate or before work
// starts go to design finalisation, the rest to delivery. The PM can move
// the line on the RFP.
func ScopeStage(gate string) string {
	if gate == delivery.GateConstructionCertificate || gate == delivery.GateWorkStarts {
		return "design_finalisation"
	}
	return "delivery"
}

// ErrNoRFPDraft is a condition whose package has no RFP draft yet.
var ErrNoRFPDraft = errors.New("Create the RFP first")

// AddConditionScopeLine appends one line from a fixed template to the RFP
// draft of the condition's package, at custom:<stage>, through the RFP's own
// setting write, and records it on the condition. An issued RFP is not
// touched: the line goes to the draft only.
func (s *Store) AddConditionScopeLine(ctx context.Context, org, project, actor, item string) (ConditionScope, error) {
	var out ConditionScope
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT to_jsonb(i) FROM project_delivery_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL`, org, project, item).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	var it delivery.Item
	if err := json.Unmarshal(raw, &it); err != nil {
		return out, err
	}
	d := delivery.Condition(it.Content)
	switch {
	case it.Kind != "condition":
		return out, fmt.Errorf("%w: only a consent condition goes onto an RFP this way", ErrInvalidDelivery)
	case it.PackageID == "":
		return out, fmt.Errorf("%w: give the condition to a package first", ErrInvalidDelivery)
	case d.RFP != nil:
		return out, fmt.Errorf("%w: this condition is already on the RFP", ErrInvalidDelivery)
	}
	var report, draft string
	err = s.pool.QueryRow(ctx, `SELECT id::text, current_draft_version_id::text FROM reports WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND kind='rfp' AND current_draft_version_id IS NOT NULL ORDER BY id LIMIT 1`, org, project, it.PackageID).Scan(&report, &draft)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNoRFPDraft
	}
	if err != nil {
		return out, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	state, err := readDraftState(ctx, tx, org, draft)
	if err != nil {
		tx.Rollback(ctx)
		return out, err
	}
	settings, err := readReportSettings(ctx, tx, org, draft)
	tx.Rollback(ctx)
	if err != nil {
		return out, err
	}
	if state.Status != "draft" {
		return out, ErrNoRFPDraft
	}
	out.ReportID, out.Stage = report, ScopeStage(d.Stage)
	key := "custom:" + out.Stage
	var lines []map[string]string
	if set, ok := settings[key]; ok {
		_ = json.Unmarshal(set.Value, &lines)
	}
	out.LineID = "uc" + strings.ReplaceAll(newID(), "-", "")[:12]
	out.Text = conditionLine(d)
	lines = append(lines, map[string]string{"id": out.LineID, "text": out.Text, "value": "I"})
	value, err := json.Marshal(lines)
	if err != nil {
		return out, err
	}
	if _, _, err := s.SetReportSettingsInGroup(ctx, org, report, actor, []SettingChange{{Key: key, Value: value}}, state.Version); err != nil {
		return out, err
	}
	d.RFP = &delivery.ConditionRFP{ReportID: report, Stage: out.Stage, LineID: out.LineID}
	details, err := json.Marshal(d)
	if err != nil {
		return out, err
	}
	wtx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer wtx.Rollback(ctx)
	if err := lockProject(ctx, wtx, org, project); err != nil {
		return out, err
	}
	if _, err := wtx.Exec(ctx, `UPDATE project_delivery_items SET details=$4::jsonb,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, item, details); err != nil {
		return out, err
	}
	return out, s.finishDeliveryWrite(ctx, wtx, org, project)
}

// conditionLine is the RFP line's fixed template: the condition's number,
// its consent and its printed heading (or opening words), never reworded.
func conditionLine(d delivery.ConditionDetails) string {
	words := strings.TrimSpace(d.Heading)
	if words == "" {
		words = strings.Join(strings.Fields(d.Text), " ")
	}
	line := "Meet consent condition " + d.Number
	if d.ConsentRef != "" {
		line += " (" + d.ConsentRef + ")"
	}
	if words != "" {
		line += ": " + words
	}
	line = strings.NewReplacer("{", "(", "}", ")").Replace(line)
	if r := []rune(line); len(r) > 300 {
		line = string(r[:299]) + "…"
	}
	return line
}

// ConsentGateDates is each gate's date from the Program, computed in Go:
// the gate hold point's start, or before Adopt the start of the activity the
// gate stands for (the construction or occupation certificate). A gate the
// Program has no row for has no date; with no Program, none has.
func (s *Store) ConsentGateDates(ctx context.Context, org, project string) (map[string]string, error) {
	st, err := s.LoadProgram(ctx, org, project)
	if errors.Is(err, ErrNotFound) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	fallback := map[string]string{"fragment:fr.construction-certificate:": "before_cc", "fragment:fr.occupation-certificate:": "before_occupation"}
	for _, a := range st.Activities {
		key := string(a.SourceKey)
		t, ok := st.Computed.Activities.Get(a.ID)
		if !ok || t.Unknown || t.Start == "" {
			continue
		}
		if gate := engine.ConsentGateOf(key); gate != "" {
			out[gate], out["held:"+gate] = t.Start, t.Start
			continue
		}
		for prefix, gate := range fallback {
			if strings.HasPrefix(key, prefix) && (out[gate] == "" || t.Start < out[gate]) {
				if _, held := out["held:"+gate]; !held {
					out[gate] = t.Start
				}
			}
		}
	}
	for gate := range out {
		if strings.HasPrefix(gate, "held:") {
			delete(out, gate)
		}
	}
	return out, nil
}
