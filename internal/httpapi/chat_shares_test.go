package httpapi_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"sitewise/internal/llm"
)

// guestChat is the guest fixture with the stand-in chat model switched on.
func guestChat(t *testing.T) (*guestFixture, *toolApp) {
	t.Helper()
	ta := &toolApp{reply: func([]string) llm.Turn { return llm.Turn{Text: "ok"} }}
	ta.model = ta.script()
	f := newGuestFixture(t, withChat(t, ta.model))
	ta.a, ta.m, ta.project = f.a, f.m, f.project
	ta.m.modelSwitch(t, true, http.StatusOK)
	return f, ta
}

func shareView(events []chatEvent) *chatEvent {
	for i := range events {
		if events[i].Type == "view" {
			return &events[i]
		}
	}
	return nil
}

// From chat (guest access plan): Share opens filled in from checked ids and
// makes no link; anything but documents, an issued report or an issued RFT
// is refused; read(shares) prints who holds a link and never a token; a
// revoke is prepared and waits for the PM's click.
func TestChatSharesAndRevokes(t *testing.T) {
	f, ta := guestChat(t)
	grants := func() int { return f.count(t, `SELECT count(*) FROM guest_grants WHERE org_id=$1::uuid`, f.m.orgID) }

	// "Share the RFT with Acme": Share opens on Acme's cell; no link is made.
	events := ta.run(t, "Share the RFT with Acme", tool("view", map[string]any{"do": "open", "dialog": "share", "fields": map[string]string{"kind": "package", "package": f.pkg[:8], "firm": "Acme"}, "final": true}))
	v := shareView(events)
	if v == nil || v.View.Dialog != "share" || v.View.Fields["package"] != f.pkg || v.View.Fields["firm"] != f.firmA || v.View.Fields["firm_name"] != "Acme Planning" || !strings.Contains(v.View.Line, "Press Create link") {
		t.Fatalf("share view %+v (results %q)", v, ta.results)
	}
	for _, e := range events {
		if e.Card != nil {
			t.Fatalf("opening Share made a %s card", e.Type)
		}
	}
	if grants() != 0 {
		t.Fatal("chat made a link")
	}
	// Documents by id; a foreign or sibling document is refused.
	events = ta.run(t, "Share A-101 with the client", tool("view", map[string]any{"do": "open", "dialog": "share", "fields": map[string]string{"kind": "documents", "documents": f.d1[:8], "shared_with": "client@example.test", "days": "14"}, "final": true}))
	if v := shareView(events); v == nil || v.View.Fields["documents"] != f.d1 || v.View.Fields["shared_with"] != "client@example.test" || v.View.Fields["days"] != "14" {
		t.Fatalf("documents view %+v", v)
	}
	for _, bad := range []map[string]string{
		{"kind": "documents", "documents": f.dSib},
		{"kind": "documents", "documents": f.dForeign},
		{"kind": "package", "package": f.pkg, "firm": firmBName},
		{"kind": "report", "report": f.rftDraft},
		{"kind": "cost_plan"},
		{"kind": "documents", "documents": f.d1, "days": "400"},
	} {
		events = ta.run(t, "Share it", tool("view", map[string]any{"do": "open", "dialog": "share", "fields": bad, "final": true}))
		if v := shareView(events); v != nil {
			t.Fatalf("%v opened Share: %+v", bad, v.View)
		}
	}
	events = ta.run(t, "Share the cost plan with the client", tool("view", map[string]any{"do": "open", "dialog": "share", "fields": map[string]string{"kind": "cost_plan"}, "final": true}))
	if len(ta.results) != 1 || !strings.Contains(ta.results[0], "Only documents, an issued report or an issued RFP or RFT can be shared.") {
		t.Fatalf("cost plan refusal %q", ta.results)
	}

	// The PM makes two links; chat reads them, with no token.
	link, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	_, reportToken := f.share(t, map[string]any{"kind": "report", "report_id": f.pmp, "shared_with": "client@example.test"})
	f.guest(t, reportToken)
	resp := f.m.do(t, http.MethodGet, "/api/projects/"+f.project+"/chat/read?area=shares", nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	read := string(raw)
	if resp.StatusCode != 200 || !strings.Contains(read, "acme@example.test") || !strings.Contains(read, "never opened") || !strings.Contains(read, "opened 1 time") || strings.Contains(read, token) || strings.Contains(read, reportToken) {
		t.Fatalf("read shares %d %s", resp.StatusCode, read)
	}
	resp = f.m.do(t, http.MethodGet, "/api/projects/"+f.project+"/chat/read?area=procurement", nil)
	raw, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(raw), "Acme Planning | invited | link not opened") {
		t.Fatalf("procurement read %s", raw)
	}
	// The named filter: rows with a firm that has not opened its link.
	events = ta.run(t, "Who has not opened the planning RFT", tool("view", map[string]any{"do": "only", "table": "procurement", "filter": "link_not_opened", "final": true}))
	if v := shareView(events); v == nil || len(v.View.IDs) != 1 || v.View.IDs[0] != f.pkg {
		t.Fatalf("filter view %+v (results %q)", v, ta.results)
	}

	// "Revoke Acme's link": prepared, nothing withdrawn until the click.
	card := ta.prepareRead(t, "shares", "shares.revoke", map[string]any{"share": link.Share.ID[:8]})
	cardSays(t, card, "acme@example.test", "can't be put back")
	acme := &guest{f: f, client: &http.Client{}}
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+token+`"}`), true); code != http.StatusNoContent {
		t.Fatalf("the link stopped before the click: %d", code)
	}
	ta.click(t, card)
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+token+`"}`), true); code != http.StatusUnauthorized {
		t.Fatalf("the link works after the click: %d", code)
	}
	ta.refusedRead(t, "shares", "shares.revoke", map[string]any{"share": link.Share.ID}, "no longer works")
	ta.refusedRead(t, "shares", "shares.revoke", map[string]any{"share": newUUID(t)}, "There is no link")
}
