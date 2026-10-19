package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"sitewise/internal/events"
	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/programread"
	"sitewise/internal/store"
)

// Reading A: a program file dropped on Program (plan
// docs/plans/2026-10-18-program.md section 10.2). The upload is filed by
// the normal intake path, unchanged: same uploader, same filing kinds, same
// single filing fan-out. Filing's AfterFile hook then starts the
// program_import read outside the request, at background priority, with at
// most one Jev fan-out per file. Its matches are proposals; nothing here
// changes the program until the user accepts.

const (
	pathProgramImport = "program_import"

	// eventProgramReadings tells open views the readings changed: a read
	// landed, so the toolbar and readings panel refetch.
	eventProgramReadings = "program.readings"
)

// programImporter reads imported program files: one goroutine per import,
// bounded by its own slots, like proposalReader.
type programImporter struct {
	store      *store.Store
	blobs      *files.Store
	ask        programread.Asker
	thresholds programread.Thresholds
	broker     *events.Broker
	log        *log.Logger
	observe    func(string, time.Duration)
	timeout    time.Duration
	slots      chan struct{}
	now        func() time.Time
	wg         sync.WaitGroup

	mu       sync.Mutex
	inflight map[string]struct{}
}

// Filed is the filing hook: a document dropped on Program starts its read.
// Any other document costs one indexed lookup. It returns at once.
func (r *programImporter) Filed(orgID, documentID string, _ identity.Text) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		refs, err := r.store.ClaimProgramImports(ctx, orgID, documentID)
		if err != nil {
			r.log.Printf("program import lookup org=%s doc=%s: %v", orgID, documentID, err)
			return
		}
		for _, ref := range refs {
			r.Start(ref, false)
		}
	}()
}

// Start reads one claimed import unless it is already being read.
func (r *programImporter) Start(ref store.ProgramImportRef, recheck bool) {
	key := ref.OrgID + "/" + ref.ImportID
	r.mu.Lock()
	if _, busy := r.inflight[key]; busy {
		r.mu.Unlock()
		return
	}
	r.inflight[key] = struct{}{}
	r.mu.Unlock()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			r.mu.Lock()
			delete(r.inflight, key)
			r.mu.Unlock()
		}()
		r.slots <- struct{}{}
		defer func() { <-r.slots }()
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()
		done, committed, err := r.read(ctx, ref.OrgID, ref.ImportID, recheck)
		if err != nil {
			r.log.Printf("program import stopped org=%s import=%s: %v", ref.OrgID, ref.ImportID, err)
			return
		}
		if !committed {
			return
		}
		if r.broker != nil {
			payload, _ := json.Marshal(map[string]any{"project_id": done.ProjectID, "program_id": done.ProgramID, "document_id": done.DocumentID, "readings_pending": done.ReadingsPending})
			if _, err := r.broker.Commit(ctx, ref.OrgID, eventProgramReadings, "", string(payload)); err != nil {
				r.log.Printf("program readings event: %v", err)
			}
		}
		if !recheck && !ref.Began.IsZero() && r.observe != nil {
			r.observe(pathProgramImport, r.now().Sub(ref.Began))
		}
	}()
}

func (r *programImporter) read(ctx context.Context, org, id string, recheck bool) (store.ProgramImportCommitted, bool, error) {
	in, err := r.store.ProgramImportForReading(ctx, org, id)
	if err != nil {
		return store.ProgramImportCommitted{}, false, err
	}
	out, notRead, err := r.outcome(ctx, in)
	if err != nil {
		return store.ProgramImportCommitted{}, false, err
	}
	return r.store.CommitProgramImport(ctx, org, id, out, notRead, recheck, r.now())
}

// Reasons a file is not read.
const (
	importNoText      = "no_text_layer"
	importUnreadable  = "unreadable_text"
	importUnsupported = "unsupported_format"
	importNoRows      = "no_program_rows"
)

