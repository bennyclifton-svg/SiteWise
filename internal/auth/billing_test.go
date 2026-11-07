package auth_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/billing"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/intake"
	"sitewise/internal/store"
)

const webhookSecret = "whsec_billing_test"

// mailbox captures every account mail; sending happens after the response.
type mailbox struct {
	mu    sync.Mutex
	links map[string]string
}

func newMailbox() *mailbox { return &mailbox{links: map[string]string{}} }

func (m *mailbox) put(to, link string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.links[to] = link
	return nil
}
func (m *mailbox) SendReset(_ context.Context, to, link string) error  { return m.put(to, link) }
func (m *mailbox) SendVerify(_ context.Context, to, link string) error { return m.put(to, link) }
func (m *mailbox) SendTeamInvite(_ context.Context, to, _ string, link string) error {
	return m.put(to, link)
}

// token waits for the mail to arrive and returns the fragment's token.
func (m *mailbox) token(t *testing.T, to, key string) string {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		m.mu.Lock()
		link := m.links[to]
		m.mu.Unlock()
		if _, tok, ok := strings.Cut(link, "#"+key+"="); ok {
			return tok
		}
	}
	t.Fatalf("no %s mail for %s", key, to)
	return ""
}

type fakeStripe struct {
	mu   sync.Mutex
	last billing.CheckoutRequest
}

func (f *fakeStripe) Checkout(_ context.Context, r billing.CheckoutRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.last = r
	return "https://checkout.stripe.test/" + r.PriceID, nil
}

func (f *fakeStripe) Portal(_ context.Context, customer, _ string) (string, error) {
	return "https://billing.stripe.test/" + customer, nil
}

type noFiler struct{}

func (noFiler) Start(string, string) {}

