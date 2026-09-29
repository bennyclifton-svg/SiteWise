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
	PublicOrigin  string
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
	return cfg, nil
}
