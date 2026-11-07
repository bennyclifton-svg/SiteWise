package config

import (
	"strings"
	"testing"
)

const (
	canaryPassword = "CANARY_DB_PASSWORD"
	canarySecret   = "CANARY_SESSION_SECRET"
	canaryAPIKey   = "CANARY_JEV_API_KEY"
)

func fullEnv() map[string]string {
	return map[string]string{
		"SITEWISE_DATABASE_URL":   "postgres://sitewise:" + canaryPassword + "@127.0.0.1/sitewise",
		"SITEWISE_FILE_DIR":       `D:\data\sitewise`,
		"SITEWISE_SESSION_SECRET": canarySecret,
		"SITEWISE_JEV_API_KEY":    canaryAPIKey,
		"SITEWISE_JEV_MODEL":      PinnedJevModel,
	}
}

func getenvFrom(vals map[string]string) func(string) string {
	return func(key string) string { return vals[key] }
}

func TestLoadRejectsMissingConfiguration(t *testing.T) {
	missing := []string{
		"SITEWISE_DATABASE_URL",
		"SITEWISE_FILE_DIR",
		"SITEWISE_SESSION_SECRET",
		"SITEWISE_JEV_API_KEY",
		"SITEWISE_JEV_MODEL",
	}
	for _, key := range missing {
		t.Run(key, func(t *testing.T) {
			vals := fullEnv()
			delete(vals, key)
			_, err := Load(getenvFrom(vals))
			if err == nil {
				t.Fatal("expected error")
			}
			msg := err.Error()
			if !strings.Contains(msg, key) {
				t.Fatalf("error %q does not name %s", msg, key)
			}
			for _, secret := range []string{canaryPassword, canarySecret, canaryAPIKey} {
				if strings.Contains(msg, secret) {
					t.Fatalf("error contains secret value %q", secret)
				}
			}
		})
	}
}

func TestLoadRejectsBlankSecrets(t *testing.T) {
	vals := fullEnv()
	vals["SITEWISE_JEV_API_KEY"] = "   "
	_, err := Load(getenvFrom(vals))
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), canaryPassword) || strings.Contains(err.Error(), canarySecret) {
		t.Fatalf("error contains a configured secret: %s", err.Error())
	}
}

func TestLoadRejectsMovingJevAlias(t *testing.T) {
	for _, alias := range []string{"jev-latest", "jev-1.13", "latest"} {
		t.Run(alias, func(t *testing.T) {
			vals := fullEnv()
			vals["SITEWISE_JEV_MODEL"] = alias
			_, err := Load(getenvFrom(vals))
			if err == nil {
				t.Fatal("expected error")
			}
			msg := err.Error()
			if strings.Contains(msg, canaryAPIKey) || strings.Contains(msg, canaryPassword) || strings.Contains(msg, canarySecret) {
				t.Fatalf("error contains a secret: %s", msg)
			}
		})
	}
}

func TestLoadAcceptsPinnedModel(t *testing.T) {
	cfg, err := Load(getenvFrom(fullEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JevModel != PinnedJevModel {
		t.Fatalf("model = %q", cfg.JevModel)
	}
	if cfg.DatabaseURL == "" || cfg.FileDir == "" || cfg.SessionSecret == "" || cfg.JevAPIKey == "" {
		t.Fatal("expected typed configuration to retain provided values")
	}
}

func TestLoadProductionRequiresMail(t *testing.T) {
	vals := fullEnv()
	vals["SITEWISE_ENV"] = "production"
	_, err := Load(getenvFrom(vals))
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "SITEWISE_MAIL_FROM") || !strings.Contains(msg, "SITEWISE_SMTP_ADDR") || !strings.Contains(msg, "SITEWISE_PUBLIC_ORIGIN") {
		t.Fatalf("error %q", msg)
	}
	if strings.Contains(msg, canaryAPIKey) || strings.Contains(msg, canaryPassword) || strings.Contains(msg, canarySecret) {
		t.Fatalf("error contains a secret: %s", msg)
	}
}

const canaryStripe = "CANARY_STRIPE_KEY"

func stripeEnv(vals map[string]string) {
	vals["SITEWISE_STRIPE_SECRET_KEY"] = canaryStripe
	vals["SITEWISE_STRIPE_WEBHOOK_SECRET"] = "whsec_canary"
	vals["SITEWISE_STRIPE_TAX_RATE"] = "txr_gst"
	vals["SITEWISE_STRIPE_PRICE_STARTER_MONTHLY"] = "price_sm"
	vals["SITEWISE_STRIPE_PRICE_STARTER_ANNUAL"] = "price_sa"
	vals["SITEWISE_STRIPE_PRICE_TEAM_MONTHLY"] = "price_tm"
	vals["SITEWISE_STRIPE_PRICE_TEAM_ANNUAL"] = "price_ta"
}

func TestLoadProductionRequiresStripe(t *testing.T) {
	vals := fullEnv()
	vals["SITEWISE_ENV"] = "production"
	vals["SITEWISE_MAIL_FROM"] = "no-reply@sitewise.au"
	vals["SITEWISE_SMTP_ADDR"] = "smtp.example:587"
	vals["SITEWISE_PUBLIC_ORIGIN"] = "https://sitewise.au"
	_, err := Load(getenvFrom(vals))
	if err == nil || !strings.Contains(err.Error(), "SITEWISE_STRIPE_SECRET_KEY") || !strings.Contains(err.Error(), "SITEWISE_STRIPE_PRICE_TEAM_ANNUAL") {
		t.Fatalf("err = %v", err)
	}
	stripeEnv(vals)
	cfg, err := Load(getenvFrom(vals))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Stripe.Enabled() || cfg.Stripe.Prices["team/year"] != "price_ta" || cfg.Stripe.TaxRateID != "txr_gst" {
		t.Fatalf("stripe %+v", cfg.Stripe.Prices)
	}
}

func TestLoadDevelopmentWithoutStripe(t *testing.T) {
	cfg, err := Load(getenvFrom(fullEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Stripe.Enabled() {
		t.Fatal("billing enabled without a key")
	}
	// A key without every price is a broken setup, refused even in development.
	vals := fullEnv()
	vals["SITEWISE_STRIPE_SECRET_KEY"] = canaryStripe
	_, err = Load(getenvFrom(vals))
	if err == nil || strings.Contains(err.Error(), canaryStripe) {
		t.Fatalf("err = %v", err)
	}
}
