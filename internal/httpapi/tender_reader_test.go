package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/llm"
	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/tender"
	tsample "sitewise/internal/tender/sample"
)

// Every firm and figure in these tests is invented.

type tenderTableBody struct {
	priceTableBody
	Change string `json:"change"`
	Tender *struct {
		On      bool  `json:"on"`
		Version int64 `json:"version"`
		Rows    []struct {
			Key    string `json:"key"`
			Label  string `json:"label"`
			System bool   `json:"system"`
		} `json:"rows"`
		Columns     []tenderColumnBody `json:"columns"`
		Adjustments []struct {
			ID      string `json:"id"`
			Label   string `json:"label"`
			Origin  string `json:"origin"`
			Reason  string `json:"reason"`
			Row     string `json:"row"`
			Amounts map[string]struct {
				Amount *string `json:"amount"`
				State  string  `json:"state"`
				Source string  `json:"source"`
			} `json:"amounts"`
			Others map[string]struct {
				Low, High, Average *string
				Count              int
			} `json:"others"`
		} `json:"adjustments"`
		Suggested int `json:"suggested"`
	} `json:"tender"`
}

type tenderColumnBody struct {
	SubmissionID string  `json:"submission_id"`
	State        string  `json:"state"`
	Error        string  `json:"error"`
	TenderSum    *string `json:"tender_sum"`
	TenderSumBy  string  `json:"tender_sum_by"`
	LikeForLike  *string `json:"like_for_like"`
	Lowest       bool    `json:"lowest"`
	Checked      bool    `json:"checked"`
	Cells        map[string]struct {
		State  string  `json:"state"`
		Amount *string `json:"amount"`
		Lines  int     `json:"lines"`
		Mark   string  `json:"mark"`
	} `json:"cells"`
	Lines []struct {
		ID      string  `json:"id"`
		Label   string  `json:"label"`
		Amount  *string `json:"amount"`
		Row     string  `json:"row"`
		Kind    string  `json:"kind"`
		Counted bool    `json:"counted"`
		Page    int     `json:"page"`
		SplitOf string  `json:"split_of"`
	} `json:"lines"`
}

func (b tenderTableBody) column(sub string) *tenderColumnBody {
	if b.Tender == nil {
		return nil
	}
	for i := range b.Tender.Columns {
		if b.Tender.Columns[i].SubmissionID == sub {
			return &b.Tender.Columns[i]
		}
	}
	return nil
}

func tinyRender(_ context.Context, _ string, pages []int) (map[int]string, error) {
	out := map[int]string{}
	for _, p := range pages {
		out[p] = "/9j/4AAQSkZJRg=="
	}
	return out, nil
}

// countingAsker counts calls and answers as the stand-in, or fails.
type countingAsker struct {
	calls atomic.Int32
	err   error
}

func (c *countingAsker) Ask(ctx context.Context, req llm.Request) (llm.Response, error) {
	c.calls.Add(1)
	if c.err != nil {
		return llm.Response{}, c.err
	}
	return tsample.Model{}.Ask(ctx, req)
}

func withTender(ask llm.Asker, render tender.Renderer) func(*httpapi.Options) {
	return func(o *httpapi.Options) {
		o.TenderLLM, o.TenderModel, o.TenderRender, o.TenderTimeout = ask, "claude-haiku-5-5", render, 20*time.Second
	}
}

