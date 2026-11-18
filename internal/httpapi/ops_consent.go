package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"sitewise/internal/delivery"
	"sitewise/internal/store"
)

// Consent conditions from chat and the screen
// (docs/plans/2026-11-17-consent-conditions.md, "From chat" and Task 6).
// Satisfied and accepting evidence are vouches: they wait for the PM's
// click. Reading a notice spends Jev credits: chat prepares it as a card.
// Adding a condition to an RFP writes the RFP's settings, which the change
// log does not capture yet, so from chat it waits for the click too.

// conditionRef is "condition 46", "cond. A1" or "C46".
var conditionRef = regexp.MustCompile(`(?i)^\s*(?:condition|cond\.?|c)\s*#?\s*([A-Z]{0,2}\d{1,3})\s*$`)

// conditionItem finds one live condition by its id, short id or printed
// number. A number two consents share is refused rather than guessed.
func conditionItem(ctx context.Context, d Deps, who opActor, project, ref string) (delivery.Item, error) {
	items, err := d.Store.ReadDelivery(ctx, who.OrgID, project)
	if err != nil {
		return delivery.Item{}, err
	}
	if m := conditionRef.FindStringSubmatch(ref); m != nil {
		var found []delivery.Item
		for _, it := range items {
			if it.Kind == "condition" && strings.EqualFold(delivery.Condition(it.Content).Number, m[1]) {
				found = append(found, it)
			}
		}
		switch len(found) {
		case 1:
			return found[0], nil
		case 0:
			return delivery.Item{}, refuse("item", "There is no condition %s on the register.", m[1])
		}
		return delivery.Item{}, refuse("item", "Two consents have a condition %s; name the row by its id.", m[1])
	}
	it, err := byRef(items, func(it delivery.Item) string { return it.ID }, ref, "register row")
	if err == nil && it.Kind != "condition" {
		return it, refuse("item", "%s is not a consent condition.", it.Title)
	}
	return it, err
}

func conditionName(it delivery.Item) string {
	return it.Title
}

var consentForwards = []forward{
	{Name: "delivery.satisfy", Click: true, About: "mark a consent condition satisfied: you are saying the certifier or council accepted it.",
		Fields: `item (a condition's row id, or its number such as condition 46)`, Route: "PATCH /projects/{id}/delivery/{item}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			it, err := satisfiable(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			return sent{Path: map[string]string{"item": it.ID}, Body: map[string]any{"version": it.Version, "status": "satisfied"}}, nil
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
			it, err := satisfiable(ctx, d, who, project, f)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Mark %s satisfied? You are saying the certifier or council accepted it.", conditionName(it)), nil
		},
		Done: func(map[string]any) string { return "Marked the condition satisfied." }},
	{Name: "delivery.accept_evidence", Click: true, About: "accept possible evidence: you vouch that the document is the evidence the consent condition asks for. The condition becomes evidence filed, not satisfied.",
		Fields: `item (the condition's id or number); reading (the possible evidence's id from read delivery)`, Route: "PATCH /projects/{id}/delivery/{item}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			it, rd, err := evidenceReading(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			return sent{Path: map[string]string{"item": it.ID}, Body: map[string]any{"version": it.Version, "accept_reading": rd.ID}}, nil
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
			it, rd, err := evidenceReading(ctx, d, who, project, f)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Accept %s as evidence for %s? It becomes evidence filed; you mark it satisfied when the council or certifier accepts it.", rd.Document, conditionName(it)), nil
		},
		Done: func(map[string]any) string { return "Linked the evidence." }},
}

