package config

import (
	"fmt"
	"strings"
)

// PinnedJevModel is the only admitted System One version.
// Moving aliases such as jev-latest are rejected.
const PinnedJevModel = "jev-1.13.0"

// Config is the process configuration. Fields are present only after Load
// succeeds; Load never puts a secret value into an error string.
type Config struct {
	DatabaseURL   string
	FileDir       string
	SessionSecret string
	JevAPIKey     string
	JevModel      string
	Env           string
	MailFrom      string
	SMTPAddr      string
	// SMTPUser and SMTPPassword sign in to a mail provider such as Resend.
	// Blank means an unauthenticated local relay.
	SMTPUser     string
	SMTPPassword string
	PublicOrigin string
	Stripe       Stripe
}

// Stripe is the billing setup. Without a secret key billing is off: plan
// limits still apply, and upgrading reports that billing is not set up.
type Stripe struct {
	SecretKey     string
	WebhookSecret string
	// TaxRateID is the 10% GST tax rate, exclusive. Optional.
	TaxRateID string
	// Prices maps "<plan>/<month|year>" to a Stripe price id.
	Prices map[string]string
}

// Enabled reports whether checkout and webhooks are configured.
func (s Stripe) Enabled() bool { return s.SecretKey != "" }

var stripePriceKeys = map[string]string{
	"starter/month": "SITEWISE_STRIPE_PRICE_STARTER_MONTHLY",
	"starter/year":  "SITEWISE_STRIPE_PRICE_STARTER_ANNUAL",
	"team/month":    "SITEWISE_STRIPE_PRICE_TEAM_MONTHLY",
	"team/year":     "SITEWISE_STRIPE_PRICE_TEAM_ANNUAL",
}

// loadStripe reads billing settings. A key needs the webhook secret and all
// four prices; production needs the key.
func loadStripe(getenv func(string) string, production bool) (Stripe, error) {
	s := Stripe{
		SecretKey:     strings.TrimSpace(getenv("SITEWISE_STRIPE_SECRET_KEY")),
		WebhookSecret: strings.TrimSpace(getenv("SITEWISE_STRIPE_WEBHOOK_SECRET")),
		TaxRateID:     strings.TrimSpace(getenv("SITEWISE_STRIPE_TAX_RATE")),
		Prices:        map[string]string{},
	}
	if s.SecretKey == "" && !production {
		return Stripe{}, nil
	}
	var missing []string
	if s.SecretKey == "" {
		missing = append(missing, "SITEWISE_STRIPE_SECRET_KEY")
	}
	if s.WebhookSecret == "" {
		missing = append(missing, "SITEWISE_STRIPE_WEBHOOK_SECRET")
	}
	for _, key := range []string{"starter/month", "starter/year", "team/month", "team/year"} {
		v := strings.TrimSpace(getenv(stripePriceKeys[key]))
		if v == "" {
			missing = append(missing, stripePriceKeys[key])
			continue
		}
		s.Prices[key] = v
	}
	if len(missing) > 0 {
		return Stripe{}, fmt.Errorf("missing configuration: %s", strings.Join(missing, ", "))
	}
	return s, nil
}

// Load reads required configuration. getenv is injected so tests can supply
// values without process-global state. A blank value counts as missing.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, fmt.Errorf("missing configuration: getenv")
	}
	keys := []string{
		"SITEWISE_DATABASE_URL",
		"SITEWISE_FILE_DIR",
		"SITEWISE_SESSION_SECRET",
		"SITEWISE_JEV_API_KEY",
		"SITEWISE_JEV_MODEL",
	}
	values := make(map[string]string, len(keys))
	var missing []string
	for _, key := range keys {
		value := strings.TrimSpace(getenv(key))
		if value == "" {
			missing = append(missing, key)
			continue
		}
		values[key] = value
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing configuration: %s", strings.Join(missing, ", "))
	}
	model := values["SITEWISE_JEV_MODEL"]
	if model != PinnedJevModel {
		return Config{}, fmt.Errorf("jev model %q is not the pinned version %s", model, PinnedJevModel)
	}
	env := strings.TrimSpace(getenv("SITEWISE_ENV"))
	if env == "" {
		env = "development"
	}
	cfg := Config{
		DatabaseURL:   values["SITEWISE_DATABASE_URL"],
		FileDir:       values["SITEWISE_FILE_DIR"],
		SessionSecret: values["SITEWISE_SESSION_SECRET"],
		JevAPIKey:     values["SITEWISE_JEV_API_KEY"],
		JevModel:      model,
		Env:           env,
		MailFrom:      strings.TrimSpace(getenv("SITEWISE_MAIL_FROM")),
		SMTPAddr:      strings.TrimSpace(getenv("SITEWISE_SMTP_ADDR")),
		PublicOrigin:  strings.TrimSpace(getenv("SITEWISE_PUBLIC_ORIGIN")),
		SMTPUser:      strings.TrimSpace(getenv("SITEWISE_SMTP_USER")),
		SMTPPassword:  getenv("SITEWISE_SMTP_PASSWORD"),
	}
	if env == "production" {
		var missing []string
		if cfg.MailFrom == "" {
			missing = append(missing, "SITEWISE_MAIL_FROM")
		}
		if cfg.SMTPAddr == "" {
			missing = append(missing, "SITEWISE_SMTP_ADDR")
		}
		if cfg.PublicOrigin == "" {
			missing = append(missing, "SITEWISE_PUBLIC_ORIGIN")
		}
		if len(missing) > 0 {
			return Config{}, fmt.Errorf("missing configuration: %s", strings.Join(missing, ", "))
		}
	}
	stripe, err := loadStripe(getenv, env == "production")
	if err != nil {
		return Config{}, err
	}
	cfg.Stripe = stripe
	return cfg, nil
}
