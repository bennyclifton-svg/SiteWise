package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/knowledge"
	"sitewise/internal/store"
)

// Program workload sizes (plan docs/plans/2026-10-18-program.md 11.1).
const (
	programBenchPackages  = 30
	programBenchSystems   = 40
	programBenchStages    = 20
	programBenchPerStage  = 49 // 20 stages + 20×49 activities = 1,000 rows
	programBenchLinkCount = 2000
)

// program measures program_generate (prepare on a project with 30
// packages and 40 systems of work), program_read (GET with 1,000 activities
// and 2,000 links), program_write (one move) and program_export (Excel of
// the same 1,000 rows). Seeding is untimed.
func (a *apiClient) program(ctx context.Context, st *store.Store, cat *knowledge.Catalog, dsn string, n int) error {
	if err := a.programGenerate(ctx, cat, dsn, n); err != nil {
		return fmt.Errorf("program generate: %w", err)
	}
	if err := a.programReadWrite(ctx, st, n); err != nil {
		return fmt.Errorf("program read and write: %w", err)
	}
	return nil
}

func (a *apiClient) programGenerate(ctx context.Context, cat *knowledge.Catalog, dsn string, n int) error {
	project, err := a.createProject(ctx, "Program generation benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project
	var profile struct {
		Parts []struct {
			ID string `json:"id"`
		} `json:"parts"`
	}
	raw, err := a.call(ctx, http.MethodGet, base+"/profile", nil, http.StatusOK)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &profile); err != nil || len(profile.Parts) == 0 {
		return fmt.Errorf("profile parts: %v", err)
	}
	made := 0
	for _, sys := range cat.Leaves() {
		if made == programBenchSystems {
			break
		}
		body, _ := json.Marshal(map[string]string{"part_id": profile.Parts[0].ID, "system_id": sys.ID, "action": "new", "title": sys.Label})
		resp, err := a.do(ctx, http.MethodPost, base+"/works", bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			made++
		}
	}
	if made != programBenchSystems {
		return fmt.Errorf("seeded %d of %d systems of work", made, programBenchSystems)
	}
	stages := 0
	for i := 0; i < programBenchPackages; i++ {
		body, _ := json.Marshal(map[string]string{"kind": "services", "title": fmt.Sprintf("Consultant %02d", i)})
		raw, err := a.call(ctx, http.MethodPost, base+"/packages", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var p struct {
			Stages []struct{} `json:"stages"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		stages += len(p.Stages)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	for i := 0; i < n; i++ {
		// Each sample creates the program afresh: the package stages are
		// unlinked and the program removed outside the timing.
		if _, err := conn.Exec(ctx, `UPDATE package_stages SET program_stage_id=NULL,stage_revision=NULL WHERE org_id=$1 AND project_id=$2`, benchOrg, project); err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, `DELETE FROM programs WHERE org_id=$1 AND project_id=$2`, benchOrg, project); err != nil {
			return err
		}
		raw, err := a.timed(ctx, "program_generate", http.MethodPost, base+"/program/prepare", []byte(`{"surface":"program"}`), http.StatusOK)
		if err != nil {
			return err
		}
		var v struct {
			Created    bool              `json:"created"`
			Activities []json.RawMessage `json:"activities"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		if !v.Created || len(v.Activities) < stages {
			return fmt.Errorf("prepare made %d rows for %d package stages", len(v.Activities), stages)
		}
	}
	return nil
}

// programBenchCreate is 20 stages of 49 activities with 2,000 forward
// links: the stage chain, each stage's activity chain, then activity j of
// one stage to activity j of the next, then start-to-start skips, until
// the count is reached. Every link runs forward, so there is no loop.
func programBenchCreate() store.ProgramCreate {
	in := store.ProgramCreate{Surface: "program", SkeletonID: "sk.generic", LibraryVersion: "bench", StartDate: "2027-03-01", StartOrigin: "user"}
	key := func(s, j int) string { return fmt.Sprintf("a%02d.%02d", s, j) }
	stage := func(s int) string { return fmt.Sprintf("s%02d", s) }
	for s := 0; s < programBenchStages; s++ {
		in.Rows = append(in.Rows, store.ProgramRowInput{Key: stage(s), Kind: "stage", Label: fmt.Sprintf("Stage %d", s+1), CatalogueStageID: "other", ReviewStatus: "proposed", Origin: "library"})
		for j := 0; j < programBenchPerStage; j++ {
			d := 1 + (s*7+j)%10
			in.Rows = append(in.Rows, store.ProgramRowInput{Key: key(s, j), ParentKey: stage(s), Kind: "activity", Label: fmt.Sprintf("Activity %d.%d", s+1, j+1),
				DurationDays: &d, DurationOrigin: "user", ReviewStatus: "proposed", Origin: "library"})
		}
	}
	add := func(pred, succ, typ, origin string) bool {
		if len(in.Links) == programBenchLinkCount {
			return false
		}
		in.Links = append(in.Links, store.ProgramLinkInput{PredKey: pred, SuccKey: succ, Type: typ, Origin: origin})
		return true
	}
	for s := 1; s < programBenchStages; s++ {
		add(stage(s-1), stage(s), "FS", "chain")
	}
	for s := 0; s < programBenchStages; s++ {
		for j := 1; j < programBenchPerStage; j++ {
			add(key(s, j-1), key(s, j), "FS", "chain")
		}
	}
	for s := 1; s < programBenchStages; s++ {
		for j := 0; j < programBenchPerStage; j++ {
			add(key(s-1, j), key(s, j), "FS", "user")
		}
	}
	for s := 0; s < programBenchStages; s++ {
		for j := 2; j < programBenchPerStage; j++ {
			if !add(key(s, j-2), key(s, j), "SS", "user") {
				return in
			}
		}
	}
	return in
}

func (a *apiClient) programReadWrite(ctx context.Context, st *store.Store, n int) error {
	project, err := a.createProject(ctx, "Program read and write benchmark")
	if err != nil {
		return err
	}
	in := programBenchCreate()
	if len(in.Rows) != 1000 || len(in.Links) != programBenchLinkCount {
		return fmt.Errorf("bench program has %d rows and %d links", len(in.Rows), len(in.Links))
	}
	created, err := st.CreateProgram(ctx, benchOrg, project, in)
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/program"
	var version int64 = 1
	moved := created.IDs[fmt.Sprintf("a%02d.%02d", programBenchStages-1, programBenchPerStage-1)]
	for i := 0; i < n; i++ {
		raw, err := a.timed(ctx, "program_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var v struct {
			Program struct {
				Version int64 `json:"version"`
			} `json:"program"`
			Activities []json.RawMessage `json:"activities"`
			Links      []json.RawMessage `json:"links"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		if len(v.Activities) != 1000 || len(v.Links) != programBenchLinkCount || v.Program.Version != version {
			return fmt.Errorf("read %d rows, %d links, version %d", len(v.Activities), len(v.Links), v.Program.Version)
		}
		// Excel export of the same 1,000 rows, expanded (program_export).
		xl, err := a.timed(ctx, "program_export", http.MethodGet, base+"/export?format=xlsx&view=expanded", nil, http.StatusOK)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(xl, []byte("PK")) {
			return fmt.Errorf("program export is not a workbook: %.40q", xl)
		}
		// One move: the last activity a working day later each time, so a
		// link never holds it.
		body, _ := json.Marshal(map[string]any{"program_id": created.ProgramID, "version": version, "ops": []map[string]any{{"op": "move", "ids": []string{moved}, "days": 1}}})
		if _, err := a.timed(ctx, "program_write", http.MethodPost, base+"/ops", body, http.StatusOK); err != nil {
			return err
		}
		version++
	}
	return nil
}
