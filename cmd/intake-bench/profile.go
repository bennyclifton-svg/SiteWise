package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/eval"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// profileRebuild measures the saved Spec Home workload, including database
// reads, canonical hashing, reconciliation, row writes and event commit. The
// private source fixture is never read from a running database by a benchmark.
func (a *apiClient) profileRebuild(ctx context.Context, st *store.Store, dsn, fixture string, cat *knowledge.Catalog, thresholds profile.Thresholds, reading profile.ReadPolicy, n int, measureProposals, manual0991, integrated bool) (string, error) {
	raw, err := os.ReadFile(fixture)
	if err != nil {
		return "", fmt.Errorf("Spec Home fixture: %w (tools/export_profile_bench.py)", err)
	}
	var tables map[string][]map[string]json.RawMessage
	if manual0991 {
		tables, err = load0991Benchmark(fixture)
	} else {
		err = json.Unmarshal(raw, &tables)
	}
	if err != nil {
		return "", err
	}
	if !manual0991 && (len(tables["documents"]) < 38 || len(tables["profile_facts"]) < 1153 || len(tables["passages"]) < 6742 || len(tables["sites"]) != 1) {
		return "", fmt.Errorf("Spec Home fixture is smaller than the frozen WP-14 workload")
	}
	label, prefix, editPath := "Spec Home", "", "spec_home_profile_edit"
	if manual0991 {
		label, prefix, editPath = "0991 manual source timing", "0991_", "0991_profile_edit"
	}
	project, err := a.createProject(ctx, label+" rebuild benchmark")
	if err != nil {
		return "", err
	}
	site, err := st.ProjectSite(ctx, benchOrg, project)
	if err != nil {
		return "", err
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var userID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE org_id=$1::uuid ORDER BY id LIMIT 1`, benchOrg).Scan(&userID); err != nil {
		return "", err
	}
	// Table names are a fixed allowlist; all fixture values are bound parameters.
	order := []string{"project_parts", "files", "documents", "decisions", "supersessions", "passages", "passage_sources", "document_sources", "profile_facts", "profile_user_values", "profile_planning_values", "work_items"}
	batch := &pgx.Batch{}
	for _, table := range order {
		for _, row := range tables[table] {
			set := func(key, value string) { row[key], _ = json.Marshal(value) }
			set("org_id", benchOrg)
			for key, value := range map[string]string{"project_id": project, "created_by_project_id": project, "site_id": site.ID, "user_id": userID} {
				if old, ok := row[key]; ok && string(old) != "null" {
					set(key, value)
				}
			}
			delete(row, "body_tsv")
			columns := make([]string, 0, len(row))
			for key := range row {
				columns = append(columns, key)
			}
			sort.Strings(columns)
			quoted := make([]string, len(columns))
			for i, key := range columns {
				quoted[i] = pgx.Identifier{key}.Sanitize()
			}
			list := strings.Join(quoted, ",")
			body, err := json.Marshal(row)
			if err != nil {
				return "", err
			}
			name := pgx.Identifier{table}.Sanitize()
			batch.Queue(`INSERT INTO `+name+` (`+list+`) SELECT `+list+` FROM jsonb_populate_record(NULL::`+name+`,$1::jsonb)`, body)
		}
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return "", fmt.Errorf("import Spec Home: %w", err)
	}
	if err := store.BumpRevision(ctx, tx, benchOrg, project, "profile_inputs"); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	build := store.ProfileBuild{Catalog: cat, KnowledgeVersion: cat.Version(), QuestionVersion: profile.QuestionVersion, ThresholdsVersion: thresholds.Version, ReadKinds: reading.Kinds(), Compute: func(s store.ProfileSnapshot) []profile.Row {
		return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: thresholds, Read: reading}, cat)
	}}
	configured := st.WithProfile(build)
	proposalCount := 0
	var evaluator *works.Evaluator
	if measureProposals {
		evaluator, err = works.NewEvaluator(cat)
		if err != nil {
			return "", err
		}
	}
	for i := 0; i < n; i++ {
		start := time.Now()
		var err error
		if integrated {
			err = configured.RebuildProfileWithProposals(ctx, benchOrg, project, evaluator)
		} else {
			err = configured.RebuildProfile(ctx, benchOrg, project, thresholds.Version, build.Compute)
		}
		rebuildElapsed := time.Since(start)
		a.samples.Add(prefix+"profile_rebuild", rebuildElapsed)
		if err != nil {
			return "", err
		}
		if integrated {
			a.samples.Add(prefix+"proposal_rebuild_integrated", rebuildElapsed)
			proposals, err := configured.ReadProposals(ctx, benchOrg, project)
			if err != nil {
				return "", err
			}
			proposalCount = len(proposals)
		} else if measureProposals {
			// Read the actual post-rebuild canonical set. This diagnostic includes
			// extra reads/reconciliation, but excludes future projection writes.
			snapshot, err := st.ProfileInput(ctx, benchOrg, project)
			if err != nil {
				return "", err
			}
			items, err := st.ReadWorks(ctx, benchOrg, project)
			if err != nil {
				return "", err
			}
			rows := build.Compute(snapshot)
			computeStart := time.Now()
			inputs, err := profile.ProposalInputs(rows, snapshot.Parts, items, cat)
			if err != nil {
				return "", err
			}
			proposals, err := evaluator.Evaluate(inputs)
			if err != nil {
				return "", err
			}
			proposalCount = len(proposals)
			a.samples.Add(prefix+"proposal_compute", time.Since(computeStart))
			a.samples.Add(prefix+"proposal_rebuild_dryrun", time.Since(start))
			persistStart := time.Now()
			if err := configured.RebuildProposals(ctx, benchOrg, project, evaluator); err != nil {
				return "", err
			}
			// Conservative diagnostic: the explicit persistence operation repeats
			// input reads and reconciliation in a second transaction. Exclude the
			// preceding dry evaluation from this separate timing.
			a.samples.Add(prefix+"proposal_rebuild_persisted", rebuildElapsed+time.Since(persistStart))
		}
		body, _ := json.Marshal(map[string]string{"value": []string{"house", "warehouse"}[i%2]})
		if _, err := a.timed(ctx, editPath, http.MethodPut, "/projects/"+project+"/profile/hdr.subclass", body, http.StatusOK); err != nil {
			return "", err
		}
	}
	note := fmt.Sprintf("%s fixture SHA-256 %x: %d documents, %d facts, %d passages; %d full rebuilds and edits.", label, sha256.Sum256(raw), len(tables["documents"]), len(tables["profile_facts"]), len(tables["passages"]), n)
	if manual0991 {
		note += " Bounded two-drawing baseline, manually proposed work items and zero machine-read facts; not an extraction score or complete corpus replay."
	}
	if integrated {
		note += fmt.Sprintf(" Integrated proposal diagnostic: %d loaded records, %d proposals on last rebuild. Profile rebuild samples include proposal evaluation and persistence in the same transaction, reusing inputs. Profile edits remain ordinary and do not establish the integrated edit budget. Not release evidence.", len(cat.InterfaceConsequences())+len(cat.Consequences())+len(cat.UnforeseenConditions()), proposalCount)
	} else if measureProposals {
		note += fmt.Sprintf(" Proposal diagnostic: %d loaded records, %d proposals on last rebuild. Dryrun excludes writes; persisted timing adds explicit transactional projection replacement to the profile rebuild, with repeated input reads and reconciliation. Neither measures integrated edit wiring or constitutes step-0 completion or release evidence.", len(cat.InterfaceConsequences())+len(cat.Consequences())+len(cat.UnforeseenConditions()), proposalCount)
	}
	return note, nil
}

type manualWorkFixture struct {
	Kind      string                                  `json:"kind"`
	Project   string                                  `json:"project"`
	KeySHA256 string                                  `json:"key_sha256"`
	Corpus    map[string]string                       `json:"corpus"`
	Notes     string                                  `json:"notes"`
	Tables    map[string][]map[string]json.RawMessage `json:"tables"`
}

func load0991Benchmark(path string) (map[string][]map[string]json.RawMessage, error) {
	var fixture manualWorkFixture
	if err := eval.ReadStrictJSON(path, &fixture); err != nil {
		return nil, err
	}
	key, hash, err := eval.LoadWorkKey("data/eval/profile/answer-keys/0991.yaml")
	if err != nil {
		return nil, err
	}
	m, err := eval.LoadProfileManifest("data/eval/profile/manifest.json")
	if err != nil {
		return nil, err
	}
	corpus, err := eval.VerifyWorkCorpus(m, key, "")
	if err != nil {
		return nil, err
	}
	if err := validateManualWorkFixture(fixture, key, hash, corpus); err != nil {
		return nil, err
	}
	return fixture.Tables, nil
}

func validateManualWorkFixture(fixture manualWorkFixture, key eval.WorkKey, hash string, corpus map[string]string) error {
	allowed := map[string]bool{"sites": true, "project_parts": true, "files": true, "documents": true, "passages": true, "profile_facts": true, "work_items": true}
	for table := range fixture.Tables {
		if !allowed[table] {
			return fmt.Errorf("unexpected table in manual timing fixture")
		}
	}
	if fixture.Kind != "manual-source-timing" || fixture.Project != "0991" || fixture.KeySHA256 != hash || len(fixture.Corpus) != len(corpus) {
		return fmt.Errorf("invalid or stale 0991 fixture metadata")
	}
	for id, hash := range corpus {
		if fixture.Corpus[id] != hash {
			return fmt.Errorf("0991 fixture source mismatch")
		}
	}
	if len(fixture.Tables["documents"]) != len(corpus) || len(fixture.Tables["sites"]) != 1 || len(fixture.Tables["passages"]) < len(corpus) || len(fixture.Tables["profile_facts"]) != 0 || len(fixture.Tables["work_items"]) != len(key.Items) {
		return fmt.Errorf("0991 manual fixture workload mismatch")
	}
	stringField := func(row map[string]json.RawMessage, key string) string {
		var value string
		_ = json.Unmarshal(row[key], &value)
		return value
	}
	expectedHashes := map[string]bool{}
	for _, value := range corpus {
		expectedHashes["\\x"+value] = true
	}
	files := map[string]string{}
	for _, file := range fixture.Tables["files"] {
		id, digest := stringField(file, "id"), stringField(file, "sha256")
		if id == "" || files[id] != "" || !expectedHashes[digest] {
			return fmt.Errorf("0991 fixture file/source mismatch")
		}
		files[id] = digest
		delete(expectedHashes, digest)
	}
	if len(expectedHashes) != 0 {
		return fmt.Errorf("0991 fixture missing source files")
	}
	documents := map[string]bool{}
	for _, doc := range fixture.Tables["documents"] {
		id, file := stringField(doc, "id"), stringField(doc, "file_id")
		if id == "" || documents[id] || files[file] == "" {
			return fmt.Errorf("0991 fixture document/file mismatch")
		}
		documents[id] = true
		delete(files, file)
	}
	seenText := map[string]bool{}
	for _, passage := range fixture.Tables["passages"] {
		if !documents[stringField(passage, "document_id")] || strings.TrimSpace(stringField(passage, "body")) == "" {
			return fmt.Errorf("0991 fixture passage/document mismatch")
		}
		seenText[stringField(passage, "document_id")] = true
	}
	if len(seenText) != len(documents) {
		return fmt.Errorf("0991 fixture document has no text")
	}
	actual := eval.WorkSnapshot{Project: key.Project}
	decode := func(name string, target any) error {
		b, err := json.Marshal(fixture.Tables[name])
		if err != nil {
			return err
		}
		return json.Unmarshal(b, target)
	}
	if err := decode("project_parts", &actual.Parts); err != nil {
		return err
	}
	if err := decode("work_items", &actual.Items); err != nil {
		return err
	}
	score, err := eval.ScoreWorkItems(key, actual)
	if err != nil {
		return err
	}
	if score.FP != 0 || score.FN != 0 {
		return fmt.Errorf("0991 manual timing annotations differ from pinned draft")
	}
	return nil
}
