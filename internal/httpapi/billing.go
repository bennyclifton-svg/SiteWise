package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/billing"
	"sitewise/internal/store"
)

// Billing, plan limits and email confirmation. None of these routes call
// Jev. See docs/design/2026-11-07-billing.md.

// TermsVersion is the date of the Terms of Service and Privacy Policy a new
// user accepts (web/public/legal). Change it when either document changes.
const TermsVersion = "2026-11-07"

const (
	verifyLifetime   = 24 * time.Hour
	limitVerifyMails = 3
	webhookMaxBytes  = 64 << 10
)

// StripeAPI is the part of Stripe the server calls while a user waits.
type StripeAPI interface {
	Checkout(ctx context.Context, r billing.CheckoutRequest) (string, error)
	Portal(ctx context.Context, customerID, returnURL string) (string, error)
}

// Billing switches on paid plans. Nil Billing means billing is off: no
// checkout, no webhook, and no plan limits or email gate, which keeps local
// development and older tests unlimited. Production always sets it.
type Billing struct {
	Stripe        StripeAPI
	WebhookSecret string
	TaxRateID     string
	// Prices maps "<plan>/<month|year>" to a Stripe price id.
	Prices map[string]string
	Now    func() time.Time
}

func (b *Billing) now() time.Time {
	if b == nil || b.Now == nil {
		return time.Now()
	}
	return b.Now()
}

// planForPrice maps a Stripe price back to a plan; unknown gives "".
func (b *Billing) planForPrice(price string) string {
	for key, id := range b.Prices {
		if id == price && price != "" {
			plan, _, _ := strings.Cut(key, "/")
			return plan
		}
	}
	return ""
}

var billingRoutes = map[string]func(http.ResponseWriter, *http.Request, Deps){
	"GET /account":              getAccount,
	"POST /billing/checkout":    postCheckout,
	"POST /billing/portal":      postPortal,
	"POST /stripe/webhook":      stripeWebhook,
	"POST /email/verify":        verifyEmail,
	"POST /email/verify/resend": resendVerify,
	"GET /team":                 getTeam,
	"POST /team/invites":        postTeamInvite,
	"DELETE /team/invites/{id}": deleteTeamInvite,
	"DELETE /team/members/{id}": deleteTeamMember,
	"POST /invites/accept":      acceptTeamInvite,
}

const (
	pathBillingRead    = "billing_read"
	pathBillingSession = "billing_session"
	pathStripeWebhook  = "stripe_webhook"
	pathTeam           = "team"
)

func init() {
	for pattern, path := range map[string]string{
		"GET /account":              pathBillingRead,
		"POST /billing/checkout":    pathBillingSession,
		"POST /billing/portal":      pathBillingSession,
		"POST /stripe/webhook":      pathStripeWebhook,
		"POST /email/verify":        pathAccountAuth,
		"POST /email/verify/resend": pathAccountAuth,
		"GET /team":                 pathTeam,
		"POST /team/invites":        pathTeam,
		"DELETE /team/invites/{id}": pathTeam,
		"DELETE /team/members/{id}": pathTeam,
		"POST /invites/accept":      pathAccountAuth,
	} {
		routePaths[pattern] = path
	}
}

type accountView struct {
	store.Profile
	BillingEnabled    bool           `json:"billing_enabled"`
	Plan              billing.Plan   `json:"plan"`
	SubscribedPlan    string         `json:"subscribed_plan"`
	Status            string         `json:"status"`
	Interval          string         `json:"interval"`
	PaidUntil         *time.Time     `json:"paid_until"`
	CancelAtPeriodEnd bool           `json:"cancel_at_period_end"`
	HasCustomer       bool           `json:"has_customer"`
	Usage             store.Usage    `json:"usage"`
	Plans             []billing.Plan `json:"plans"`
}

func getAccount(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	ctx := r.Context()
	profile, err := deps.Store.Profile(ctx, session.OrgID, session.UserID)
	if err != nil {
		readFailed(w, deps, "account", err)
		return
	}
	b, err := deps.Store.OrgBilling(ctx, session.OrgID)
	if err != nil {
		readFailed(w, deps, "account", err)
		return
	}
	usage, err := deps.Store.Usage(ctx, session.OrgID, deps.Billing.now())
	if err != nil {
		readFailed(w, deps, "account", err)
		return
	}
	writeJSON(w, http.StatusOK, accountView{
		Profile: profile, BillingEnabled: deps.Billing != nil, Plan: b.Effective(),
		SubscribedPlan: b.Plan, Status: b.Status, Interval: b.Interval, PaidUntil: b.PaidUntil,
		CancelAtPeriodEnd: b.CancelAtPeriodEnd, HasCustomer: b.CustomerID != "",
		Usage: usage, Plans: billing.Plans(),
	})
}

