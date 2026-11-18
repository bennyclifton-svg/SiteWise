package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sitewise/internal/chatfix"
)

// The register's budgets (docs/plans/2026-11-17-rfis-review-comments.md,
// Task 7): a read of 1,000 rows with their readings (delivery_read), the
// same read with a System filter and its also-open line
// (delivery_system_read), and the screen's writes through POST /ops
// (delivery_write). Thirty samples each: more than five, so a slow cached
// generic plan after the fifth call on a connection would show.
func TestRegisterBudget(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Register budget")
	ctx := context.Background()
	docs, err := chatfix.SeedDocuments(ctx, a.store, m.orgID, project, []chatfix.Document{{Title: "Reply", Kind: "correspondence", Pages: []string{"Response to RFI-001."}}})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// 1,000 rows: three RFIs in four on the envelope, every tenth with a
	// possible answer to check.
	if _, err := pool.Exec(ctx, `INSERT INTO project_delivery_items(org_id,id,project_id,kind,title,status,number,system_id,target_date,details,origin,review_status)
 SELECT $1::uuid, gen_random_uuid(), $2::uuid, CASE WHEN g%4=0 THEN 'risk' ELSE 'rfi' END, 'Row '||g, 'open', CASE WHEN g%4=0 THEN NULL ELSE g END,
 CASE WHEN g%4=0 THEN 'substructure' ELSE 'envelope' END, DATE '2026-11-02' + (g%30), '{}', 'user', 'accepted_for_planning' FROM generate_series(1,1000) g`, m.orgID, project); err != nil {
		t.Fatal(err)
	}
	for _, doc := range docs {
		if _, err := pool.Exec(ctx, `INSERT INTO delivery_readings(org_id,id,project_id,item_id,document_id,field,decided_by,band,basis)
 SELECT org_id, gen_random_uuid(), project_id, id, $3::uuid, 'answers', 'rule', 'amber', 'cites the number' FROM project_delivery_items
 WHERE org_id=$1::uuid AND project_id=$2::uuid AND kind='rfi' AND number%10=1`, m.orgID, project, doc); err != nil {
			t.Fatal(err)
		}
	}
	base := "/api/projects/" + project + "/delivery"
	samples := map[string][]int64{}
	timed := func(path, method, url string, body []byte, want int, out any) {
		start := time.Now()
		m.call(t, method, url, body, want, out)
		samples[path] = append(samples[path], time.Since(start).Microseconds())
	}
	for i := 0; i < 30; i++ {
		var view struct {
			Items []json.RawMessage `json:"items"`
		}
		timed("delivery_read", http.MethodGet, base, nil, http.StatusOK, &view)
		if len(view.Items) != 1000+i {
			t.Fatalf("read %d rows", len(view.Items))
		}
		timed("delivery_system_read", http.MethodGet, base+"?system=envelope", nil, http.StatusOK, nil)
		add, _ := json.Marshal(map[string]any{"area": "delivery", "ops": []map[string]any{{"op": "delivery.add", "fields": map[string]any{"kind": "rfi", "title": fmt.Sprintf("Budget RFI %d", i), "raised_on": "2026-11-20"}}}})
		timed("delivery_write", http.MethodPost, "/api/projects/"+project+"/ops", add, http.StatusOK, nil)
		edit, _ := json.Marshal(map[string]any{"area": "delivery", "ops": []map[string]any{{"op": "delivery.edit", "fields": map[string]any{"item": fmt.Sprintf("RFI-%03d", 1000+i), "status": "open"}}}})
		timed("delivery_write", http.MethodPost, "/api/projects/"+project+"/ops", edit, http.StatusOK, nil)
	}
	for _, path := range []string{"delivery_read", "delivery_system_read", "delivery_write"} {
		s := append([]int64(nil), samples[path]...)
		t.Logf("%s: %d samples, p50 %d us, p90 %d us", path, len(s), pct(s, 50), pct(s, 90))
	}
	gateTender(t, []string{"delivery_read", "delivery_system_read", "delivery_write"}, samples)
	if a.jevHits.Load() != 0 {
		t.Fatal("the register called Jev")
	}
}