// tenderSetup is a town-planning package and the labels of its base rows.
func tenderSetup(t *testing.T, opts ...func(*httpapi.Options)) (*app, *member, string, string, []string) {
	t.Helper()
	a := newApp(t, append([]func(*httpapi.Options){withProfile(t), greenProposals}, opts...)...)
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Tender project")
	var pkg struct {
		ID string `json:"id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID
	var table priceTableBody
	m.getJSON(t, base+"/prices", &table)
	var labels []string
	for _, it := range table.Items {
		if it.Base {
			labels = append(labels, it.Label)
		}
	}
	if len(labels) < 2 {
		t.Fatal("no schedule")
	}
	return a, m, project, base, labels
}

func (m *member) modelSwitch(t *testing.T, on bool, want int) {
	t.Helper()
	m.call(t, http.MethodPut, "/api/account/model", []byte(`{"enabled":`+strconv.FormatBool(on)+`}`), want, nil)
}

// dropTender uploads a synthetic proposal priced at 1,000.00 a row, plus
// any extra lines, and returns its submission.
func (m *member) dropTender(t *testing.T, base, firm string, labels []string, extra ...psample.Line) string {
	t.Helper()
	var lines []psample.Line
	for _, l := range labels {
		lines = append(lines, psample.Line{Label: l, Cents: 100000})
	}
	lines = append(lines, extra...)
	pdf := psample.PDF(psample.Proposal{Firm: firm, ABN: "88071762537", Email: "office@invented.test", Lines: lines})
	var up struct {
		SubmissionID string `json:"submission_id"`
	}
	m.call(t, http.MethodPost, base+"/submissions?name="+url.QueryEscape("Fee proposal ("+firm+").pdf"), pdf, http.StatusCreated, &up)
	return up.SubmissionID
}

// waitTender polls the price table until the submission's tender state is
// one of want, or the old read has finished when want is "".
func (m *member) waitTender(t *testing.T, base, sub string, want ...string) tenderTableBody {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		var table tenderTableBody
		m.getJSON(t, base+"/prices", &table)
		col := table.column(sub)
		state := ""
		if col != nil {
			state = col.State
		}
		read := false
		for _, c := range table.Columns {
			read = read || (c.Submission.ID == sub && c.Submission.Status != "reading")
		}
		for _, w := range want {
			if read && state == w {
				return table
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("tender state %q, want %v", state, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func money(t *testing.T, v *string) float64 {
	t.Helper()
	if v == nil {
		t.Fatal("amount is null")
	}
	f, err := strconv.ParseFloat(*v, 64)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAccountModelSwitch(t *testing.T) {
	a := newApp(t, withTender(tsample.Model{}, tinyRender))
	m := a.member(t, newUUID(t))
	var account struct {
		Model struct {
			Available bool       `json:"available"`
			Enabled   bool       `json:"enabled"`
			EnabledAt *time.Time `json:"enabled_at"`
		} `json:"model"`
	}
	m.getJSON(t, "/api/account", &account)
	if !account.Model.Available || account.Model.Enabled {
		t.Fatalf("%+v", account.Model)
	}
	m.modelSwitch(t, true, http.StatusOK)
	m.getJSON(t, "/api/account", &account)
	if !account.Model.Enabled || account.Model.EnabledAt == nil {
		t.Fatalf("%+v", account.Model)
	}
	other := a.member(t, newUUID(t))
	other.getJSON(t, "/api/account", &account)
	if account.Model.Enabled {
		t.Fatal("another org was switched on")
	}
	anon := &member{app: a, client: &http.Client{}}
	if got := anon.status(t, http.MethodPut, "/api/account/model", []byte(`{"enabled":true}`)); got != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", got)
	}
	if got := m.status(t, http.MethodPut, "/api/account/model", []byte(`{}`)); got != http.StatusBadRequest {
		t.Fatalf("empty body: %d", got)
	}
	// A server with no model key says so.
	plain := newApp(t)
	p := plain.member(t, newUUID(t))
	p.modelSwitch(t, true, http.StatusConflict)
	p.getJSON(t, "/api/account", &account)
	if account.Model.Available {
		t.Fatal("a server with no key says a model is available")
	}
}

func TestDroppedTenderIsReadIntoLines(t *testing.T) {
	ask := &countingAsker{}
	a, m, project, base, labels := tenderSetup(t, withTender(ask, tinyRender))
	m.modelSwitch(t, true, http.StatusOK)
	sub := m.dropTender(t, base, "Invented Planning Pty Ltd", labels, psample.Line{Label: "Zebra crossing study", Cents: 50000})
	table := m.waitTender(t, base, sub, "read")
	col := table.column(sub)
	want := float64(len(labels))*1000 + 500
	if !table.Tender.On || money(t, col.TenderSum) != want || col.Checked || len(col.Lines) < len(labels)+2 {
		t.Fatalf("on %v sum %v lines %d", table.Tender.On, col.TenderSum, len(col.Lines))
	}
	var sum float64
	for _, row := range table.Tender.Rows {
		if cell, ok := col.Cells[row.Key]; ok && cell.Amount != nil {
			sum += money(t, cell.Amount)
		}
	}
	if sum != want {
		t.Fatalf("cells add to %v, tender sum %v", sum, want)
	}
	last := table.Tender.Rows[len(table.Tender.Rows)-1]
	if !last.System || money(t, col.Cells[last.Key].Amount) != 500 {
		t.Fatalf("the unmatched line is not in Not broken down: %+v", col.Cells[last.Key])
	}
	if ask.calls.Load() != 1 {
		t.Fatalf("%d model calls", ask.calls.Load())
	}
	// Read again: one more call, read again, still to check.
	read := "/api/projects/" + project + "/submissions/" + sub + "/tender-read"
	m.call(t, http.MethodPost, read, nil, http.StatusOK, nil)
	deadline := time.Now().Add(20 * time.Second)
	for ask.calls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	again := m.waitTender(t, base, sub, "read")
	if ask.calls.Load() != 2 || again.Tender.Version <= table.Tender.Version || again.column(sub).Checked {
		t.Fatalf("calls %d version %d -> %d", ask.calls.Load(), table.Tender.Version, again.Tender.Version)
	}
	// Another org sees nothing of it.
	other := a.member(t, newUUID(t))
	if got := other.status(t, http.MethodPost, read, nil); got != http.StatusNotFound {
		t.Fatalf("another org: %d", got)
	}
	if got := other.status(t, http.MethodGet, base+"/prices", nil); got != http.StatusNotFound {
		t.Fatalf("another org read the table: %d", got)
	}
}

func TestSwitchOffNothingIsSent(t *testing.T) {
	ask := &countingAsker{}
	_, m, project, base, labels := tenderSetup(t, withTender(ask, tinyRender))
	sub := m.dropTender(t, base, "Invented Planning Pty Ltd", labels)
	table := m.waitTender(t, base, sub, "")
	time.Sleep(300 * time.Millisecond)
	if ask.calls.Load() != 0 || table.Tender.On || len(table.Tender.Columns) != 0 {
		t.Fatalf("calls %d on %v columns %d", ask.calls.Load(), table.Tender.On, len(table.Tender.Columns))
	}
	if got := m.status(t, http.MethodPost, "/api/projects/"+project+"/submissions/"+sub+"/tender-read", nil); got != http.StatusConflict {
		t.Fatalf("read again with the switch off: %d", got)
	}
	if ask.calls.Load() != 0 {
		t.Fatal("a page left the server with the switch off")
	}
}

func TestPlacingASubmissionQueuesIt(t *testing.T) {
	ask := &countingAsker{}
	_, m, project, base, labels := tenderSetup(t, withTender(ask, tinyRender))
	sub := m.dropTender(t, base, "Invented Planning Pty Ltd", labels)
	m.waitTender(t, base, sub, "")
	m.modelSwitch(t, true, http.StatusOK)
	var pkg struct {
		ID string `json:"id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.civil"}`), http.StatusCreated, &pkg)
	if pkg.ID == "" {
		t.Skip("no second discipline row in this catalogue")
	}
	m.call(t, http.MethodPatch, "/api/projects/"+project+"/submissions/"+sub, []byte(`{"package_id":"`+pkg.ID+`"}`), http.StatusOK, nil)
	m.waitTender(t, "/api/projects/"+project+"/procurement/packages/"+pkg.ID, sub, "read")
	if ask.calls.Load() != 1 {
		t.Fatalf("%d model calls", ask.calls.Load())
	}
}

func TestTenderFailuresTellTheScreen(t *testing.T) {
	for _, c := range []struct {
		name   string
		ask    *countingAsker
		render tender.Renderer
		words  string
	}{
		{"transport", &countingAsker{err: errors.New("connection reset by peer at 10.0.0.1")}, tinyRender, "The model could not be reached. It will be tried again."},
		{"refusal", &countingAsker{err: llm.ErrRefused}, tinyRender, "The model declined to read this file."},
		{"no renderer", &countingAsker{}, nil, "Page images cannot be made on this server."},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, m, _, base, labels := tenderSetup(t, withTender(c.ask, c.render))
			m.modelSwitch(t, true, http.StatusOK)
			sub := m.dropTender(t, base, "Invented Planning Pty Ltd", labels)
			table := m.waitTender(t, base, sub, "failed")
			if col := table.column(sub); col.Error != c.words || strings.Contains(col.Error, "10.0.0.1") {
				t.Fatalf("error %q", col.Error)
			}
			// The old reading still shows its figures.
			for _, col := range table.Columns {
				if col.Submission.ID == sub && len(col.Cells) == 0 {
					t.Fatal("the old reading's cells are gone")
				}
			}
		})
	}
}
