package platform

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/002_hierarchy.sql
var hierarchyMigration string

const SchemaVersion = 2

// Migrate uses a separate administrative connection. The runtime refuses owner credentials.
func Migrate(ctx context.Context, url string) error {
	return migrateTo(ctx, url, SchemaVersion)
}
func migrateTo(ctx context.Context, url string, target int) error {
	c, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(8182052)`); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM pg_namespace WHERE nspname='holocron')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		var latest int
		if err = tx.QueryRow(ctx, `SELECT max(version) FROM holocron.schema_version`).Scan(&latest); err != nil {
			return err
		}
		if latest > SchemaVersion {
			return fmt.Errorf("database schema %d is newer than supported version %d", latest, SchemaVersion)
		}
		var digest string
		err = tx.QueryRow(ctx, `SELECT digest FROM holocron.schema_version WHERE version=1`).Scan(&digest)
		if err != nil {
			return err
		}
		if digest != Digest(Schema) {
			return fmt.Errorf("migration checksum differs; use a new database or explicit upgrade migration")
		}
	} else {
		if _, err = tx.Exec(ctx, Schema); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `CREATE TABLE holocron.schema_version(version integer PRIMARY KEY,digest text NOT NULL);`); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO holocron.schema_version VALUES(1,$1)`, Digest(Schema)); err != nil {
			return err
		}
	}
	if target >= 2 {
		var digest string
		err = tx.QueryRow(ctx, `SELECT digest FROM holocron.schema_version WHERE version=2`).Scan(&digest)
		if err == pgx.ErrNoRows {
			if _, err = tx.Exec(ctx, hierarchyMigration); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO holocron.schema_version VALUES(2,$1)`, Digest(hierarchyMigration)); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if digest != Digest(hierarchyMigration) {
			return fmt.Errorf("migration 2 checksum mismatch")
		}
	}
	return tx.Commit(ctx)
}

type ProvisionInput struct {
	Tenant, Domain, ID, Owner, Role, Token string
	Expires                                time.Time
	Curator                                bool
}

// SetCuration changes metadata authority only. It does not retract historical
// evidence or confer evidence-review authority. Old response replay is fenced.
func SetCuration(ctx context.Context, url, tenant, domain, id string, enabled bool) error {
	c, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE holocron.domains SET auth_epoch=auth_epoch+1 WHERE tenant=$1 AND id=$2`, tenant, domain); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO holocron.curation_grants(tenant,domain,principal,enabled) VALUES($1,$2,$3,$4) ON CONFLICT(tenant,domain,principal) DO UPDATE SET enabled=EXCLUDED.enabled`, tenant, domain, id, enabled); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO holocron.administrative_audit(tenant,domain,operation,target) VALUES($1,$2,$3,$4)`, tenant, domain, fmt.Sprintf("curation.set.%t", enabled), id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func Provision(ctx context.Context, url string, v ProvisionInput) error {
	if err := required(v.Tenant, v.Domain, v.ID, v.Owner, v.Role, v.Token); err != nil {
		return err
	}
	if !strings.Contains("|worker|validator|reviewer|admin|", "|"+v.Role+"|") || !v.Expires.After(time.Now()) {
		return fmt.Errorf("invalid role or expiry")
	}
	c, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO holocron.domains(tenant,id) VALUES($1,$2) ON CONFLICT DO NOTHING`, v.Tenant, v.Domain); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SELECT epoch FROM holocron.domains WHERE tenant=$1 AND id=$2 FOR UPDATE`, v.Tenant, v.Domain); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO holocron.principals(tenant,domain,id,owner,role,token_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, v.Tenant, v.Domain, v.ID, v.Owner, v.Role, Digest(v.Token), v.Expires)
	if err != nil {
		return err
	}
	if v.Curator {
		if _, err = tx.Exec(ctx, `INSERT INTO holocron.curation_grants(tenant,domain,principal) VALUES($1,$2,$3)`, v.Tenant, v.Domain, v.ID); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO holocron.administrative_audit(tenant,domain,operation,target) VALUES($1,$2,'principal.provision',$3)`, v.Tenant, v.Domain, v.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Revoke invalidates receipts atomically. Read-time assessment recomputes eligibility
// using current participant status even before the next graph write creates history.
func Revoke(ctx context.Context, url, tenant, domain, id string) error {
	c, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE holocron.domains SET epoch=epoch+1,auth_epoch=auth_epoch+1 WHERE tenant=$1 AND id=$2`, tenant, domain); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE holocron.principals SET revoked=true WHERE tenant=$1 AND domain=$2 AND id=$3`, tenant, domain, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("principal not found")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO holocron.administrative_audit(tenant,domain,operation,target) VALUES($1,$2,'principal.revoke',$3)`, tenant, domain, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PurgeDomain is an explicit administrative maintenance operation. It deletes
// the complete information boundary, including cached responses and derivations.
// The minimal prototype deliberately has no unsafe "delete just the source" API.
// Run during a maintenance window; retain/replay the deletion ledger on restore.
func PurgeDomain(ctx context.Context, url, tenant, domain string) error {
	if err := required(tenant, domain); err != nil {
		return err
	}
	c, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT epoch FROM holocron.domains WHERE tenant=$1 AND id=$2 FOR UPDATE`, tenant, domain); err != nil {
		return err
	}
	tables := []string{"association_reviews", "knowledge_associations", "edge_retractions", "knowledge_edges", "node_revisions", "knowledge_nodes", "node_kinds", "deliveries", "commands", "outcome_claims", "outcomes", "receipts", "attempts", "investigations", "opportunities", "assessments", "lifecycle", "reviews", "challenge_evidence", "challenges", "justification_evidence", "premises", "justifications", "evidence_parents", "evidence", "claims", "audit", "outbox"}
	for _, table := range tables {
		if _, err = tx.Exec(ctx, "ALTER TABLE holocron."+table+" DISABLE TRIGGER immutable"); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "DELETE FROM holocron."+table+" WHERE tenant=$1 AND domain=$2", tenant, domain); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "ALTER TABLE holocron."+table+" ENABLE TRIGGER immutable"); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM holocron.curation_grants WHERE tenant=$1 AND domain=$2`, tenant, domain); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM holocron.principals WHERE tenant=$1 AND domain=$2`, tenant, domain); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM holocron.domains WHERE tenant=$1 AND id=$2`, tenant, domain); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO holocron.administrative_audit(tenant,domain,operation,target) VALUES($1,$2,'domain.purge',$2)`, tenant, domain); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