func pct(s []int64, p int) int64 {
	c := append([]int64(nil), s...)
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && c[j] < c[j-1]; j-- {
			c[j], c[j-1] = c[j-1], c[j]
		}
	}
	return c[(len(c)*p+99)/100-1]
}

// The consent conditions' budgets (docs/plans/2026-11-17-consent-conditions.md,
// Task 10): the register read with 300 condition rows (delivery_read), a
// condition edit through POST /ops (delivery_write), and Read conditions on
// a 120-condition notice (delivery_import: Go's split of the stored text and
// the queued job). Thirty samples each; no Jev.
func TestConsentBudget(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Consent budget")
	ctx := context.Background()
	notice := "NOTICE OF DETERMINATION\nDevelopment Application No: DA/2026/0120\nDate of determination: 2 March 2026\n\nPRIOR TO THE ISSUE OF A CONSTRUCTION CERTIFICATE\n"
	for i := 1; i <= 120; i++ {
		notice += fmt.Sprintf("%d. Heading %d\nThe applicant must provide the plan the certifier asks for under this condition, prepared by a suitably qualified person.\nReason: To ensure compliance.\n", i, i)
	}
	var seeds []chatfix.Document
	for i := 0; i < 30; i++ {
		seeds = append(seeds, chatfix.Document{Title: fmt.Sprintf("Notice %d", i), Kind: "statutory_instrument", Pages: []string{notice}})
	}
	docs, err := chatfix.SeedDocuments(ctx, a.store, m.orgID, project, seeds)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `INSERT INTO project_delivery_items(org_id,id,project_id,kind,title,status,system_id,details,origin,review_status,provenance,source_proposal_key)
 SELECT $1::uuid, gen_random_uuid(), $2::uuid, 'condition', 'Condition '||g||' · Heading '||g, 'open', 'site',
 jsonb_build_object('consent_ref','DA/2025/0001','number',g::text,'heading','Heading '||g,'text',repeat('The council''s words for this condition. ',20),'stage',(ARRAY['before_cc','before_work','during_works','before_occupation'])[1+g%4],'evidence_needed',true),
 'document','proposed', jsonb_build_object('reading',jsonb_build_object('fingerprint','f'||g)), 'consent:DA/2025/0001:'||g FROM generate_series(1,300) g`, m.orgID, project); err != nil {
		t.Fatal(err)
	}
	base := "/api/projects/" + project + "/delivery"
	samples := map[string][]int64{}
	timed := func(path, method, url string, body []byte, want int, out any) {
		start := time.Now()
		m.call(t, method, url, body, want, out)
		samples[path] = append(samples[path], time.Since(start).Microseconds())
	}
	i := 0
	for _, doc := range docs {
		var view struct {
			Items []json.RawMessage `json:"items"`
		}
		timed("delivery_read", http.MethodGet, base, nil, http.StatusOK, &view)
		if len(view.Items) != 300 {
			t.Fatalf("read %d rows", len(view.Items))
		}
		edit, _ := json.Marshal(map[string]any{"area": "delivery", "ops": []map[string]any{{"op": "delivery.edit", "fields": map[string]any{"item": fmt.Sprintf("condition %d", 1+i%300), "gate": "before_occupation"}}}})
		timed("delivery_write", http.MethodPost, "/api/projects/"+project+"/ops", edit, http.StatusOK, nil)
		body, _ := json.Marshal(map[string]string{"document_id": doc})
		timed("delivery_import", http.MethodPost, base+"/imports", body, http.StatusAccepted, nil)
		i++
	}
	for _, path := range []string{"delivery_read", "delivery_write", "delivery_import"} {
		s := append([]int64(nil), samples[path]...)
		t.Logf("%s: %d samples, p50 %d us, p90 %d us", path, len(s), pct(s, 50), pct(s, 90))
	}
	gateTender(t, []string{"delivery_read", "delivery_write", "delivery_import"}, samples)
	if a.jevHits.Load() != 0 {
		t.Fatal("the conditions' screen paths called Jev")
	}
}
