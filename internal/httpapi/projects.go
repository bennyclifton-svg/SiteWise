package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/events"
	"sitewise/internal/intake"
	"sitewise/internal/store"
)

// Deps is the HTTP wiring. Store and PublicOrigin serve sessions and
// projects; the intake fields serve uploads, corrections and events.
type Deps struct {
	Store          *store.Store
	PublicOrigin   string
	Log            *log.Logger
	MaxBodyBytes   int64
	SecureCookie   bool
	MaxUploadBytes int64
	Uploader       *intake.Uploader
	Service        *intake.Service
	Catalog        intake.Catalog
	Broker         *events.Broker
	Filer          Filer
	// Closing ends event streams when the server shuts down.
	Closing <-chan struct{}
}

// Filer starts a foreground filing that outlives the upload request.
type Filer interface {
	Start(orgID, documentID string)
}

// Handler serves session consumption and project routes.
func Handler(deps Deps) http.Handler {
	if deps.MaxBodyBytes <= 0 {
		deps.MaxBodyBytes = 1 << 20
	}
	if deps.Log == nil {
		deps.Log = log.New(io.Discard, "", 0)
	}
	if deps.MaxUploadBytes <= 0 {
		deps.MaxUploadBytes = 200 << 20
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /session", func(w http.ResponseWriter, r *http.Request) {
		consumeSession(w, r, deps)
	})
	mux.HandleFunc("POST /projects", func(w http.ResponseWriter, r *http.Request) {
		createProject(w, r, deps)
	})
	mux.HandleFunc("GET /projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		getProject(w, r, deps)
	})
	routes := map[string]func(http.ResponseWriter, *http.Request, Deps){
		"GET /session":                       checkSession,
		"GET /projects":                      listProjects,
		"GET /projects/{id}/documents":       listDocuments,
		"POST /projects/{id}/files":          uploadFile,
		"GET /documents/{id}":                getDocument,
		"POST /documents/{id}/filing":        retryFiling,
		"PUT /documents/{id}/fields/{field}": correctField,
		"GET /catalog":                       getCatalog,
		"GET /events":                        streamEvents,
	}
	for pattern, h := range routes {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) { h(w, r, deps) })
	}
	return mux
}

func consumeSession(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	session, err := deps.Store.ConsumeInvite(r.Context(), auth.HashToken(body.Token), time.Now())
	if err != nil {
		deps.Log.Printf("invite rejected")
		http.Error(w, "invite rejected", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   deps.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func createProject(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	id := newProjectID()
	if err := deps.Store.CreateProject(r.Context(), session.OrgID, id, body.Name); err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func getProject(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project, err := deps.Store.GetProject(r.Context(), session.OrgID, r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(project)
}

func memberSession(w http.ResponseWriter, r *http.Request, deps Deps) (store.Session, bool) {
	cookie, err := r.Cookie(auth.SessionCookie)
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return store.Session{}, false
	}
	session, expires, err := deps.Store.LookupSession(r.Context(), cookie.Value)
	if err != nil || !expires.After(time.Now()) {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return store.Session{}, false
	}
	member, err := deps.Store.Member(r.Context(), session.OrgID, session.UserID)
	if err != nil {
		http.Error(w, "membership failed", http.StatusInternalServerError)
		return store.Session{}, false
	}
	if !member {
		http.Error(w, "forbidden", http.StatusForbidden)
		return store.Session{}, false
	}
	return session, true
}

func originOK(r *http.Request, publicOrigin string) bool {
	return r.Header.Get("Origin") == publicOrigin
}

func readJSON(w http.ResponseWriter, r *http.Request, limit int64, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(dest)
}

func newProjectID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
