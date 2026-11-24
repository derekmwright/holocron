package platform

import (
	"context"
	"encoding/json"
	neturl "net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestMigrationPreservesV1RecordsAndReplay(t *testing.T) {
	adminURL := os.Getenv("HOLOCRON_ADMIN_DATABASE_URL")
	if adminURL == "" {
		t.Skip("requires disposable integration cluster")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	name := "migration_" + NewID()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
	u, err := neturl.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	url := u.String()
	if err = migrateTo(ctx, url, 1); err != nil {
		t.Fatal(err)
	}
	legacy, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer legacy.Close(ctx)
	if _, err = legacy.Exec(ctx, `INSERT INTO holocron.domains(tenant,id) VALUES('legacy','legacy')`); err != nil {
		t.Fatal(err)
	}
	if _, err = legacy.Exec(ctx, `INSERT INTO holocron.principals(tenant,domain,id,owner,role,token_hash,expires_at) VALUES('legacy','legacy','actor','owner','worker',$1,$2)`, Digest("legacy-token"), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	id := NewID()
	body := encode(ClaimInput{Statement: "Legacy untouched", Scope: scope(), Kind: "finite_cases", Limitations: "Finite"})
	if _, err = legacy.Exec(ctx, `INSERT INTO holocron.claims(tenant,domain,id,author,statement,body) VALUES('legacy','legacy',$1,'actor','Legacy untouched',$2)`, id, body); err != nil {
		t.Fatal(err)
	}
	var before string
	if err = legacy.QueryRow(ctx, `SELECT body::text FROM holocron.claims WHERE id=$1`, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	request := encode(IDInput{ID: id})
	var value any
	_ = json.Unmarshal(request, &value)
	response := []byte(`{"legacy": true, "snapshot": 1}`)
	if _, err = legacy.Exec(ctx, `INSERT INTO holocron.commands VALUES('legacy','legacy','actor','claim.get','legacy-key',$1,$2)`, Digest(string(encode(value))), response); err != nil {
		t.Fatal(err)
	}
	app := *u
	app.User = neturl.User("holocron_app")
	if service, e := Open(ctx, app.String()); e == nil {
		service.Close()
		t.Fatal("new runtime accepted v1 schema without migration")
	}
	if err = Migrate(ctx, url); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(ctx, url); err != nil {
		t.Fatal("migration retry", err)
	}
	var after string
	if err = legacy.QueryRow(ctx, `SELECT body::text FROM holocron.claims WHERE id=$1`, id).Scan(&after); err != nil || before != after {
		t.Fatal("migration rewrote immutable record", err)
	}
	service, err := Open(ctx, app.String())
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	replay, err := service.Execute(ctx, "legacy-token", "claim.get", "legacy-key", request)
	if err != nil || string(replay) != string(response) {
		t.Fatalf("legacy idempotency lost: %s %v", replay, err)
	}
	var version int
	if err = legacy.QueryRow(ctx, `SELECT max(version) FROM holocron.schema_version`).Scan(&version); err != nil || version != 2 {
		t.Fatal("wrong migration version", version, err)
	}
	if _, err = legacy.Exec(ctx, `UPDATE holocron.schema_version SET digest='tampered' WHERE version=2`); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(ctx, url); err == nil {
		t.Fatal("checksum mismatch ignored")
	}
}
