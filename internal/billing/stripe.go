package billing

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Stripe is reached over plain HTTPS: two form POSTs and webhook signature
// checks are all SiteWise needs, so the official SDK is not a dependency.
// API reference: https://docs.stripe.com/api and
// https://docs.stripe.com/webhooks#verify-manually

// DefaultBaseURL is Stripe's API host.
const DefaultBaseURL = "https://api.stripe.com"

// signatureTolerance is Stripe's recommended replay window.
const signatureTolerance = 5 * time.Minute

// ErrSignature means a webhook did not come from Stripe with our secret.
var ErrSignature = errors.New("stripe signature rejected")

// Client calls the Stripe API with a secret key.
type Client struct {
	Key     string
	BaseURL string
	HTTP    *http.Client
}

// CheckoutRequest is one subscription checkout for one org.
type CheckoutRequest struct {
	OrgID      string
	Email      string
	CustomerID string // reuse an existing Stripe customer when known
	PriceID    string
	TaxRateID  string // GST at 10%, exclusive; blank adds none
	SuccessURL string
	CancelURL  string
}

// Checkout creates a hosted Checkout session and returns its URL. The org id
// rides in the session and the subscription metadata, never from a browser.
func (c Client) Checkout(ctx context.Context, r CheckoutRequest) (string, error) {
	form := url.Values{
		"mode":                                {"subscription"},
		"line_items[0][price]":                {r.PriceID},
		"line_items[0][quantity]":             {"1"},
		"client_reference_id":                 {r.OrgID},
		"metadata[org_id]":                    {r.OrgID},
		"subscription_data[metadata][org_id]": {r.OrgID},
		"success_url":                         {r.SuccessURL},
		"cancel_url":                          {r.CancelURL},
		// A tax invoice needs the buyer's business details and ABN.
		"billing_address_collection": {"required"},
		"tax_id_collection[enabled]": {"true"},
	}
	if r.TaxRateID != "" {
		form.Set("line_items[0][tax_rates][0]", r.TaxRateID)
	}
	if r.CustomerID != "" {
		form.Set("customer", r.CustomerID)
		// Stripe requires these when collecting a tax id for a known customer.
		form.Set("customer_update[name]", "auto")
		form.Set("customer_update[address]", "auto")
	} else {
		form.Set("customer_email", r.Email)
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := c.post(ctx, "/v1/checkout/sessions", form, &out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", errors.New("stripe checkout returned no url")
	}
	return out.URL, nil
}

// Portal creates a Customer Portal session and returns its URL.
func (c Client) Portal(ctx context.Context, customerID, returnURL string) (string, error) {
	var out struct {
		URL string `json:"url"`
	}
	form := url.Values{"customer": {customerID}, "return_url": {returnURL}}
	if err := c.post(ctx, "/v1/billing_portal/sessions", form, &out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", errors.New("stripe portal returned no url")
	}
	return out.URL, nil
}

func (c Client) post(ctx context.Context, path string, form url.Values, dest any) error {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// A retried click never creates a second session.
	req.Header.Set("Idempotency-Key", randomKey())
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("stripe %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("stripe %s: %w", path, err)
	}
	if resp.StatusCode/100 != 2 {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		return fmt.Errorf("stripe %s: %d %s", path, resp.StatusCode, e.Error.Message)
	}
	return json.Unmarshal(body, dest)
}

func randomKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// VerifySignature checks a Stripe-Signature header against the raw body:
// HMAC-SHA256 of "<t>.<body>" with the endpoint secret, within five minutes
// either side of now.
func VerifySignature(payload []byte, header, secret string, now time.Time) error {
	if secret == "" || header == "" {
		return ErrSignature
	}
	var ts int64
	var sigs [][]byte
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return ErrSignature
			}
			ts = n
		case "v1":
			if b, err := hex.DecodeString(v); err == nil {
				sigs = append(sigs, b)
			}
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return ErrSignature
	}
	age := now.Sub(time.Unix(ts, 0))
	if age > signatureTolerance || age < -signatureTolerance {
		return ErrSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	want := mac.Sum(nil)
	for _, s := range sigs {
		if hmac.Equal(s, want) {
			return nil
		}
	}
	return ErrSignature
}

// Event is the part of a Stripe event that billing applies.
type Event struct {
	ID                string
	Type              string
	Created           time.Time
	OrgID             string
	CustomerID        string
	SubscriptionID    string
	Status            string
	PriceID           string
	Interval          string
	PeriodEnd         time.Time
	CancelAtPeriodEnd bool
}

// Handled reports whether billing acts on this event type.
func (e Event) Handled() bool {
	switch e.Type {
	case "checkout.session.completed", "customer.subscription.created",
		"customer.subscription.updated", "customer.subscription.deleted":
		return true
	}
	return false
}

// ParseEvent reads a verified webhook body. Unknown types parse with only
// their id, type and time, so the caller can acknowledge and skip them.
func ParseEvent(payload []byte) (Event, error) {
	var raw struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Created int64  `json:"created"`
		Data    struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Event{}, err
	}
	if raw.ID == "" || raw.Type == "" {
		return Event{}, errors.New("stripe event has no id or type")
	}
	e := Event{ID: raw.ID, Type: raw.Type, Created: time.Unix(raw.Created, 0)}
	switch {
	case raw.Type == "checkout.session.completed":
		var o struct {
			ClientReferenceID string            `json:"client_reference_id"`
			Customer          string            `json:"customer"`
			Subscription      string            `json:"subscription"`
			Metadata          map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(raw.Data.Object, &o); err != nil {
			return Event{}, err
		}
		e.OrgID = o.ClientReferenceID
		if e.OrgID == "" {
			e.OrgID = o.Metadata["org_id"]
		}
		e.CustomerID, e.SubscriptionID = o.Customer, o.Subscription
	case strings.HasPrefix(raw.Type, "customer.subscription."):
		var o struct {
			ID                string            `json:"id"`
			Customer          string            `json:"customer"`
			Status            string            `json:"status"`
			CancelAtPeriodEnd bool              `json:"cancel_at_period_end"`
			CurrentPeriodEnd  int64             `json:"current_period_end"`
			Metadata          map[string]string `json:"metadata"`
			Items             struct {
				Data []struct {
					CurrentPeriodEnd int64 `json:"current_period_end"`
					Price            struct {
						ID        string `json:"id"`
						Recurring struct {
							Interval string `json:"interval"`
						} `json:"recurring"`
					} `json:"price"`
				} `json:"data"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw.Data.Object, &o); err != nil {
			return Event{}, err
		}
		e.SubscriptionID, e.CustomerID, e.Status = o.ID, o.Customer, o.Status
		e.CancelAtPeriodEnd, e.OrgID = o.CancelAtPeriodEnd, o.Metadata["org_id"]
		end := o.CurrentPeriodEnd
		if len(o.Items.Data) > 0 {
			item := o.Items.Data[0]
			e.PriceID, e.Interval = item.Price.ID, item.Price.Recurring.Interval
			if item.CurrentPeriodEnd != 0 {
				end = item.CurrentPeriodEnd
			}
		}
		if end != 0 {
			e.PeriodEnd = time.Unix(end, 0)
		}
	}
	return e, nil
}
