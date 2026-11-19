package config

import (
	"fmt"
	"regexp"
	"strconv"
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
	// TenderKey is the model provider's key for the tender pass. Blank
	// means the pass is off for every org.
	TenderKey string
	// TenderModel is the pinned model id for the tender pass.
	TenderModel string
	// ChatModel is the model id for the chat strip.
	ChatModel string
	// ChatThinking lets the chat model use hidden thinking at low effort.
	// Off by default: measured on 15 November it was a little slower.
	ChatThinking bool
	// TenderSkipJevPrices stops asking Jev the prices the tender pass reads.
	// Off until the owner has used the new tables on a live package.
	TenderSkipJevPrices bool
	// MailInDomain and MailgunSigningKey switch on email intake: the domain
	// Mailgun receives project mail for, and the account's HTTP webhook
	// signing key. Both or neither; neither is off (the webhook answers 404
	// and no project address is shown).
	MailInDomain      string
	MailgunSigningKey string
	// MailSendDailyCap is every paid org's sends together in a day
	// (SITEWISE_MAIL_SEND_DAILY_CAP); zero is the default, 2,000.
	// MailSendFreeDailyCap is every free org's (…_FREE_DAILY_CAP; 200).
	MailSendDailyCap, MailSendFreeDailyCap int
	// MailSend is SITEWISE_MAIL_SEND=1: sending mail out is on. Off by
	// default; it stays off while Stripe is in test mode.
	MailSend bool
}

// MailInEnabled reports whether email intake is configured.
func (c Config) MailInEnabled() bool { return c.MailInDomain != "" && c.MailgunSigningKey != "" }

var mailDomain = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

// loadMailIn reads the two email intake settings. The key's value never
// appears in an error.
func loadMailIn(getenv func(string) string) (domain, key string, err error) {
	domain = strings.ToLower(strings.TrimSpace(getenv("SITEWISE_MAIL_IN_DOMAIN")))
	key = strings.TrimSpace(getenv("SITEWISE_MAILGUN_SIGNING_KEY"))
	switch {
	case domain == "" && key == "":
		return "", "", nil
	case domain == "":
		return "", "", fmt.Errorf("missing configuration: SITEWISE_MAIL_IN_DOMAIN (SITEWISE_MAILGUN_SIGNING_KEY is set)")
	case key == "":
		return "", "", fmt.Errorf("missing configuration: SITEWISE_MAILGUN_SIGNING_KEY (SITEWISE_MAIL_IN_DOMAIN is set)")
	case !mailDomain.MatchString(domain):
		return "", "", fmt.Errorf("SITEWISE_MAIL_IN_DOMAIN must be a domain name such as in.example.com")
	}
	return domain, key, nil
}

// DefaultTenderModel reads tenders unless SITEWISE_TENDER_MODEL names another.
const DefaultTenderModel = "claude-haiku-5-5"

var tenderModelID = regexp.MustCompile(`^claude-[a-z]+-[0-9]+(-[0-9]+)?$`)

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
	// A missing key is not an error: the tender pass is simply off.
	cfg.TenderKey = strings.TrimSpace(getenv("SITEWISE_ANTHROPIC_HAIKU_KEY"))
	cfg.TenderModel = strings.TrimSpace(getenv("SITEWISE_TENDER_MODEL"))
	if cfg.TenderModel == "" {
		cfg.TenderModel = DefaultTenderModel
	}
	if !tenderModelID.MatchString(cfg.TenderModel) {
		return Config{}, fmt.Errorf("SITEWISE_TENDER_MODEL must be a pinned Claude model id")
	}
	// Chat uses the same key and its own model id, so the owner can move
	// chat to another model without touching the tender pass.
	cfg.ChatModel = strings.TrimSpace(getenv("SITEWISE_CHAT_MODEL"))
	if cfg.ChatModel == "" {
		cfg.ChatModel = DefaultTenderModel
	}
	cfg.ChatThinking = strings.TrimSpace(getenv("SITEWISE_CHAT_THINKING")) == "1"
	if !tenderModelID.MatchString(cfg.ChatModel) {
		return Config{}, fmt.Errorf("SITEWISE_CHAT_MODEL %q is not a model id", cfg.ChatModel)
	}
	switch v := strings.TrimSpace(getenv("SITEWISE_TENDER_SKIP_JEV_PRICES")); v {
	case "", "0", "false":
	case "1", "true":
		cfg.TenderSkipJevPrices = true
	default:
		return Config{}, fmt.Errorf("SITEWISE_TENDER_SKIP_JEV_PRICES must be 1 or 0")
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
	mailDomainIn, mailKey, err := loadMailIn(getenv)
	if err != nil {
		return Config{}, err
	}
	cfg.MailInDomain, cfg.MailgunSigningKey = mailDomainIn, mailKey
	for key, into := range map[string]*int{"SITEWISE_MAIL_SEND_DAILY_CAP": &cfg.MailSendDailyCap, "SITEWISE_MAIL_SEND_FREE_DAILY_CAP": &cfg.MailSendFreeDailyCap} {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 1000000 {
				return Config{}, fmt.Errorf("%s must be a whole number from 1", key)
			}
			*into = n
		}
	}
	switch v := strings.TrimSpace(getenv("SITEWISE_MAIL_SEND")); v {
	case "", "0", "false":
	case "1", "true":
		cfg.MailSend = true
	default:
		return Config{}, fmt.Errorf("SITEWISE_MAIL_SEND must be 1 or 0")
	}
	stripe, err := loadStripe(getenv, env == "production")
	if err != nil {
		return Config{}, err
	}
	cfg.Stripe = stripe
	return cfg, nil
}
