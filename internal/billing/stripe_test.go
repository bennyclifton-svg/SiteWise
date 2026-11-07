package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func sign(payload, secret string, at time.Time) string {
	ts := fmt.Sprint(at.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + payload))
	return "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	body := `{"id":"evt_1"}`
	good := sign(body, "whsec_test", now)
	if err := VerifySignature([]byte(body), good, "whsec_test", now); err != nil {
		t.Fatalf("valid signature refused: %v", err)
	}
	// A second v1 (Stripe rolls secrets) may carry the match.
	both := "t=" + fmt.Sprint(now.Unix()) + ",v1=00ff," + strings.Split(good, ",")[1]
	if err := VerifySignature([]byte(body), both, "whsec_test", now); err != nil {
		t.Fatalf("second v1 refused: %v", err)
	}
	bad := []struct {
		name, header, secret, body string
		at                         time.Time
	}{
		{"wrong secret", good, "whsec_other", body, now},
		{"changed body", good, "whsec_test", `{"id":"evt_2"}`, now},
		{"stale", good, "whsec_test", body, now.Add(6 * time.Minute)},
		{"future", good, "whsec_test", body, now.Add(-6 * time.Minute)},
		{"empty", "", "whsec_test", body, now},
		{"no v1", "t=" + fmt.Sprint(now.Unix()), "whsec_test", body, now},
		{"garbage", "nonsense", "whsec_test", body, now},
		{"empty secret", good, "", body, now},
	}
	for _, c := range bad {
		if err := VerifySignature([]byte(c.body), c.header, c.secret, c.at); !errors.Is(err, ErrSignature) {
			t.Errorf("%s: err = %v, want ErrSignature", c.name, err)
		}
	}
}

func TestCheckoutSendsSubscriptionForm(t *testing.T) {
	var form url.Values
	var auth, idem string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" || r.Method != http.MethodPost {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		auth, idem = r.Header.Get("Authorization"), r.Header.Get("Idempotency-Key")
		_ = r.ParseForm()
		form = r.PostForm
		_, _ = w.Write([]byte(`{"id":"cs_1","url":"https://checkout.stripe.com/c/pay/cs_1"}`))
	}))
	defer srv.Close()
	c := Client{Key: "sk_test_1", BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := c.Checkout(context.Background(), CheckoutRequest{
		OrgID: "org-1", Email: "a@example.test", PriceID: "price_s", TaxRateID: "txr_gst",
		SuccessURL: "https://sitewise.au/account?checkout=done", CancelURL: "https://sitewise.au/account",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://checkout.stripe.com/c/pay/cs_1" {
		t.Fatalf("url = %q", got)
	}
	if auth != "Bearer sk_test_1" || idem == "" {
		t.Fatalf("auth %q idempotency %q", auth, idem)
	}
	want := map[string]string{
		"mode":                                "subscription",
		"line_items[0][price]":                "price_s",
		"line_items[0][quantity]":             "1",
		"line_items[0][tax_rates][0]":         "txr_gst",
		"client_reference_id":                 "org-1",
		"metadata[org_id]":                    "org-1",
		"subscription_data[metadata][org_id]": "org-1",
		"customer_email":                      "a@example.test",
		"billing_address_collection":          "required",
		"tax_id_collection[enabled]":          "true",
	}
	for k, v := range want {
		if form.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, form.Get(k), v)
		}
	}
	if form.Get("customer") != "" {
		t.Error("customer sent for a new customer")
	}
}

func TestCheckoutReusesCustomer(t *testing.T) {
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		_, _ = w.Write([]byte(`{"url":"https://checkout.stripe.com/x"}`))
	}))
	defer srv.Close()
	c := Client{Key: "sk", BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Checkout(context.Background(), CheckoutRequest{OrgID: "o", Email: "a@example.test", CustomerID: "cus_1", PriceID: "p", SuccessURL: "s", CancelURL: "c"}); err != nil {
		t.Fatal(err)
	}
	if form.Get("customer") != "cus_1" || form.Get("customer_email") != "" || form.Get("customer_update[name]") != "auto" {
		t.Fatalf("form %v", form)
	}
}

func TestStripeErrorIsReturnedWithoutKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"No such price: 'price_x'"}}`))
	}))
	defer srv.Close()
	c := Client{Key: "sk_live_secret", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Portal(context.Background(), "cus_1", "https://sitewise.au/account")
	if err == nil || !strings.Contains(err.Error(), "No such price") || strings.Contains(err.Error(), "sk_live_secret") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseSubscriptionEvent(t *testing.T) {
	// Newer API versions keep the period end on the item.
	body := `{"id":"evt_9","type":"customer.subscription.updated","created":1800000000,
	 "data":{"object":{"id":"sub_1","customer":"cus_1","status":"past_due","cancel_at_period_end":true,
	 "metadata":{"org_id":"org-1"},
	 "items":{"data":[{"current_period_end":1802592000,"price":{"id":"price_t","recurring":{"interval":"year"}}}]}}}}`
	e, err := ParseEvent([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != "evt_9" || e.Type != "customer.subscription.updated" || e.OrgID != "org-1" || e.CustomerID != "cus_1" ||
		e.SubscriptionID != "sub_1" || e.Status != "past_due" || e.PriceID != "price_t" || e.Interval != "year" ||
		!e.CancelAtPeriodEnd || e.PeriodEnd.Unix() != 1802592000 || e.Created.Unix() != 1800000000 {
		t.Fatalf("event %+v", e)
	}
	// Older API versions keep it on the subscription.
	old := `{"id":"evt_8","type":"customer.subscription.created","created":1,"data":{"object":{"id":"sub_1","customer":"cus_1","status":"active","current_period_end":5,"items":{"data":[{"price":{"id":"p","recurring":{"interval":"month"}}}]}}}}`
	e, err = ParseEvent([]byte(old))
	if err != nil || e.PeriodEnd.Unix() != 5 || e.OrgID != "" {
		t.Fatalf("old event %+v %v", e, err)
	}
}

func TestParseCheckoutCompleted(t *testing.T) {
	body := `{"id":"evt_c","type":"checkout.session.completed","created":2,"data":{"object":{"id":"cs_1","mode":"subscription","client_reference_id":"org-1","customer":"cus_1","subscription":"sub_1"}}}`
	e, err := ParseEvent([]byte(body))
	if err != nil || e.OrgID != "org-1" || e.CustomerID != "cus_1" || e.SubscriptionID != "sub_1" || e.Status != "" {
		t.Fatalf("%+v %v", e, err)
	}
	if _, err := ParseEvent([]byte(`{"id":"","type":"x"}`)); err == nil {
		t.Fatal("event without id accepted")
	}
}
