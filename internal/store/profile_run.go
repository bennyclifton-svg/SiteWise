package store

// Reads for the profile's run strip (docs/design/2026-11-10-run-time-feedback.md):
// where each document's reading is, which passages a pass has just read, and
// what an update changes elsewhere. Each is a bounded, org-scoped read of
// state the worker and the other surfaces already keep; none calls Jev.

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// ProfileReadingDocument is one document in a profile update.
type ProfileReadingDocument struct {
	DocumentID string `json:"document_id"`
	Filename   string `json:"filename"`
	// State is queued, reading or failed. A document whose passes are done
	// is not listed; the browser remembers it from earlier reads.
	State string `json:"state"`
	// Pass is label (what each passage speaks to) or evidence (the systems'
	// questions), the pass the state belongs to.
	Pass     string `json:"pass"`
	Units    int    `json:"units"`
	Labelled int    `json:"labelled"`
	Evidence int    `json:"evidence"`
}

// ProfileProgress is what a profile update is doing now.
type ProfileProgress struct {
	Revision int64 `json:"revision"`
	// Preparing counts documents whose text is still being prepared.
	Preparing int                      `json:"preparing"`
	Documents []ProfileReadingDocument `json:"documents"`
}

func (s *Store) ProfileProgress(ctx context.Context, org, project string) (ProfileProgress, error) {
	out := ProfileProgress{Documents: []ProfileReadingDocument{}}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	// Document ids first, as ReadProfile does, so stale statistics cannot turn
	// the job lookups into project-wide scans (F31).
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid),
 COALESCE((SELECT revision FROM profile_builds WHERE org_id=$1::uuid AND project_id=$2::uuid),0),
 (SELECT count(DISTINCT j.document_id) FROM jobs j
  WHERE j.org_id=$1::uuid AND j.document_id = ANY(ARRAY(SELECT id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid))
  AND j.kind='full_text' AND j.status IN ('queued','leased'))`, org, project).Scan(&exists, &out.Revision, &out.Preparing); err != nil {
		return out, err
	}
	if !exists {
		return out, ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT d.id::text, d.filename, j.kind, j.status, j.status='leased' AND j.locked_until > now()
 FROM jobs j JOIN documents d ON d.org_id=j.org_id AND d.id=j.document_id
 WHERE j.org_id=$1::uuid AND j.document_id = ANY(ARRAY(SELECT id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid))
 AND j.kind IN ('label','evidence') AND j.status IN ('queued','leased','failed')
 ORDER BY d.filename, d.id`, org, project)
	if err != nil {
		return out, err
	}
	index := map[string]int{}
	for rows.Next() {
		var id, filename, kind, status string
		var live bool
		if err := rows.Scan(&id, &filename, &kind, &status, &live); err != nil {
			rows.Close()
			return out, err
		}
		state := "queued" // an expired lease waits to be claimed again
		if live {
			state = "reading"
		} else if status == "failed" {
			state = "failed"
		}
		at, seen := index[id]
		if !seen {
			index[id] = len(out.Documents)
			out.Documents = append(out.Documents, ProfileReadingDocument{DocumentID: id, Filename: filename, State: state, Pass: kind})
			continue
		}
		// Reading outranks a failure, and a failure outranks waiting.
		if rank(state) > rank(out.Documents[at].State) {
			out.Documents[at].State, out.Documents[at].Pass = state, kind
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Documents) > 0 {
		ids := make([]string, len(out.Documents))
		for i, d := range out.Documents {
			ids[i] = d.DocumentID
		}
		// The same counts as source coverage, for the listed documents only.
		counts, err := tx.Query(ctx, `SELECT d.id::text,
 (SELECT count(*) FROM passages p WHERE p.org_id=$1::uuid AND p.document_id=d.id),
 (SELECT count(*) FROM passage_sources ps WHERE ps.org_id=$1::uuid AND ps.document_id=d.id AND ps.outcome<>'pending'),
 (SELECT count(*) FROM passage_calls pc WHERE pc.org_id=$1::uuid AND pc.document_id=d.id AND pc.stage='evidence')
 FROM unnest($2::uuid[]) AS d(id)`, org, ids)
		if err != nil {
			return out, err
		}
		for counts.Next() {
			var id string
			var units, labelled, evidence int
			if err := counts.Scan(&id, &units, &labelled, &evidence); err != nil {
				counts.Close()
				return out, err
			}
			d := &out.Documents[index[id]]
			d.Units, d.Labelled, d.Evidence = units, labelled, evidence
		}
		counts.Close()
		if err := counts.Err(); err != nil {
			return out, err
		}
	}
	return out, tx.Commit(ctx)
}

