package platform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestDomainPurgeAndRestoreLedger(t *testing.T) {
	s := fixture(t)
	url := os.Getenv("HOLOCRON_ADMIN_DATABASE_URL")
	if url == "" {
		t.Skip("requires isolated admin fixture")
	}
	ctx := context.Background()
	tenant, domain, token := NewID(), NewID(), NewID()
	if err := Provision(ctx, url, ProvisionInput{Tenant: tenant, Domain: domain, ID: token, Owner: token, Role: "worker", Token: token, Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	c := invoke(t, s, token, "claim.publish", ClaimInput{Statement: "sensitive fixture", Scope: scope(), Kind: "finite_cases", Limitations: "private"})["id"].(string)
	// Include a cached response, which must not outlive deletion of its sources.
	invoke(t, s, token, "context.get", ContextInput{Scope: scope(), ByteBudget: 4096})
	pgBin := os.Getenv("PG_BIN")
	if pgBin == "" {
		pgBin = "/usr/lib/postgresql/17/bin"
	}
	dump := filepath.Join(t.TempDir(), "before-purge.sql")
	if out, err := exec.Command(filepath.Join(pgBin, "pg_dump"), "--dbname", url, "--file", dump).CombinedOutput(); err != nil {
		t.Fatalf("dump: %v %s", err, out)
	}
	if err := PurgeDomain(ctx, url, tenant, domain); err != nil {
		t.Fatal(err)
	}
	reject(t, s, token, "claim.get", IDInput{ID: c})
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	var count int
	for _, table := range []string{"claims", "assessments", "commands", "receipts", "outbox", "principals"} {
		if err = admin.QueryRow(ctx, "SELECT count(*) FROM holocron."+table+" WHERE tenant=$1 AND domain=$2", tenant, domain).Scan(&count); err != nil || count != 0 {
			t.Fatal("purge left payload", table, count, err)
		}
	}
	// Restore a pre-deletion backup into a different disposable database. A live
	// system must replay its externally retained purge ledger BEFORE serving it.
	restoreName := "restore_" + NewID()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+restoreName); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP DATABASE "+restoreName+" WITH (FORCE)")
	config, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.Database = restoreName
	// ConnString returns the original string, so set dbname using libpq environment.
	cmd := exec.Command(filepath.Join(pgBin, "psql"), "--no-psqlrc", "--set", "ON_ERROR_STOP=1", "--file", dump)
	cmd.Env = append(os.Environ(), "PGHOST="+config.Host, "PGPORT="+strconv.Itoa(int(config.Port)), "PGUSER="+config.User, "PGDATABASE="+restoreName)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("restore: %v %s", err, out)
	}
	restored, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close(ctx)
	if err = restored.QueryRow(ctx, `SELECT count(*) FROM holocron.claims WHERE tenant=$1 AND domain=$2`, tenant, domain).Scan(&count); err != nil || count != 1 {
		t.Fatal("backup fixture missing", count, err)
	}
	// Reconstruct a DSN without credentials in output. Test fixtures use socket trust.
	replayURL := "postgresql:///" + restoreName + "?host=" + config.Host + "&sslmode=disable"
	if err = PurgeDomain(ctx, replayURL, tenant, domain); err != nil {
		t.Fatal(err)
	}
	if err = restored.QueryRow(ctx, `SELECT count(*) FROM holocron.claims WHERE tenant=$1 AND domain=$2`, tenant, domain).Scan(&count); err != nil || count != 0 {
		t.Fatal("restore resurrected deleted claim", count, err)
	}
}
