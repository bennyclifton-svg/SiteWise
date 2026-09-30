package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/config"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/latency"
	"sitewise/internal/store"
	"sitewise/web"
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
	if len(args) > 0 && args[0] == "serve" {
		return runServe(args[1:], getenv, stderr, stdout)
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fmt.Fprintf(stdout, "sitewise ready model=%s\n", cfg.JevModel)
	return 0
}

// runServe starts the API, event stream and embedded SPA. PDFium and the Jev
// connection are warmed before the listener opens so the first filing is not
// the one that pays for them.
func runServe(args []string, getenv func(string) string, stderr, stdout io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "127.0.0.1:8080", "listen address")
	data := fs.String("data", "data/intake", "intake vocabulary and thresholds directory")
	maxUpload := fs.Int64("max-upload", 200<<20, "largest accepted upload in bytes")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	origin := cfg.PublicOrigin
	if origin == "" {
		origin = localOrigin(*addr)
	}
	logger := log.New(stderr, "", log.LstdFlags)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	blobs, err := files.Open(cfg.FileDir, *maxUpload)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	cat, err := intake.LoadCatalog(*data)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	thresholds, err := intake.LoadThresholds(*data)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	client, err := jev.New(jev.Options{APIKey: cfg.JevAPIKey, Model: cfg.JevModel, Logger: slog.New(slog.NewJSONHandler(stderr, nil))})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := identity.Warm(ctx); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := client.Warm(ctx); err != nil {
		logger.Printf("jev warm-up failed; filings will show grey until it answers: %v", err)
	}
	srv, err := httpapi.New(httpapi.Options{
		Store:          st,
		Blobs:          blobs,
		Jev:            client,
		Catalog:        cat,
		Thresholds:     thresholds,
		Static:         web.Dist(),
		PublicOrigin:   origin,
		SecureCookie:   cfg.Env == "production",
		MaxUploadBytes: *maxUpload,
		Log:            logger,
	})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if n, err := srv.Resume(ctx); err != nil {
		logger.Printf("resume failed: %v", err)
	} else if n > 0 {
		logger.Printf("resumed %d filings", n)
	}
	return listen(ctx, &http.Server{Addr: *addr, Handler: srv, ReadHeaderTimeout: 10 * time.Second}, srv, logger, stdout, cfg.JevModel)
}

func listen(ctx context.Context, hs *http.Server, srv *httpapi.Server, logger *log.Logger, stdout io.Writer, model string) int {
	errc := make(chan error, 1)
	go func() { errc <- hs.ListenAndServe() }()
	fmt.Fprintf(stdout, "sitewise listening addr=%s model=%s\n", hs.Addr, model)
	select {
	case err := <-errc:
		logger.Printf("listen: %v", err)
		return 1
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// Event streams never go idle on their own; end them so Shutdown can.
	hs.RegisterOnShutdown(srv.CloseStreams)
	if err := hs.Shutdown(shutdown); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		logger.Printf("shutdown: %v", err)
	}
	if err := srv.Wait(shutdown); err != nil {
		logger.Printf("filings still running at shutdown resume on next start: %v", err)
	}
	return 0
}

// localOrigin is the development origin when none is configured. Production
// config requires SITEWISE_PUBLIC_ORIGIN.
func localOrigin(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
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
