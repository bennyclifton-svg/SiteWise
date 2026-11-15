package store

import (
	"context"
)

// ChatPassage is a passage with where it sits in its document, for the
// chat's document tools. Page is 0 when the reader recorded none.
type ChatPassage struct {
	ID         string
	DocumentID string
	Ordinal    int
	Page       int
	Section    string
	Body       string
	// Hits is how many passages of the same document matched the search.
	Hits int
}

const chatPassageColumns = `p.id::text,p.document_id::text,p.ordinal,COALESCE(ps.page,0),COALESCE(ps.section,''),p.body`

const chatPassageFrom = ` FROM passages p
JOIN documents d ON d.org_id=p.org_id AND d.id=p.document_id
LEFT JOIN passage_sources ps ON ps.org_id=p.org_id AND ps.passage_id=p.id`

// ChatSearchPassages runs the full-text index over one project's filed
// documents, or over one of them. The query is in websearch form, so
// alternatives are joined with "or". Another org's or another project's
// passages are never in the result.
func (s *Store) ChatSearchPassages(ctx context.Context, org, project, query, document string, limit int) ([]ChatPassage, error) {
	rows, err := s.pool.Query(ctx, `WITH q AS (SELECT websearch_to_tsquery('english',$3) AS tsq)
SELECT `+chatPassageColumns+`,count(*) OVER (PARTITION BY p.document_id)`+chatPassageFrom+`, q
WHERE p.org_id=$1::uuid AND d.project_id=$2::uuid AND ($4='' OR p.document_id::text=$4) AND p.body_tsv @@ q.tsq
ORDER BY ts_rank(p.body_tsv,q.tsq) DESC, p.document_id, p.ordinal LIMIT $5`, org, project, query, document, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatPassage
	for rows.Next() {
		var p ChatPassage
		if err := rows.Scan(&p.ID, &p.DocumentID, &p.Ordinal, &p.Page, &p.Section, &p.Body, &p.Hits); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ChatDocumentPassages reads one document of the project in order. With a
// page range it returns the passages on those pages; a document whose
// passages carry no page numbers is read whole.
func (s *Store) ChatDocumentPassages(ctx context.Context, org, project, document string, from, to int) ([]ChatPassage, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+chatPassageColumns+`,0`+chatPassageFrom+`
WHERE p.org_id=$1::uuid AND d.project_id=$2::uuid AND p.document_id::text=$3
 AND ($4=0 OR COALESCE(ps.page,0)=0 OR COALESCE(ps.page,0) BETWEEN $4 AND $5)
ORDER BY p.ordinal`, org, project, document, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatPassage
	for rows.Next() {
		var p ChatPassage
		if err := rows.Scan(&p.ID, &p.DocumentID, &p.Ordinal, &p.Page, &p.Section, &p.Body, &p.Hits); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ChatSystemPassages lists passages the profile's reading labelled with one
// of the given building systems (or a part of one: "mechanical" also finds
// "mechanical.ventilation").
func (s *Store) ChatSystemPassages(ctx context.Context, org, project string, systems []string, limit int) ([]ChatPassage, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT ON (p.document_id,p.ordinal) `+chatPassageColumns+`,0`+chatPassageFrom+`
JOIN passage_systems x ON x.org_id=p.org_id AND x.passage_id=p.id
WHERE p.org_id=$1::uuid AND d.project_id=$2::uuid
 AND EXISTS (SELECT 1 FROM unnest($3::text[]) want WHERE x.system_id=want OR x.system_id LIKE want||'.%' OR want LIKE x.system_id||'.%')
ORDER BY p.document_id,p.ordinal LIMIT $4`, org, project, systems, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatPassage
	for rows.Next() {
		var p ChatPassage
		if err := rows.Scan(&p.ID, &p.DocumentID, &p.Ordinal, &p.Page, &p.Section, &p.Body, &p.Hits); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DocumentFirm says a document is a firm's submission to a package.
type DocumentFirm struct {
	Firm    string
	Package string
	Status  string // received, awarded, unsuccessful, …
}

// DocumentFirms maps each document of the project that is a tender or fee
// proposal to its firm, so an answer can say whose document it quotes.
func (s *Store) DocumentFirms(ctx context.Context, org, project string) (map[string]DocumentFirm, error) {
	rows, err := s.pool.Query(ctx, `SELECT s.document_id::text,COALESCE(f.name,''),COALESCE(p.title,''),s.status
FROM submissions s
LEFT JOIN firms f ON f.org_id=s.org_id AND f.id=s.firm_id
LEFT JOIN packages p ON p.org_id=s.org_id AND p.id=s.package_id
WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid AND s.document_id IS NOT NULL`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]DocumentFirm{}
	for rows.Next() {
		var id string
		var f DocumentFirm
		if err := rows.Scan(&id, &f.Firm, &f.Package, &f.Status); err != nil {
			return nil, err
		}
		out[id] = f
	}
	return out, rows.Err()
}

// ChatPassageCounts is how many passages each document of the project has.
// A filed document with none has no searchable text.
func (s *Store) ChatPassageCounts(ctx context.Context, org, project string) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT p.document_id::text,count(*) FROM passages p
JOIN documents d ON d.org_id=p.org_id AND d.id=p.document_id
WHERE p.org_id=$1::uuid AND d.project_id=$2::uuid GROUP BY p.document_id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