func (r *programImporter) outcome(ctx context.Context, in store.ProgramImportInput) (programread.Outcome, string, error) {
	format := proposalFormat(in.Filename)
	if format == "" {
		return programread.Outcome{}, importUnsupported, nil
	}
	path, err := r.blobs.Path(in.SHA256)
	if err != nil {
		return programread.Outcome{}, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return programread.Outcome{}, "", err
	}
	text, err := identity.Extract(ctx, format, f, in.Size, programread.ExtractLimits())
	f.Close()
	switch {
	case errors.Is(err, identity.ErrTooLarge), errors.Is(err, identity.ErrMalformed):
		return programread.Outcome{}, importUnreadable, nil
	case err != nil:
		return programread.Outcome{}, "", err
	case format == "pdf" && !text.TextLayer:
		return programread.Outcome{}, importNoText, nil
	}
	h := programread.HarvestText(in.Filename, text, in.PerWeek)
	if len(h.Rows) == 0 {
		return programread.Outcome{}, importNoRows, nil
	}
	// One background fan-out at most, inside programread.Read.
	// https://docs.typesafe.ai/patterns/fan-out
	out, err := programread.Read(ctx, r.ask, r.thresholds, h, in.Targets)
	return out, "", err
}

// RunProgramImportRechecks re-asks grey program reads in the background
// with the proposal read's spacing: 30 seconds, at most three times.
func (s *Server) RunProgramImportRechecks(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		now := time.Now()
		due, err := s.store.DueProgramImportRechecks(ctx, now)
		if err != nil {
			continue
		}
		for _, ref := range due {
			if ok, err := s.store.ClaimProgramImportRecheck(ctx, ref, now); err == nil && ok {
				s.imports.Start(store.ProgramImportRef{OrgID: ref.OrgID, ImportID: ref.ImportID}, true)
			}
		}
	}
}

// ProgramImporter starts a read for a file already filed.
type ProgramImporter interface {
	Start(ref store.ProgramImportRef, recheck bool)
}

// postProgramImport is POST /program/import: a program file as a multipart
// upload (field "file"), or the raw body with ?name= like every other
// upload. It files through the intake uploader unchanged, marks the
// document for the program read, and returns 202: the readings arrive as a
// program.readings event.
func postProgramImport(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if _, err := d.Store.LoadProgram(r.Context(), s.OrgID, project); errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]bool{"empty": true})
		return
	} else if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	began := time.Now()
	body := http.MaxBytesReader(w, r.Body, d.MaxUploadBytes)
	defer body.Close()
	name, file, err := importBody(r, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filing, err := d.Uploader.Upload(r.Context(), s.OrgID, project, intake.Upload{Filename: name, Body: file, Reason: intake.ReasonFor(name)})
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, files.ErrTooLarge):
		http.Error(w, "file is larger than the upload limit", http.StatusRequestEntityTooLarge)
		return
	case err != nil && r.Context().Err() != nil:
		return
	case errors.Is(err, intake.ErrFilename):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case err != nil:
		d.Log.Printf("program import upload failed org=%s: %v", s.OrgID, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	id, err := d.Store.OpenProgramImport(r.Context(), s.OrgID, project, filing.DocumentID)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	if filing.Status == store.StatusPending {
		d.Filer.Start(s.OrgID, filing.DocumentID)
	} else if d.ProgramImporter != nil {
		// Bytes already filed: no filing runs, so read now.
		refs, err := d.Store.ClaimProgramImport(r.Context(), s.OrgID, id)
		if err != nil {
			programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
			return
		}
		for _, ref := range refs {
			ref.Began = began
			d.ProgramImporter.Start(ref, false)
		}
	}
	if d.Broker != nil {
		d.Broker.Wake(s.OrgID)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"import_id": id, "document_id": filing.DocumentID})
}

// importBody is the uploaded file and its name: the first file part of a
// multipart form, or the raw body named by ?name=.
func importBody(r *http.Request, body io.Reader) (string, io.Reader, error) {
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if media != "multipart/form-data" {
		return strings.TrimSpace(r.URL.Query().Get("name")), body, nil
	}
	r.Body = io.NopCloser(body)
	mr, err := r.MultipartReader()
	if err != nil {
		return "", nil, errors.New("invalid multipart body")
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			return "", nil, errors.New("no file in the upload")
		}
		if part.FileName() != "" {
			return strings.TrimSpace(part.FileName()), part, nil
		}
	}
}

// postProgramReadingsDismiss is POST /program/readings/dismiss
// {"version","ids"}: one undoable batch.
func postProgramReadingsDismiss(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		Version int64    `json:"version"`
		IDs     []string `json:"ids"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Version < 1 {
		programInvalid(w, "version", "The program version is required.")
		return
	}
	for _, id := range body.IDs {
		if !uuidPattern.MatchString(id) {
			programInvalid(w, "ids", store.MsgNotInProgram)
			return
		}
	}
	res, err := d.Store.DismissProgramReadings(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, body.Version, body.IDs)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	writeProgramResult(w, r, d, s.OrgID, res)
}
