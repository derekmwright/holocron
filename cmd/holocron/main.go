package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/derekmwright/holocron/internal/platform"
	"github.com/derekmwright/holocron/internal/telemetry"
	"github.com/derekmwright/holocron/internal/transport"
)

func main() {
	if err := run(); err != nil {
		slog.Error("holocron stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "migrate":
		return platform.Migrate(ctx, os.Getenv("HOLOCRON_ADMIN_DATABASE_URL"))
	case "provision":
		var in platform.ProvisionInput
		if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
			return err
		}
		return platform.Provision(ctx, os.Getenv("HOLOCRON_ADMIN_DATABASE_URL"), in)
	case "revoke":
		var in struct{ Tenant, Domain, ID string }
		if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
			return err
		}
		return platform.Revoke(ctx, os.Getenv("HOLOCRON_ADMIN_DATABASE_URL"), in.Tenant, in.Domain, in.ID)
	case "purge-domain":
		var in struct{ Tenant, Domain string }
		if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
			return err
		}
		return platform.PurgeDomain(ctx, os.Getenv("HOLOCRON_ADMIN_DATABASE_URL"), in.Tenant, in.Domain)
	case "serve":
		shutdownTelemetry, err := telemetry.Start()
		if err != nil {
			return err
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdownTelemetry(ctx)
		}()
		s, err := platform.Open(ctx, os.Getenv("HOLOCRON_DATABASE_URL"))
		if err != nil {
			return err
		}
		defer s.Close()
		addr := os.Getenv("HOLOCRON_LISTEN")
		if addr == "" {
			addr = "127.0.0.1:8080"
		}
		server := &http.Server{Addr: addr, Handler: transport.Handler(s), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
		done := make(chan error, 1)
		go func() { done <- server.ListenAndServe() }()
		slog.Info("holocron listening", "address", addr)
		select {
		case err = <-done:
			if err != http.ErrServerClosed {
				return err
			}
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return server.Shutdown(shutdown)
		}
		return nil
	default:
		return fmt.Errorf("usage: holocron [serve|migrate|provision|revoke|purge-domain]")
	}
}
