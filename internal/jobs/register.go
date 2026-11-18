package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// The register's reply match (docs/plans/2026-11-17-rfis-review-comments.md,
// Task 4). After a document is filed and its text extracted, Go's rule
// matches the Open RFIs it cites by number ("RFI-007", "RFI 7") or by the
// sender's ref. For correspondence, one Jev fan-out then asks Question A
// about the RFIs the rule left: one noul per RFI, all in one request. A
// reading is a proposal; only the PM's edit makes an RFI answered.

// RegisterQuestionVersion names Question A's wording; change it with the
// wording so stored readings say which they answered.
const RegisterQuestionVersion = "register.answers/1"

// maxRegisterCandidates caps the state: "accuracy falls as the state grows"
// (https://docs.typesafe.ai/model-jaggedness/jev-1.13).
const maxRegisterCandidates = 12

// registerTextLimit is how much of the document's text the state carries.
const registerTextLimit = 2400

// RegisterThresholds bands Question A's answers. Amber zero is no approved
// cut-off: every Jev reading is stored blank and none is shown (owner
// decision 4; data/register/thresholds.json is absent until the owner
// approves one). The cut-off is on the probability of the action taken
// (https://docs.typesafe.ai/confidence).
type RegisterThresholds struct {
	Amber float64 `json:"amber"`
}

func (t RegisterThresholds) band(p float64) string {
	if t.Amber > 0 && p >= t.Amber {
		return "amber"
	}
	return "blank"
}

func (w *Worker) register(ctx context.Context, job store.ClaimedJob) error {
	doc, rfis, err := w.Store.RegisterCandidates(ctx, job.OrgID, job.DocumentID)
	if err != nil || len(rfis) == 0 {
		return err
	}
	passages, err := w.Store.DocumentPassages(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return err
	}
	bodies := make([]string, len(passages))
	for i, p := range passages {
		bodies[i] = p.Body
	}
	text := strings.Join(bodies, "\n\n")
	readings, left := RuleMatches(text, rfis)
	if err := w.Store.SaveRegisterReadings(ctx, job, readings); err != nil {
		return err
	}
	if doc.Kind != "correspondence" || len(left) == 0 {
		return nil
	}
	call, asked := RegisterCall(doc, text, left)
	result, askErr := w.ask(ctx, call)
	var jevReadings []store.RegisterReading
	for _, r := range asked {
		key := registerKey(r.slot)
		a, ok := result.Answers[key]
		if askErr != nil || !ok || a.Type != jev.TypeNoul {
			// Unresolved or timed out: stored grey and never shown.
			jevReadings = append(jevReadings, store.RegisterReading{ItemID: r.rfi.ID, DecidedBy: "jev", Band: "grey", QuestionVersion: RegisterQuestionVersion})
			continue
		}
		p := a.Noul
		jevReadings = append(jevReadings, store.RegisterReading{ItemID: r.rfi.ID, DecidedBy: "jev", Band: w.RegisterThresholds.band(p), Probability: &p, QuestionVersion: RegisterQuestionVersion})
	}
	if err := w.Store.SaveRegisterReadings(ctx, job, jevReadings); err != nil {
		return err
	}
	// A failed call retries within the job's attempts; the grey readings
	// are replaced when it answers.
	return askErr
}

// RuleMatches is Go's rule: an Open RFI the text cites by our number or by
// the sender's ref. It returns those readings and the RFIs left for Jev.
func RuleMatches(text string, rfis []store.RegisterRFI) ([]store.RegisterReading, []store.RegisterRFI) {
	var hits []store.RegisterReading
	var left []store.RegisterRFI
	for _, r := range rfis {
		number := regexp.MustCompile(fmt.Sprintf(`(?i)\bRFI[\s\-#:.]*(?:No\.?\s*)?0*%d\b`, r.Number))
		switch {
		case number.MatchString(text):
			hits = append(hits, store.RegisterReading{ItemID: r.ID, DecidedBy: "rule", Band: "amber", Basis: fmt.Sprintf("cites RFI-%03d", r.Number)})
		case citesRef(text, r.Ref):
			hits = append(hits, store.RegisterReading{ItemID: r.ID, DecidedBy: "rule", Band: "amber", Basis: "cites ref " + r.Ref})
		default:
			left = append(left, r)
		}
	}
	return hits, left
}

