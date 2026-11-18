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

// DocumentHits is what one document holds for a search by subject: how many
// of its passages the full-text index matched, and how many the profile's
// reading labelled with one of the subject's building systems.
type DocumentHits struct {
	Passages int
	Labelled int
}

// ChatDocumentHits counts, for every document of the project, the passages
// that match the query (websearch form, like ChatSearchPassages) and the
// passages labelled with one of the systems or a part of one. It is one
// round trip whatever the project's size: the two counts are grouped in the
// database, so 200,000 passages never cross to Go. A document with neither
// is not in the map. An empty query or no systems skips that count.
//
// The labels are found by a range on passage_systems_by_system (migration
// 064): from "system" up to "system/" in the C collation, then kept only
// when they are the system or "system.part" ("system-x" sorts inside the
// range too). Without the range the count read every label in the table.
//
// A label on a wider system does not count: a passage labelled
// "substructure" is about footings as often as about the basement, so only
// the system itself or a part of it ("mechanical" finds
// "mechanical.ventilation") says the passage is about the subject.
func (s *Store) ChatDocumentHits(ctx context.Context, org, project, query string, systems []string) (map[string]DocumentHits, error) {
	if systems == nil {
		systems = []string{}
	}
	rows, err := s.pool.Query(ctx, `WITH q AS (SELECT websearch_to_tsquery('english',$3) AS tsq),
hit AS (
 SELECT p.document_id,count(*) AS n FROM passages p
 JOIN documents d ON d.org_id=p.org_id AND d.id=p.document_id, q
 WHERE $3<>'' AND p.org_id=$1::uuid AND d.project_id=$2::uuid AND p.body_tsv @@ q.tsq
 GROUP BY p.document_id),
lab AS (
 SELECT p.document_id,count(DISTINCT p.id) AS n FROM unnest($4::text[]) want
 JOIN passage_systems x ON x.org_id=$1::uuid
  AND x.system_id COLLATE "C" >= want AND x.system_id COLLATE "C" < want||'/'
  AND (x.system_id=want OR x.system_id LIKE want||'.%')
 JOIN passages p ON p.org_id=x.org_id AND p.id=x.passage_id
 JOIN documents d ON d.org_id=p.org_id AND d.id=p.document_id
 WHERE d.project_id=$2::uuid
 GROUP BY p.document_id)
SELECT COALESCE(hit.document_id,lab.document_id)::text,COALESCE(hit.n,0),COALESCE(lab.n,0)
FROM hit FULL JOIN lab ON lab.document_id=hit.document_id`, org, project, query, systems)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]DocumentHits{}
	for rows.Next() {
		var id string
		var h DocumentHits
		if err := rows.Scan(&id, &h.Passages, &h.Labelled); err != nil {
			return nil, err
		}
		out[id] = h
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

// ChatDocumentsWithText lists the documents of the project that have at
// least one passage. A filed document with none has no searchable text. It
// asks the passages index once per document rather than counting every
// passage: on 200,000 passages the count took 100 ms and only "any or
// none" was ever used.
func (s *Store) ChatDocumentsWithText(ctx context.Context, org, project string) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT d.id::text FROM documents d
WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid
 AND EXISTS (SELECT 1 FROM passages p WHERE p.org_id=d.org_id AND p.document_id=d.id)`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
