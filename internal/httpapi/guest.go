package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/files"
	"sitewise/internal/intake"
	"sitewise/internal/store"
)

// Guest routes (docs/plans/2026-11-17-guest-access.md, Task 5). A person
// outside the org holds a link to one shared thing. These are the only
// routes that read the guest cookie, and they read nothing else: no member
// route reads it, and no guest route reads the session cookie. None calls
// Jev, the LLM or chat; a firm's quote takes the ordinary filing and
// proposal read, exactly as a PM drop on that firm's cell does.

// GuestCookie holds the link's token, scoped to the guest API path.
const GuestCookie = "sitewise_guest"

const guestPath = "/api/guest"

// msgGuestGone is the one answer for a wrong, expired, revoked or paused
// link: no oracle for which.
const msgGuestGone = "This link has expired or been withdrawn. Ask the person who sent it for a new one."

const (
	pathGuestOpen = "guest_open"
	// Exchanges per client address; misses on any guest route count too.
	guestPerIP     = 30
	guestReadsHour = 300
	// Quotes per link an hour (security review M3; the lifetime cap is
	// store.GuestQuoteLifetime).
	guestQuoteHour = 5
	guestMaxQuote  = 50 << 20
)

var guestRoutes = map[string]func(http.ResponseWriter, *http.Request, Deps){
	"POST /guest/session":              postGuestSession,
	"GET /guest":                       getGuestPage,
	"GET /guest/documents/{doc}/file":  getGuestDocument,
	"GET /guest/issues/{version}/file": getGuestIssue,
	"POST /guest/quote":                postGuestQuote,
}

func init() {
	routePaths["POST /guest/session"] = pathGuestOpen
	routePaths["GET /guest"] = pathGuestOpen
}

// A token is 64 lowercase hex characters (auth.NewToken).
var tokenPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func guestHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "private, no-store")
	h.Set("X-Robots-Tag", "noindex")
}

// guestCookie is the guest cookie: script cannot read it, it is never sent
// cross-site, only the guest routes see it, and in production (SecureCookie,
// set from SITEWISE_ENV=production) only over HTTPS.
func guestCookie(deps Deps, token string, expires time.Time) *http.Cookie {
	return &http.Cookie{Name: GuestCookie, Value: token, Path: guestPath, Expires: expires, HttpOnly: true, Secure: deps.SecureCookie, SameSite: http.SameSiteStrictMode}
}

func clearGuestCookie(w http.ResponseWriter, deps Deps) {
	c := guestCookie(deps, "", time.Time{})
	c.MaxAge = -1
	http.SetCookie(w, c)
}

func guestGone(w http.ResponseWriter, deps Deps) {
	clearGuestCookie(w, deps)
	http.Error(w, msgGuestGone, http.StatusUnauthorized)
}

// guestMissKey counts failed link checks per address, on every guest route,
// so a guessed token sent as a cookie is limited as an exchange is.
func guestMissKey(r *http.Request) string { return "guest-miss:" + clientIP(r) }

// guestMiss answers a token that found no live link, and counts it against
// the address. Only misses count and only misses are blocked, so a valid
// link behind a shared address keeps working (security review L2).
func guestMiss(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !deps.limits.allow(guestMissKey(r), guestPerIP, limitWindow) {
		tooMany(w)
		return
	}
	guestGone(w, deps)
}

// msgOtherLink: the cookie now holds another link than the one this page
// was opened on (a second link opened in another tab).
const msgOtherLink = "This page is for another link. Reload."

// guestShareHeader carries the link id the page was opened on.
const guestShareHeader = "X-Guest-Share"

// sameShare refuses a download or upload made by a page opened on another
// link than the cookie's (security review M1). Downloads are plain links,
// so they carry it as ?share=; an upload sends the header.
func sameShare(w http.ResponseWriter, r *http.Request, g store.GuestGrant) bool {
	share := r.Header.Get(guestShareHeader)
	if share == "" {
		share = r.URL.Query().Get("share")
	}
	if !strings.EqualFold(share, g.ID) {
		http.Error(w, msgOtherLink, http.StatusConflict)
		return false
	}
	return true
}

// guestGrant is the one check every guest route passes. It reads only the
// guest cookie, hashes it and finds the live grant; the org, the project
// and the shared thing come from the grant, never from the request. A
// write also needs this origin. A miss clears the cookie.
func guestGrant(w http.ResponseWriter, r *http.Request, deps Deps, write bool) (store.GuestGrant, bool) {
	guestHeaders(w)
	if write && !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return store.GuestGrant{}, false
	}
	cookie, err := r.Cookie(GuestCookie)
	if err != nil || !tokenPattern.MatchString(cookie.Value) {
		guestMiss(w, r, deps)
		return store.GuestGrant{}, false
	}
	g, err := deps.Store.GuestGrantByToken(r.Context(), auth.HashToken(cookie.Value), time.Now())
	if errors.Is(err, store.ErrGuestGone) {
		guestMiss(w, r, deps)
		return store.GuestGrant{}, false
	}
	if err != nil {
		deps.Log.Printf("guest link check failed: %v", err)
		http.Error(w, "read failed", http.StatusInternalServerError)
		return store.GuestGrant{}, false
	}
	if !deps.limits.allow("guest-read:"+g.ID, guestReadsHour, time.Hour) {
		tooMany(w)
		return store.GuestGrant{}, false
	}
	return g, true
}