// citesRef matches the sender's own number as a whole token. A ref of one
// or two characters is too common to match on.
func citesRef(text, ref string) bool {
	ref = strings.TrimSpace(ref)
	if len([]rune(ref)) < 3 {
		return false
	}
	re := regexp.MustCompile(`(?i)(^|[^\pL\pN])` + regexp.QuoteMeta(ref) + `($|[^\pL\pN])`)
	return re.MatchString(text)
}

type askedRFI struct {
	slot int
	rfi  store.RegisterRFI
}

func registerKey(slot int) string { return fmt.Sprintf("register.answers.r%d", slot) }

// RegisterCall builds Question A for the RFIs the rule left: the same
// discipline's package first, then the most recent, at most twelve. One
// noul per RFI, because one letter often answers several and TypeSafe says
// to "add one Noul per option when several can apply"
// (https://docs.typesafe.ai/patterns/companion-noul); one request
// (https://docs.typesafe.ai/patterns/fan-out). Jev compares no dates and
// counts nothing: the state is text only.
func RegisterCall(doc store.RegisterDocument, text string, rfis []store.RegisterRFI) (jev.Call, []askedRFI) {
	ordered := append([]store.RegisterRFI(nil), rfis...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i].Discipline == doc.Discipline && doc.Discipline != "", ordered[j].Discipline == doc.Discipline && doc.Discipline != ""
		if a != b {
			return a
		}
		return ordered[i].Number > ordered[j].Number
	})
	if len(ordered) > maxRegisterCandidates {
		ordered = ordered[:maxRegisterCandidates]
	}
	state := map[string]any{"document": map[string]any{"title": doc.Title, "kind": doc.Kind, "discipline": doc.Discipline, "text": registerExcerpt(text)}}
	list := map[string]any{}
	questions := map[string]jev.Question{}
	var asked []askedRFI
	for i, r := range ordered {
		slot := i + 1
		list[fmt.Sprintf("r%d", slot)] = map[string]any{"number": fmt.Sprintf("RFI-%03d", r.Number), "ref": r.Ref, "subject": r.Subject, "question": firstRunes(r.Question, 400), "to": r.OwnerText}
		questions[registerKey(slot)] = jev.Question{Type: jev.TypeNoul,
			Instructions: fmt.Sprintf("Using `document` and `rfis.r%d`, does the document reply to that request, giving the information, decision or instruction it asks for?", slot),
			Criteria: map[string]string{
				"true":  "The document states the answer to the request, or says the answer is attached.",
				"false": "The document is about something else, only mentions the topic, asks its own question, or is the request itself.",
			}}
		asked = append(asked, askedRFI{slot: slot, rfi: r})
	}
	state["rfis"] = list
	return jev.Call{State: state, Questions: questions, Priority: jev.PriorityBackground, QuestionVersion: RegisterQuestionVersion}, asked
}

// registerExcerpt is up to 2,400 characters of the text, cut around the
// first "RFI" or "response" so the reply's own words are in the state.
func registerExcerpt(text string) string {
	r := []rune(text)
	if len(r) <= registerTextLimit {
		return text
	}
	lower := strings.ToLower(text)
	at := -1
	for _, w := range []string{"rfi", "response"} {
		if i := strings.Index(lower, w); i >= 0 && (at < 0 || i < at) {
			at = i
		}
	}
	start := 0
	if at >= 0 {
		start = len([]rune(lower[:at])) - registerTextLimit/3
	}
	start = max(0, min(start, len(r)-registerTextLimit))
	return string(r[start : start+registerTextLimit])
}

func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// LoadRegisterThresholds reads dir/thresholds.json. A missing file is no
// cut-off; a file for another wording of Question A is refused, so an old
// cut-off can never band new answers.
func LoadRegisterThresholds(dir string) (RegisterThresholds, error) {
	body, err := os.ReadFile(filepath.Join(dir, "thresholds.json"))
	if os.IsNotExist(err) {
		return RegisterThresholds{}, nil
	}
	if err != nil {
		return RegisterThresholds{}, err
	}
	var t struct {
		QuestionVersion string  `json:"question_version"`
		Amber           float64 `json:"amber"`
	}
	if err := json.Unmarshal(body, &t); err != nil {
		return RegisterThresholds{}, err
	}
	if t.QuestionVersion != RegisterQuestionVersion || t.Amber <= 0 || t.Amber > 1 {
		return RegisterThresholds{}, fmt.Errorf("register thresholds: want question_version %q and an amber cut-off in (0, 1]", RegisterQuestionVersion)
	}
	return RegisterThresholds{Amber: t.Amber}, nil
}