func billingHandler(t *testing.T, st *store.Store, logs *bytes.Buffer, mail *mailbox, b *httpapi.Billing) http.Handler {
	t.Helper()
	blobs, err := files.Open(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return httpapi.Handler(httpapi.Deps{
		Store: st, Blobs: blobs, Uploader: intake.NewUploader(blobs, st), Filer: noFiler{},
		PublicOrigin: origin, Log: log.New(logs, "", 0), MaxBodyBytes: 4096,
		ResetMail: mail, Billing: b,
	})
}

func testBilling(stripe *fakeStripe) *httpapi.Billing {
	return &httpapi.Billing{
		Stripe: stripe, WebhookSecret: webhookSecret, TaxRateID: "txr_gst",
		Prices: map[string]string{"starter/month": "price_sm", "starter/year": "price_sa", "team/month": "price_tm", "team/year": "price_ta"},
	}
}

// signUp makes an account and returns its session cookie and org id.
func signUp(t *testing.T, h http.Handler, st *store.Store, email string) (*http.Cookie, string) {
	t.Helper()
	rec := postJSON(t, h, "/signup", nil, creds(email, "correct horse"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("signup %d %s", rec.Code, rec.Body)
	}
	a, err := st.AccountByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	return sessionCookie(t, rec), a.OrgID
}

func upload(t *testing.T, h http.Handler, cookie *http.Cookie, project, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := request(t, http.MethodPost, "/projects/"+project+"/files?name="+name, cookie, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	h.ServeHTTP(rec, req)
	return rec
}

func createProject(t *testing.T, h http.Handler, cookie *http.Cookie, name string) (string, int) {
	t.Helper()
	rec := postJSON(t, h, "/projects", cookie, map[string]string{"name": name})
	var out struct{ ID string }
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return out.ID, rec.Code
}

type account struct {
	Email          string
	Verified       bool `json:"email_verified"`
	Role           string
	BillingEnabled bool `json:"billing_enabled"`
	Plan           billing.Plan
	Usage          store.Usage
	HasCustomer    bool `json:"has_customer"`
}

func getAccount(t *testing.T, h http.Handler, cookie *http.Cookie) account {
	t.Helper()
	rec := getWith(t, h, "/account", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("account %d %s", rec.Code, rec.Body)
	}
	var a account
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestUploadNeedsConfirmedEmailAndCountsOnce(t *testing.T) {
	st, logs := openStore(t)
	mail := newMailbox()
	h := billingHandler(t, st, logs, mail, testBilling(&fakeStripe{}))
	email := uniqueEmail(t, st)
	cookie, _ := signUp(t, h, st, email)
	project, code := createProject(t, h, cookie, "Mine")
	if code != http.StatusCreated {
		t.Fatalf("create project %d", code)
	}
	if got := upload(t, h, cookie, project, "a.txt", "hello"); got.Code != http.StatusForbidden || !strings.Contains(got.Body.String(), "Confirm your email") {
		t.Fatalf("unverified upload %d %s", got.Code, got.Body)
	}
	if a := getAccount(t, h, cookie); a.Verified || a.Role != "owner" || !a.BillingEnabled || a.Plan.ID != billing.Free {
		t.Fatalf("account before verify %+v", a)
	}
	tok := mail.token(t, email, "verify")
	if got := postJSON(t, h, "/email/verify", nil, map[string]string{"token": tok}); got.Code != http.StatusNoContent {
		t.Fatalf("verify %d %s", got.Code, got.Body)
	}
	if got := postJSON(t, h, "/email/verify", nil, map[string]string{"token": tok}); got.Code != http.StatusUnauthorized {
		t.Fatalf("second use of link %d", got.Code)
	}
	if got := upload(t, h, cookie, project, "a.txt", "hello"); got.Code != http.StatusCreated {
		t.Fatalf("verified upload %d %s", got.Code, got.Body)
	}
	// Dropping the same file again files nothing new and is not counted.
	if got := upload(t, h, cookie, project, "a.txt", "hello"); got.Code != http.StatusOK {
		t.Fatalf("re-drop %d %s", got.Code, got.Body)
	}
	if a := getAccount(t, h, cookie); !a.Verified || a.Usage.DocumentsThisMonth != 1 || a.Usage.StorageBytes != 5 {
		t.Fatalf("account after uploads %+v", a)
	}
}

func TestFreePlanLimits(t *testing.T) {
	st, logs := openStore(t)
	h := billingHandler(t, st, logs, newMailbox(), testBilling(&fakeStripe{}))
	email := uniqueEmail(t, st)
	cookie, org := signUp(t, h, st, email)
	ctx := context.Background()
	verifyDirect(t, st, org)
	project, _ := createProject(t, h, cookie, "First")
	if _, code := createProject(t, h, cookie, "Second"); code != http.StatusPaymentRequired {
		t.Fatalf("second free project %d", code)
	}
	for range billing.Lookup(billing.Free).DocumentsPerMonth {
		if err := st.CountDocument(ctx, org, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if got := upload(t, h, cookie, project, "b.txt", "over"); got.Code != http.StatusPaymentRequired || !strings.Contains(got.Body.String(), "50 documents a month") {
		t.Fatalf("over cap %d %s", got.Code, got.Body)
	}
	if err := st.SetPlan(ctx, org, billing.Starter); err != nil {
		t.Fatal(err)
	}
	if _, code := createProject(t, h, cookie, "Second"); code != http.StatusCreated {
		t.Fatalf("starter second project %d", code)
	}
	if got := upload(t, h, cookie, project, "b.txt", "over"); got.Code != http.StatusCreated {
		t.Fatalf("starter upload %d %s", got.Code, got.Body)
	}
}

// verifyDirect marks the org's owner verified without the mail round trip.
func verifyDirect(t *testing.T, st *store.Store, org string) {
	t.Helper()
	raw := strings.Repeat("ab", 32)
	hash := sha256.Sum256([]byte(raw))
	members, _, err := st.Team(context.Background(), org, time.Now())
	if err != nil || len(members) == 0 {
		t.Fatalf("members %v", err)
	}
	if err := st.CreateEmailVerification(context.Background(), org, members[0].UserID, hash[:], time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := st.VerifyEmail(context.Background(), hash[:], time.Now()); err != nil {
		t.Fatal(err)
	}
}

func signedWebhook(t *testing.T, h http.Handler, body string, secret string) *httptest.ResponseRecorder {
	t.Helper()
	ts := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + body))
	rec := httptest.NewRecorder()
	// Stripe sends no Origin and no session.
	req := httptest.NewRequest(http.MethodPost, "/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Stripe-Signature", "t="+ts+",v1="+hex.EncodeToString(mac.Sum(nil)))
	h.ServeHTTP(rec, req)
	return rec
}

func TestCheckoutAndWebhookRaiseThePlan(t *testing.T) {
	st, logs := openStore(t)
	stripe := &fakeStripe{}
	h := billingHandler(t, st, logs, newMailbox(), testBilling(stripe))
	email := uniqueEmail(t, st)
	cookie, org := signUp(t, h, st, email)

	if got := postJSON(t, h, "/billing/checkout", cookie, map[string]string{"plan": "gold", "interval": "month"}); got.Code != http.StatusBadRequest {
		t.Fatalf("unknown plan %d", got.Code)
	}
	rec := postJSON(t, h, "/billing/checkout", cookie, map[string]string{"plan": "team", "interval": "year"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "price_ta") {
		t.Fatalf("checkout %d %s", rec.Code, rec.Body)
	}
	if stripe.last.OrgID != org || stripe.last.Email != email || stripe.last.TaxRateID != "txr_gst" || !strings.HasSuffix(stripe.last.SuccessURL, "/account?checkout=done") {
		t.Fatalf("checkout request %+v", stripe.last)
	}
	if got := postJSON(t, h, "/billing/portal", cookie, nil); got.Code != http.StatusConflict {
		t.Fatalf("portal before customer %d", got.Code)
	}

	n := time.Now().UnixNano()
	event := fmt.Sprintf(`{"id":"evt_%d","type":"customer.subscription.created","created":%d,"data":{"object":{"id":"sub_%d","customer":"cus_%d","status":"active","metadata":{"org_id":%q},"items":{"data":[{"current_period_end":%d,"price":{"id":"price_ta","recurring":{"interval":"year"}}}]}}}}`,
		n, time.Now().Unix(), n, n, org, time.Now().Add(365*24*time.Hour).Unix())
	if got := signedWebhook(t, h, event, "whsec_forged"); got.Code != http.StatusBadRequest {
		t.Fatalf("forged webhook %d", got.Code)
	}
	if a := getAccount(t, h, cookie); a.Plan.ID != billing.Free {
		t.Fatalf("forged webhook changed plan to %s", a.Plan.ID)
	}
	if got := signedWebhook(t, h, event, webhookSecret); got.Code != http.StatusOK {
		t.Fatalf("webhook %d %s", got.Code, got.Body)
	}
	if got := signedWebhook(t, h, event, webhookSecret); got.Code != http.StatusOK {
		t.Fatalf("duplicate webhook %d", got.Code)
	}
	a := getAccount(t, h, cookie)
	if a.Plan.ID != billing.Team || !a.HasCustomer {
		t.Fatalf("after webhook %+v", a)
	}
	if got := postJSON(t, h, "/billing/checkout", cookie, map[string]string{"plan": "starter", "interval": "month"}); got.Code != http.StatusConflict {
		t.Fatalf("second subscription %d", got.Code)
	}
	if got := postJSON(t, h, "/billing/portal", cookie, nil); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "billing.stripe.test/cus_") {
		t.Fatalf("portal %d %s", got.Code, got.Body)
	}
}

func TestBillingOffKeepsDevelopmentUnlimited(t *testing.T) {
	st, logs := openStore(t)
	h := billingHandler(t, st, logs, newMailbox(), nil)
	cookie, _ := signUp(t, h, st, uniqueEmail(t, st))
	for _, name := range []string{"One", "Two"} {
		project, code := createProject(t, h, cookie, name)
		if code != http.StatusCreated {
			t.Fatalf("project %s %d", name, code)
		}
		// No email gate without billing.
		if got := upload(t, h, cookie, project, "c.txt", name); got.Code != http.StatusCreated {
			t.Fatalf("upload %d %s", got.Code, got.Body)
		}
	}
	if got := signedWebhook(t, h, `{"id":"evt_x","type":"x"}`, webhookSecret); got.Code != http.StatusNotFound {
		t.Fatalf("webhook without billing %d", got.Code)
	}
	if got := postJSON(t, h, "/billing/checkout", cookie, map[string]string{"plan": "starter", "interval": "month"}); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("checkout without billing %d", got.Code)
	}
}

func TestTeamInvites(t *testing.T) {
	st, logs := openStore(t)
	mail := newMailbox()
	h := billingHandler(t, st, logs, mail, testBilling(&fakeStripe{}))
	ctx := context.Background()
	ownerEmail := uniqueEmail(t, st)
	owner, org := signUp(t, h, st, ownerEmail)
	project, _ := createProject(t, h, owner, "Shared")

	colleague := "colleague-" + newID() + "@example.com"
	if got := postJSON(t, h, "/team/invites", owner, map[string]string{"email": colleague}); got.Code != http.StatusPaymentRequired {
		t.Fatalf("free plan invite %d %s", got.Code, got.Body)
	}
	if err := st.SetPlan(ctx, org, billing.Team); err != nil {
		t.Fatal(err)
	}
	if got := postJSON(t, h, "/team/invites", owner, map[string]string{"email": ownerEmail}); got.Code != http.StatusConflict {
		t.Fatalf("invite an existing account %d", got.Code)
	}
	if got := postJSON(t, h, "/team/invites", owner, map[string]string{"email": colleague}); got.Code != http.StatusCreated {
		t.Fatalf("invite %d %s", got.Code, got.Body)
	}
	if got := postJSON(t, h, "/team/invites", owner, map[string]string{"email": colleague}); got.Code != http.StatusConflict {
		t.Fatalf("duplicate invite %d", got.Code)
	}
	tok := mail.token(t, colleague, "invite")
	if got := postJSON(t, h, "/invites/accept", nil, map[string]any{"token": tok, "password": "colleague pass"}); got.Code != http.StatusBadRequest {
		t.Fatalf("accept without terms %d", got.Code)
	}
	rec := postJSON(t, h, "/invites/accept", nil, map[string]any{"token": tok, "password": "colleague pass", "accept_terms": true})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("accept %d %s", rec.Code, rec.Body)
	}
	member := sessionCookie(t, rec)
	// The member works in the owner's org and is verified by the link.
	if got := getWith(t, h, "/projects/"+project, member); got.Code != http.StatusOK {
		t.Fatalf("member reads shared project %d", got.Code)
	}
	if a := getAccount(t, h, member); a.Role != "member" || !a.Verified || a.Plan.ID != billing.Team {
		t.Fatalf("member account %+v", a)
	}
	// Only the owner manages billing and team.
	if got := postJSON(t, h, "/billing/checkout", member, map[string]string{"plan": "starter", "interval": "month"}); got.Code != http.StatusForbidden {
		t.Fatalf("member checkout %d", got.Code)
	}
	if got := postJSON(t, h, "/team/invites", member, map[string]string{"email": "x-" + newID() + "@example.com"}); got.Code != http.StatusForbidden {
		t.Fatalf("member invite %d", got.Code)
	}
	if got := postJSON(t, h, "/invites/accept", nil, map[string]any{"token": tok, "password": "again pass", "accept_terms": true}); got.Code != http.StatusUnauthorized {
		t.Fatalf("reused invite %d", got.Code)
	}
	// Five seats: owner, colleague and three more invites; the next is refused.
	for i := range 3 {
		if got := postJSON(t, h, "/team/invites", owner, map[string]string{"email": fmt.Sprintf("seat%d-%s@example.com", i, newID())}); got.Code != http.StatusCreated {
			t.Fatalf("seat %d %d %s", i, got.Code, got.Body)
		}
	}
	if got := postJSON(t, h, "/team/invites", owner, map[string]string{"email": "sixth-" + newID() + "@example.com"}); got.Code != http.StatusPaymentRequired {
		t.Fatalf("sixth seat %d", got.Code)
	}
	var team struct {
		Members []store.TeamMember
		Invites []store.TeamInvite
		Seats   int
	}
	_ = json.Unmarshal(getWith(t, h, "/team", owner).Body.Bytes(), &team)
	if len(team.Members) != 2 || len(team.Invites) != 3 || team.Seats != 5 {
		t.Fatalf("team %+v", team)
	}
	rev := httptest.NewRecorder()
	h.ServeHTTP(rev, request(t, http.MethodDelete, "/team/invites/"+team.Invites[0].ID, owner, nil))
	if rev.Code != http.StatusNoContent {
		t.Fatalf("revoke %d", rev.Code)
	}
	// The owner cannot be removed; a removed member is signed out.
	var memberID, ownerID string
	for _, m := range team.Members {
		if m.Role == "member" {
			memberID = m.UserID
		} else {
			ownerID = m.UserID
		}
	}
	del := httptest.NewRecorder()
	h.ServeHTTP(del, request(t, http.MethodDelete, "/team/members/"+ownerID, owner, nil))
	if del.Code != http.StatusConflict {
		t.Fatalf("remove owner %d", del.Code)
	}
	del = httptest.NewRecorder()
	h.ServeHTTP(del, request(t, http.MethodDelete, "/team/members/"+memberID, owner, nil))
	if del.Code != http.StatusNoContent {
		t.Fatalf("remove member %d", del.Code)
	}
	if got := getWith(t, h, "/projects/"+project, member); got.Code != http.StatusUnauthorized {
		t.Fatalf("removed member still signed in %d", got.Code)
	}
	if got := postJSON(t, h, "/signin", nil, creds(colleague, "colleague pass")); got.Code != http.StatusUnauthorized {
		t.Fatalf("removed member signs in %d", got.Code)
	}
}