func satisfiable(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (delivery.Item, error) {
	if err := need(f, "item"); err != nil {
		return delivery.Item{}, err
	}
	it, err := conditionItem(ctx, d, who, project, fieldText(f, "item"))
	if err != nil {
		return it, err
	}
	if it.Status == "satisfied" {
		return it, refuse("item", "%s is already satisfied.", conditionName(it))
	}
	return it, nil
}

func evidenceReading(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (delivery.Item, delivery.Reading, error) {
	if err := need(f, "item", "reading"); err != nil {
		return delivery.Item{}, delivery.Reading{}, err
	}
	it, err := conditionItem(ctx, d, who, project, fieldText(f, "item"))
	if err != nil {
		return it, delivery.Reading{}, err
	}
	rd, err := byRef(it.Readings, func(r delivery.Reading) string { return r.ID }, fieldText(f, "reading"), "possible evidence on that condition")
	return it, rd, err
}

// resolveReadNotice is delivery.read_notice's resolver (ops_reads.go): the
// PM's click queues the read.
func resolveReadNotice(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
	v, name, err := projectDocument(ctx, d, who, project, fieldText(f, "document"))
	if err != nil {
		return sent{}, "", "", err
	}
	pages, err := d.Store.ConsentDocument(ctx, who.OrgID, project, v.ID)
	if err != nil {
		return sent{}, "", "", err
	}
	if msg := consentRefusal(pages); msg != "" {
		return sent{}, "", "", refuse("document", "%s: %s.", name, strings.TrimSuffix(msg, "."))
	}
	reads, err := d.Store.ConsentReads(ctx, who.OrgID, project)
	if err != nil {
		return sent{}, "", "", err
	}
	for _, r := range reads {
		if r.DocumentID == v.ID && (r.Status == "queued" || r.Status == "reading") {
			return sent{}, "", "", refuse("document", "The conditions of %s are already being read; nothing more is queued.", name)
		}
	}
	return sent{Body: map[string]any{"document_id": v.ID}},
		fmt.Sprintf("Read the consent conditions of %s into the register? Each becomes a row to check. %s", name, costWords),
		fmt.Sprintf("Queued: the conditions of %s are being read.", name), nil
}

// consentOperations is Add to RFP, an operation of its own: it writes the
// RFP draft's setting and the condition in Go, so one operation stands for
// both writes.
func consentOperations() []operation {
	return []operation{{
		Name:   "reports.add_scope_line",
		About:  "Add a consent condition to its package's RFP draft as one scope line (fixed wording: the condition's number, consent and printed heading), at design finalisation for conditions before the construction certificate or work starting, else at delivery.",
		Fields: `item (a condition's row id, or its number such as condition 46)`,
		NoUndo: true,
		// The RFP's settings are not captured by the change log, so from chat
		// this waits for the PM's click; the screen's Add to RFP applies it.
		Hold: func(ctx context.Context, _ Deps, _ opActor, _ string, _ json.RawMessage) (string, error) {
			if r, ok := ctx.Value(opRequestKey{}).(*http.Request); ok && strings.HasSuffix(r.URL.Path, "/ops") {
				return "", nil
			}
			return "Adding a line to an RFP can't be undone from chat, so it waits for your click.", nil
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
			it, err := scopeCondition(ctx, d, who, project, raw)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Add %s to its package's RFP draft as a scope line at %s? This can't be undone from chat.", conditionName(it), strings.ReplaceAll(store.ScopeStage(delivery.Condition(it.Content).Stage), "_", " ")), nil
		},
		Run: func(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
			it, err := scopeCondition(ctx, d, who, project, raw)
			if err != nil {
				return "", err
			}
			line, err := d.Store.AddConditionScopeLine(ctx, who.OrgID, project, who.UserID, it.ID)
			switch {
			case errors.Is(err, store.ErrNoRFPDraft):
				return "", refuse("item", "Create the RFP first.")
			case errors.Is(err, store.ErrInvalidDelivery):
				return "", refuse("item", "%s", strings.TrimPrefix(err.Error(), store.ErrInvalidDelivery.Error()+": "))
			case err != nil:
				return "", err
			}
			if d.Broker != nil {
				d.Broker.Wake(who.OrgID)
			}
			return fmt.Sprintf("Added %q to the RFP draft at %s.", line.Text, strings.ReplaceAll(line.Stage, "_", " ")), nil
		},
	}}
}

func scopeCondition(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (delivery.Item, error) {
	var f struct {
		Item string `json:"item"`
	}
	if err := decodeFields(raw, &f); err != nil {
		return delivery.Item{}, err
	}
	if f.Item == "" {
		return delivery.Item{}, refuse("item", "item is needed.")
	}
	it, err := conditionItem(ctx, d, who, project, f.Item)
	if err != nil {
		return it, err
	}
	if it.PackageID == "" {
		return it, refuse("item", "Give %s to a package first: Add to RFP puts it on that package's RFP.", conditionName(it))
	}
	if delivery.Condition(it.Content).RFP != nil {
		return it, refuse("item", "%s is already on the RFP.", conditionName(it))
	}
	return it, nil
}

// conditionFields are a condition's editable fields, lifted into details
// over what the row has: the gate, who acts (a discipline) and whether
// evidence is needed. Its number and words are never sent.
func liftConditionFields(f map[string]any, current json.RawMessage) error {
	details := map[string]any{}
	_ = json.Unmarshal(current, &details)
	lifted := false
	if g, ok := f["gate"]; ok {
		details["stage"] = g
		delete(f, "gate")
		lifted = true
	}
	if v, ok := f["discipline"]; ok {
		details["discipline"] = v
		delete(f, "discipline")
		lifted = true
	}
	if v, ok := f["evidence_needed"]; ok {
		details["evidence_needed"] = v
		delete(f, "evidence_needed")
		lifted = true
	}
	if given, ok := f["details"].(map[string]any); ok {
		for k, v := range given {
			details[k] = v
		}
		lifted = true
	}
	if lifted {
		f["details"] = details
	}
	return nil
}