func rank(state string) int {
	switch state {
	case "reading":
		return 2
	case "failed":
		return 1
	}
	return 0
}

// ProfileFind is one passage a pass has just read, and what it spoke to.
type ProfileFind struct {
	PassageID  string `json:"passage_id"`
	DocumentID string `json:"document_id"`
	// Stage is label or evidence.
	Stage   string    `json:"stage"`
	Page    int       `json:"page"`
	Keys    []string  `json:"keys"`
	Systems []string  `json:"systems"`
	Excerpt string    `json:"excerpt"`
	At      time.Time `json:"at"`
}

// maxFinds bounds one read; the browser asks again from the last time it saw.
const maxFinds = 200

// ProfileFinds lists passages of the given documents read after a time,
// oldest first. Documents outside the project are ignored.
func (s *Store) ProfileFinds(ctx context.Context, org, project string, documents []string, after time.Time) ([]ProfileFind, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	if len(documents) == 0 {
		return []ProfileFind{}, nil
	}
	return profileFinds(ctx, s.pool, org, project, documents, after)
}

func profileFinds(ctx context.Context, q rowQuerier, org, project string, documents []string, after time.Time) ([]ProfileFind, error) {
	rows, err := q.Query(ctx, `SELECT pc.passage_id::text, p.document_id::text, pc.stage, COALESCE(ps.page,0), COALESCE(ps.mapped_keys,'{}'),
 ARRAY(SELECT system_id FROM passage_systems ss WHERE ss.org_id=p.org_id AND ss.passage_id=p.id ORDER BY system_id),
 left(p.body,160), pc.created_at
 FROM passages p
 JOIN documents d ON d.org_id=p.org_id AND d.id=p.document_id AND d.project_id=$2::uuid
 JOIN passage_calls pc ON pc.org_id=p.org_id AND pc.passage_id=p.id AND pc.stage IN ('label','evidence')
 LEFT JOIN passage_sources ps ON ps.org_id=p.org_id AND ps.passage_id=p.id
 WHERE p.org_id=$1::uuid AND p.document_id = ANY($3::uuid[]) AND pc.created_at > $4
 ORDER BY pc.created_at, pc.passage_id, pc.stage
 LIMIT $5`, org, project, documents, after, maxFinds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProfileFind{}
	for rows.Next() {
		var f ProfileFind
		if err := rows.Scan(&f.PassageID, &f.DocumentID, &f.Stage, &f.Page, &f.Keys, &f.Systems, &f.Excerpt, &f.At); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ProfileNext counts what the other surfaces now propose or flag, so the
// profile can say where an update matters next. Each count is that
// surface's own: nothing here is estimated.
type ProfileNext struct {
	// Procurement: disciplines proposed that are not packages yet.
	Procurement int `json:"procurement"`
	// Costs: cost lines proposed that are not in the plan yet.
	Costs int `json:"costs"`
	// Program: changes waiting for review.
	Program int `json:"program"`
	// Drafts: RFP and RFT drafts built from an earlier profile.
	Drafts int `json:"drafts"`
}

func (s *Store) ProfileNext(ctx context.Context, org, project string) (ProfileNext, error) {
	var out ProfileNext
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid),
 (SELECT count(*) FROM program_changes c JOIN programs g ON g.org_id=c.org_id AND g.id=c.program_id
  WHERE c.org_id=$1::uuid AND g.project_id=$2::uuid AND c.status='pending'),
 (SELECT count(*) FROM reports r JOIN report_versions v ON v.org_id=r.org_id AND v.id=r.current_draft_version_id
  WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.kind IN ('rfp','rft') AND v.status='draft'
  AND COALESCE((v.source_revisions->>'profile_revision')::bigint,0)
   <> COALESCE((SELECT revision FROM profile_builds WHERE org_id=$1::uuid AND project_id=$2::uuid),0))`,
		org, project).Scan(&exists, &out.Program, &out.Drafts); err != nil {
		return out, err
	}
	if !exists {
		return out, ErrNotFound
	}
	// The two proposal reads are independent; side by side they cost the slower one.
	var wg sync.WaitGroup
	var packagesErr, costsErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		o, err := s.ReadPackageProposals(ctx, org, project)
		out.Procurement, packagesErr = len(o.Suggestions), err
	}()
	go func() {
		defer wg.Done()
		plan, err := s.ReadCostPlan(ctx, org, project, "")
		out.Costs, costsErr = len(plan.Suggestions), err
	}()
	wg.Wait()
	if packagesErr != nil {
		return out, packagesErr
	}
	return out, costsErr
}
