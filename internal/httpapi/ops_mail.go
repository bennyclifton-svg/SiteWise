package httpapi

import (
	"context"
	"errors"
	"fmt"

	"sitewise/internal/store"
)

// Mail from chat (docs/plans/2026-11-17-correspondence-transmittals.md,
// A4 and "From chat"). Accepting held mail starts Jev filing, so it is a
// prepared read: always a card for the PM's click (plan 0, decision 1).
// Discarding held mail cannot be put back, so it waits for a click too.
// Rotating the address is never chat's (ops_routes.go). No card or summary
// carries a message's subject or body: those are outside text.
var mailForwards = []forward{
	{
		Name:   "mail.link",
		About:  "set the package a message belongs to (or clear it); it applies at once and Undo puts it back.",
		Fields: `message (id from read messages), package (id or title from read procurement; "" clears it)`,
		Route:  "PATCH /projects/{id}/mail/{message}",
		Undo:   true,
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			m, err := recordMessage(ctx, d, who, project, fieldText(f, "message"))
			if err != nil {
				return sent{}, err
			}
			pkg, title := "", ""
			if fieldText(f, "package") != "" {
				p, err := pkgOf(ctx, d, who, project, f)
				if err != nil {
					return sent{}, err
				}
				pkg, title = p.ID, p.Title
			}
			if title == "" {
				f["_done"] = fmt.Sprintf("Cleared the package of the message %s %s.", direction(m), m.With)
			} else {
				f["_done"] = fmt.Sprintf("Linked the message %s %s to %s.", direction(m), m.With, title)
			}
			return sent{Path: map[string]string{"message": m.ID}, Body: map[string]any{"version": m.Version, "package_id": pkg}}, nil
		},
		Done: func(f map[string]any) string { return fieldText(f, "_done") },
	},
	{
		Name:   "mail.dismiss_pick",
		About:  "say a message is not the notice, claim or RFI Jev's chip suggests (\"Not one\"); the chip goes, and Undo brings it back.",
		Fields: `message (id from read messages)`,
		Route:  "PATCH /projects/{id}/mail/{message}",
		Undo:   true,
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			m, err := recordMessage(ctx, d, who, project, fieldText(f, "message"))
			if err != nil {
				return sent{}, err
			}
			if m.Notice == "" {
				return sent{}, refuse("message", "The message %s %s has no suggestion to dismiss.", direction(m), m.With)
			}
			f["_done"] = fmt.Sprintf("Dismissed the suggestion on the message %s %s.", direction(m), m.With)
			return sent{Path: map[string]string{"message": m.ID}, Body: map[string]any{"version": m.Version, "pick_state": "dismissed"}}, nil
		},
		Done: func(f map[string]any) string { return fieldText(f, "_done") },
	},
	readForward("mail.accept",
		"file a message held for the team (from someone outside the team, or unconfirmed); Jev reads each of its files, which uses credits and can't be undone.",
		`message (id from read messages)`, "POST /projects/{id}/mail/{message}/accept",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			m, err := heldMessage(ctx, d, who, project, fieldText(f, "message"))
			if err != nil {
				return sent{}, "", "", err
			}
			if m.HeldReason == store.MailHeldFull {
				return sent{}, "", "", refuse("message", "The message from %s arrived when held mail was full, so its files were not kept; there is nothing to file.", m.Sender)
			}
			files := 0
			for _, p := range m.Parts {
				if p.SHA256 != nil && p.DocumentID == "" {
					files++
				}
			}
			return sent{Path: map[string]string{"message": m.ID}},
				fmt.Sprintf("File the held message from %s (%s)? %s", m.Sender, fmt.Sprintf("%d file%s", files, plural(files)), costWords),
				fmt.Sprintf("Filing the message from %s.", m.Sender), nil
		}),
	{
		Name:   "mail.discard",
		About:  "delete a message held for the team, and its files; a discarded message can't be put back. A filed message can never be deleted.",
		Fields: `message (id from read messages)`,
		Route:  "POST /projects/{id}/mail/{message}/discard",
		Click:  true,
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			m, err := heldMessage(ctx, d, who, project, fieldText(f, "message"))
			if err != nil {
				return sent{}, err
			}
			f["_done"] = fmt.Sprintf("Discarded the held message from %s.", m.Sender)
			return sent{Path: map[string]string{"message": m.ID}}, nil
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
			m, err := heldMessage(ctx, d, who, project, fieldText(f, "message"))
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Discard the held message from %s and its files? It can't be put back.", m.Sender), nil
		},
		Done: func(f map[string]any) string { return fieldText(f, "_done") },
	},
}

// heldMessage finds a held message of this project by its id or short id.
// A filed message is refused in plain words: the record is frozen.
func heldMessage(ctx context.Context, d Deps, who opActor, project, ref string) (store.MailMessage, error) {
	if ref == "" {
		return store.MailMessage{}, refuse("message", "message is needed.")
	}
	m, err := messageRef(ctx, d, who, project, ref)
	if err != nil {
		return store.MailMessage{}, err
	}
	if m.Status != "held" {
		return store.MailMessage{}, refuse("message", "That message is filed. A filed message is part of the record: it can't be accepted again, edited or deleted.")
	}
	return m, nil
}

// recordMessage finds a filed message of this project by its id or short id.
func recordMessage(ctx context.Context, d Deps, who opActor, project, ref string) (store.MailMessage, error) {
	if ref == "" {
		return store.MailMessage{}, refuse("message", "message is needed.")
	}
	m, err := messageRef(ctx, d, who, project, ref)
	if err != nil {
		return store.MailMessage{}, err
	}
	if m.Status != "filed" {
		return store.MailMessage{}, refuse("message", "That message is held: accept or discard it first.")
	}
	return m, nil
}

// direction is "from" or "to", for a message's one line.
func direction(m store.MailMessage) string {
	if m.Direction == "out" {
		return "to"
	}
	return "from"
}

// messageRef reads one message of this project by id or short id, in SQL.
func messageRef(ctx context.Context, d Deps, who opActor, project, ref string) (store.MailMessage, error) {
	m, err := d.Store.MailMessageByRef(ctx, who.OrgID, project, ref)
	if errors.Is(err, store.ErrNotFound) {
		return m, refuse("message", "There is no message with id %q here, or it names more than one. Use an id from a read.", ref)
	}
	return m, err
}
