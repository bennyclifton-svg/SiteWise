package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"sitewise/internal/auth"
	"sitewise/internal/config"
	"sitewise/internal/latency"
	"sitewise/internal/store"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr, os.Stdout))
}

func run(args []string, getenv func(string) string, stderr, stdout io.Writer) int {
	if len(args) > 0 && args[0] == "gate" {
		return latency.RunGate(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "bootstrap" {
		return runBootstrap(args[1:], getenv, stderr, stdout)
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fmt.Fprintf(stdout, "sitewise ready model=%s\n", cfg.JevModel)
	return 0
}

func runBootstrap(args []string, getenv func(string) string, stderr, stdout io.Writer) int {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	org := fs.String("org", "", "organisation name")
	email := fs.String("email", "", "admin email")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *org == "" || *email == "" {
		fmt.Fprintln(stderr, "bootstrap requires -org and -email")
		return 2
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx := context.Background()
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	var mail auth.Mailer = &auth.Sink{}
	if cfg.Env == "production" {
		mail = auth.SMTP{Addr: cfg.SMTPAddr, From: cfg.MailFrom}
	}
	raw, _, err := auth.Bootstrap(ctx, st, mail, *org, *email, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fmt.Fprintln(stdout, raw)
	return 0
}