// postGuestSession exchanges the token from the link's fragment for the
// guest cookie. The token never appears in a URL the server sees, in a log
// or in an answer.
func postGuestSession(w http.ResponseWriter, r *http.Request, deps Deps) {
	guestHeaders(w)
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := readJSON(w, r, 4096, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(body.Token)
	if !tokenPattern.MatchString(token) {
		guestMiss(w, r, deps)
		return
	}
	g, err := deps.Store.GuestGrantByToken(r.Context(), auth.HashToken(token), time.Now())
	if errors.Is(err, store.ErrGuestGone) {
		guestMiss(w, r, deps)
		return
	}
	if err != nil {
		deps.Log.Printf("guest link exchange failed: %v", err)
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	if err := deps.Store.TouchGuestGrant(r.Context(), g, true, time.Now()); err != nil {
		deps.Log.Printf("guest open not recorded share=%s: %v", g.ID, err)
	} else if deps.Broker != nil && g.Kind == store.GuestPackage {
		deps.Broker.Wake(g.OrgID)
	}
	http.SetCookie(w, guestCookie(deps, token, g.ExpiresAt))
	w.WriteHeader(http.StatusNoContent)
}

func getGuestPage(w http.ResponseWriter, r *http.Request, deps Deps) {
	g, ok := guestGrant(w, r, deps, false)
	if !ok {
		return
	}
	page, err := deps.Store.GuestPage(r.Context(), g)
	if errors.Is(err, store.ErrGuestGone) {
		guestGone(w, deps)
		return
	}
	if err != nil {
		deps.Log.Printf("guest page share=%s: %v", g.ID, err)
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	if err := deps.Store.TouchGuestGrant(r.Context(), g, false, time.Now()); err != nil {
		deps.Log.Printf("guest open not recorded share=%s: %v", g.ID, err)
	}
	writeJSON(w, http.StatusOK, page)
}

func getGuestDocument(w http.ResponseWriter, r *http.Request, deps Deps) {
	g, ok := guestGrant(w, r, deps, false)
	if !ok || !sameShare(w, r, g) {
		return
	}
	f, err := deps.Store.GuestDocumentFile(r.Context(), g, r.PathValue("doc"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		deps.Log.Printf("guest download share=%s: %v", g.ID, err)
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	serveFile(w, r, deps, f.SHA256, f.MediaType, f.Filename)
}

func getGuestIssue(w http.ResponseWriter, r *http.Request, deps Deps) {
	g, ok := guestGrant(w, r, deps, false)
	if !ok || !sameShare(w, r, g) {
		return
	}
	issue, err := deps.Store.GuestIssueFile(r.Context(), g, r.PathValue("version"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		deps.Log.Printf("guest issue download share=%s: %v", g.ID, err)
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	serveFile(w, r, deps, issue.ExportSHA256, "application/pdf", fmt.Sprintf("Issue %d.pdf", issue.Number))
}

// serveFile streams stored bytes as an attachment, never cached. A missing
// blob is 404. It is the writer every guest download uses; the member
// downloads keep theirs (documents.go has an inline view of its own).
func serveFile(w http.ResponseWriter, r *http.Request, deps Deps, sum []byte, mediaType, filename string) {
	if deps.Blobs == nil {
		http.Error(w, "file unavailable", http.StatusServiceUnavailable)
		return
	}
	f, err := deps.Blobs.Open(sum)
	if errors.Is(err, files.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "file unavailable", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = io.Copy(w, f)
}

// msgQuoteLimit: a link's lifetime of quotes is used up.
var msgQuoteLimit = fmt.Sprintf("This link has taken its %d quotes. Ask the person who sent it for a new link.", store.GuestQuoteLifetime)

// guestReserve is the share of the org's monthly allowance a guest's upload
// never takes.
const guestReserve = 0.10

// guestQuoteTypes are what a firm may send (owner decision 7's default).
var guestQuoteTypes = map[string]bool{".pdf": true, ".docx": true, ".xlsx": true}

// postGuestQuote takes a firm's quote on its own invited cell. The route
// names no package and no firm: both come from the grant. The type is
// checked by name before a byte is read; the grant is checked again under
// the project lock after the bytes land, and a document this request made
// is deleted before filing when the link went in between, so no Jev call
// is made for it.
func postGuestQuote(w http.ResponseWriter, r *http.Request, deps Deps) {
	g, ok := guestGrant(w, r, deps, true)
	if !ok || !sameShare(w, r, g) {
		return
	}
	if g.Kind != store.GuestPackage {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if !guestQuoteTypes[strings.ToLower(path.Ext(name))] {
		http.Error(w, "Send a PDF, Word or Excel file.", http.StatusBadRequest)
		return
	}
	if !deps.limits.allow("guest-quote:"+g.ID, guestQuoteHour, time.Hour) {
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "That is a lot of files for one link. Wait an hour and try again.", http.StatusTooManyRequests)
		return
	}
	if placed, err := deps.Store.GuestQuotesPlaced(r.Context(), g); err != nil {
		deps.Log.Printf("guest quote count share=%s: %v", g.ID, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	} else if placed >= store.GuestQuoteLifetime {
		http.Error(w, msgQuoteLimit, http.StatusConflict)
		return
	}
	// A guest never takes the org's last tenth of its monthly allowance, so
	// a link can never push the org's own PMs to the plan cap (M3).
	if !orgUploadAllowed(w, r, deps, g.OrgID, guestReserve, "The project team's upload allowance is used up. Let the person who sent this link know.") {
		return
	}
	limit := int64(guestMaxQuote)
	if deps.MaxUploadBytes < limit {
		limit = deps.MaxUploadBytes
	}
	body := http.MaxBytesReader(w, r.Body, limit)
	defer body.Close()
	// The upload lands held (not filed, no intake job): nothing can claim
	// it or send it to Jev until OpenGuestQuote has rechecked the link and
	// released it in the transaction that places it.
	reason := intake.ReasonFor(name)
	if reason == "" {
		reason = store.GuestHoldReason
	}
	filing, err := deps.Uploader.Upload(r.Context(), g.OrgID, g.ProjectID, intake.Upload{Filename: name, Body: body, Reason: reason})
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, files.ErrTooLarge):
		http.Error(w, fmt.Sprintf("The file is larger than %d MB.", limit>>20), http.StatusRequestEntityTooLarge)
		return
	case err != nil && r.Context().Err() != nil:
		return
	case errors.Is(err, intake.ErrFilename):
		http.Error(w, "Send a PDF, Word or Excel file.", http.StatusBadRequest)
		return
	case err != nil:
		deps.Log.Printf("guest quote upload failed share=%s: %v", g.ID, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	q, err := deps.Store.OpenGuestQuote(r.Context(), g, filing.DocumentID, filing.Created, time.Now())
	if err != nil {
		if filing.Created {
			dropGuestUpload(deps, g, filing.DocumentID)
		}
		switch {
		case errors.Is(err, store.ErrGuestGone):
			guestGone(w, deps)
		case errors.Is(err, store.ErrGuestQuoteLimit):
			http.Error(w, msgQuoteLimit, http.StatusConflict)
		case errors.Is(err, store.ErrGuestAwarded):
			http.Error(w, "This tender has been awarded.", http.StatusConflict)
		case errors.Is(err, store.ErrConflict):
			http.Error(w, "The project team already has this file.", http.StatusConflict)
		default:
			deps.Log.Printf("guest quote share=%s: %v", g.ID, err)
			http.Error(w, "upload failed", http.StatusInternalServerError)
		}
		return
	}
	if filing.Created {
		countDocument(deps, g.OrgID)
		// As uploadSubmission: new bytes, released from hold, are filed first
		// and the filing hook reads the proposal; anything else is read at once.
		if q.Queued {
			deps.Filer.Start(g.OrgID, filing.DocumentID)
		} else if deps.ProposalReader != nil {
			deps.ProposalReader.Start(g.OrgID, q.SubmissionID, time.Now(), true)
		}
	}
	if deps.Broker != nil {
		deps.Broker.Wake(g.OrgID)
	}
	writeJSON(w, http.StatusCreated, q)
}

// dropGuestUpload deletes a document a refused quote made, before anything
// files it. Its bytes go when nothing else references them.
func dropGuestUpload(deps Deps, g store.GuestGrant, document string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := deps.Store.DeleteDocuments(ctx, g.OrgID, g.ProjectID, []string{document}, g.CreatedBy)
	if err != nil {
		deps.Log.Printf("guest quote: refused upload not removed share=%s: %v", g.ID, err)
		return
	}
	for _, sum := range res.Unreferenced {
		if deps.Blobs == nil {
			break
		}
		if _, err := deps.Blobs.Remove(sum, blobMinAge, func() (bool, error) { return deps.Store.BlobReferenced(ctx, sum) }); err != nil {
			deps.Log.Printf("guest quote: stored file kept: %v", err)
		}
	}
}