func readFailed(w http.ResponseWriter, deps Deps, what string, err error) {
	deps.Log.Printf("%s: %v", what, err)
	http.Error(w, "read failed", http.StatusInternalServerError)
}

func billingOff(w http.ResponseWriter) {
	http.Error(w, "Billing is not set up on this server yet.", http.StatusServiceUnavailable)
}

func postCheckout(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := ownerSession(w, r, deps)
	if !ok {
		return
	}
	if deps.Billing == nil {
		billingOff(w)
		return
	}
	var body struct {
		Plan     string `json:"plan"`
		Interval string `json:"interval"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	price := deps.Billing.Prices[body.Plan+"/"+body.Interval]
	if !billing.Paid(body.Plan) || price == "" {
		http.Error(w, "Choose Starter or Team, monthly or yearly.", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	b, err := deps.Store.OrgBilling(ctx, session.OrgID)
	if err != nil {
		readFailed(w, deps, "checkout", err)
		return
	}
	// A second subscription would bill twice; plan changes go through the portal.
	if b.SubscriptionID != "" && billing.Effective(b.Plan, b.Status) != billing.Free {
		http.Error(w, "You already have a subscription. Use Manage billing to change plan.", http.StatusConflict)
		return
	}
	profile, err := deps.Store.Profile(ctx, session.OrgID, session.UserID)
	if err != nil {
		readFailed(w, deps, "checkout", err)
		return
	}
	url, err := deps.Billing.Stripe.Checkout(ctx, billing.CheckoutRequest{
		OrgID: session.OrgID, Email: profile.Email, CustomerID: b.CustomerID, PriceID: price,
		TaxRateID: deps.Billing.TaxRateID, SuccessURL: deps.PublicOrigin + "/account?checkout=done",
		CancelURL: deps.PublicOrigin + "/account",
	})
	if err != nil {
		deps.Log.Printf("checkout org=%s: %v", session.OrgID, err)
		http.Error(w, "Could not reach the payment page. Try again in a minute.", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func postPortal(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := ownerSession(w, r, deps)
	if !ok {
		return
	}
	if deps.Billing == nil {
		billingOff(w)
		return
	}
	b, err := deps.Store.OrgBilling(r.Context(), session.OrgID)
	if err != nil {
		readFailed(w, deps, "portal", err)
		return
	}
	if b.CustomerID == "" {
		http.Error(w, "There is no billing account yet. Choose a plan first.", http.StatusConflict)
		return
	}
	url, err := deps.Billing.Stripe.Portal(r.Context(), b.CustomerID, deps.PublicOrigin+"/account")
	if err != nil {
		deps.Log.Printf("portal org=%s: %v", session.OrgID, err)
		http.Error(w, "Could not reach the billing page. Try again in a minute.", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// stripeWebhook applies Stripe's signed events. It is the one route without
// an origin or session check: the signature is the authentication. A store
// failure answers 500 so Stripe retries; everything else answers 200 so a
// bad or unknown event is not retried forever.
func stripeWebhook(w http.ResponseWriter, r *http.Request, deps Deps) {
	if deps.Billing == nil {
		http.NotFound(w, r)
		return
	}
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, webhookMaxBytes))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := billing.VerifySignature(payload, r.Header.Get("Stripe-Signature"), deps.Billing.WebhookSecret, deps.Billing.now()); err != nil {
		deps.Log.Printf("stripe webhook: signature rejected")
		http.Error(w, "signature rejected", http.StatusBadRequest)
		return
	}
	e, err := billing.ParseEvent(payload)
	if err != nil {
		deps.Log.Printf("stripe webhook: unreadable event: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	outcome, err := deps.Store.ApplyStripe(r.Context(), e, deps.Billing.planForPrice(e.PriceID))
	if err != nil {
		deps.Log.Printf("stripe webhook %s %s: %v", e.Type, e.ID, err)
		http.Error(w, "not applied", http.StatusInternalServerError)
		return
	}
	if outcome == store.StripeApplied && strings.HasPrefix(e.Type, "customer.subscription.") && e.Type != "customer.subscription.deleted" && deps.Billing.planForPrice(e.PriceID) == "" {
		deps.Log.Printf("stripe webhook %s: price %q is not one of the configured prices; plan unchanged", e.ID, e.PriceID)
	}
	deps.Log.Printf("stripe webhook %s %s: %s", e.Type, e.ID, outcome)
	w.WriteHeader(http.StatusOK)
}

// uploadAllowed is checked before an upload's bytes are read. With billing
// on, the uploader's email must be confirmed and the org must be inside its
// monthly documents and storage. Parallel uploads at the cap may each pass:
// the overshoot is at most the number in flight (plan, Decisions).
func uploadAllowed(w http.ResponseWriter, r *http.Request, deps Deps, session store.Session) bool {
	if deps.Billing == nil {
		return true
	}
	ctx := r.Context()
	profile, err := deps.Store.Profile(ctx, session.OrgID, session.UserID)
	if err != nil {
		readFailed(w, deps, "upload allowance", err)
		return false
	}
	if !profile.Verified {
		http.Error(w, "Confirm your email address to upload documents. Use the link we emailed you, or send a new one from the Account page.", http.StatusForbidden)
		return false
	}
	b, err := deps.Store.OrgBilling(ctx, session.OrgID)
	if err != nil {
		readFailed(w, deps, "upload allowance", err)
		return false
	}
	usage, err := deps.Store.Usage(ctx, session.OrgID, deps.Billing.now())
	if err != nil {
		readFailed(w, deps, "upload allowance", err)
		return false
	}
	plan := b.Effective()
	if usage.DocumentsThisMonth >= plan.DocumentsPerMonth {
		http.Error(w, fmt.Sprintf("Your %s plan reads %d documents a month and this month's are used. Upgrade on the Account page to read more.", plan.Name, plan.DocumentsPerMonth), http.StatusPaymentRequired)
		return false
	}
	if usage.StorageBytes >= plan.StorageBytes {
		http.Error(w, fmt.Sprintf("Your %s plan stores %s and it is full. Delete documents or upgrade on the Account page.", plan.Name, sizeLabel(plan.StorageBytes)), http.StatusPaymentRequired)
		return false
	}
	return true
}

// projectAllowed is checked before a project is created.
func projectAllowed(w http.ResponseWriter, r *http.Request, deps Deps, session store.Session) bool {
	if deps.Billing == nil {
		return true
	}
	b, err := deps.Store.OrgBilling(r.Context(), session.OrgID)
	if err != nil {
		readFailed(w, deps, "project allowance", err)
		return false
	}
	usage, err := deps.Store.Usage(r.Context(), session.OrgID, deps.Billing.now())
	if err != nil {
		readFailed(w, deps, "project allowance", err)
		return false
	}
	plan := b.Effective()
	if !plan.AllowsProjects(usage.Projects) {
		http.Error(w, fmt.Sprintf("Your %s plan includes %d active project%s. Delete one or upgrade on the Account page.", plan.Name, plan.Projects, plural(plan.Projects)), http.StatusPaymentRequired)
		return false
	}
	return true
}

// countDocument records one document read this month. It runs whether or not
// billing is on, so real use per org can be measured before prices go live.
// A failed count is logged and never fails the upload that already landed.
func countDocument(deps Deps, orgID string) {
	if err := deps.Store.CountDocument(context.Background(), orgID, deps.Billing.now()); err != nil {
		deps.Log.Printf("usage count org=%s: %v", orgID, err)
	}
}

func sizeLabel(bytes int64) string {
	const gb = 1 << 30
	if bytes >= gb {
		return fmt.Sprintf("%d GB", bytes/gb)
	}
	return fmt.Sprintf("%d MB", bytes>>20)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// sendVerify mails a fresh confirmation link after the response, like a
// password reset, so a slow mail provider never holds a request.
func sendVerify(deps Deps, orgID, userID, email string) {
	if deps.AccountMail == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	raw, hash, err := auth.NewToken()
	if err != nil {
		deps.Log.Printf("verify mail: %v", err)
		return
	}
	if err := deps.Store.CreateEmailVerification(ctx, orgID, userID, hash, time.Now().Add(verifyLifetime)); err != nil {
		deps.Log.Printf("verify mail: %v", err)
		return
	}
	// The token rides in the fragment, which browsers never send to a server.
	if err := deps.AccountMail.SendVerify(ctx, email, deps.PublicOrigin+"/#verify="+raw); err != nil {
		deps.Log.Printf("verify mail failed: %v", err)
	}
}

func verifyEmail(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !accountRequestOK(w, r, deps) {
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err := deps.Store.VerifyEmail(r.Context(), auth.HashToken(strings.TrimSpace(body.Token)), time.Now())
	if errors.Is(err, store.ErrLinkUsed) {
		http.Error(w, "That confirmation link has expired or was already used. Send a new one from the Account page.", http.StatusUnauthorized)
		return
	}
	if err != nil {
		deps.Log.Printf("verify email: %v", err)
		http.Error(w, "verification failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func resendVerify(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	profile, err := deps.Store.Profile(r.Context(), session.OrgID, session.UserID)
	if err != nil {
		readFailed(w, deps, "resend verify", err)
		return
	}
	if profile.Verified {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if deps.AccountMail == nil {
		http.Error(w, "Email is not set up on this server.", http.StatusServiceUnavailable)
		return
	}
	if !deps.limits.allow("verify:"+session.UserID, limitVerifyMails, time.Hour) {
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "We have sent three links in the last hour. Check your inbox and spam folder, or try again later.", http.StatusTooManyRequests)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	go sendVerify(deps, session.OrgID, session.UserID, profile.Email)
}
