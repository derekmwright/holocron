-- The migration is applied by an administrative connection, never the runtime role.
CREATE SCHEMA holocron;
SET search_path = holocron, public;
CREATE TABLE domains (
 tenant text NOT NULL, id text NOT NULL, epoch bigint NOT NULL DEFAULT 0,
 PRIMARY KEY(tenant,id)
);
-- Administrative records are not granted to the runtime role and contain no payloads.
CREATE TABLE administrative_audit (
 seq bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 actor text NOT NULL DEFAULT current_user, tenant text NOT NULL, domain text NOT NULL,
 operation text NOT NULL, target text NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE principals (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, owner text NOT NULL,
 role text NOT NULL CHECK(role IN ('worker','validator','reviewer','admin')),
 token_hash text NOT NULL UNIQUE, expires_at timestamptz NOT NULL,
 revoked boolean NOT NULL DEFAULT false,
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain) REFERENCES domains(tenant,id)
);
CREATE FUNCTION authenticate(token_digest text)
RETURNS TABLE(tenant text, domain text, id text, owner text, role text)
LANGUAGE sql SECURITY DEFINER SET search_path = holocron, pg_temp AS $$
 SELECT p.tenant,p.domain,p.id,p.owner,p.role FROM principals p
 WHERE p.token_hash=token_digest AND NOT p.revoked AND p.expires_at>clock_timestamp()
$$;
REVOKE ALL ON FUNCTION authenticate(text) FROM PUBLIC;
CREATE TABLE claims (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL,
 author text NOT NULL, previous text, statement text NOT NULL, body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 search tsvector GENERATED ALWAYS AS (to_tsvector('english',statement)) STORED,
 PRIMARY KEY(tenant,domain,id), UNIQUE(tenant,domain,previous),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,previous) REFERENCES claims(tenant,domain,id)
);
CREATE INDEX claims_search ON claims USING gin(search);
CREATE TABLE evidence (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 root text NOT NULL, body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE evidence_parents (
 tenant text NOT NULL, domain text NOT NULL, evidence text NOT NULL, parent text NOT NULL,
 PRIMARY KEY(tenant,domain,evidence,parent), CHECK(evidence<>parent),
 FOREIGN KEY(tenant,domain,evidence) REFERENCES evidence(tenant,domain,id),
 FOREIGN KEY(tenant,domain,parent) REFERENCES evidence(tenant,domain,id)
);
CREATE TABLE justifications (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 claim text NOT NULL, body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,claim) REFERENCES claims(tenant,domain,id)
);
CREATE TABLE premises (
 tenant text NOT NULL, domain text NOT NULL, justification text NOT NULL, claim text NOT NULL,
 PRIMARY KEY(tenant,domain,justification,claim),
 FOREIGN KEY(tenant,domain,justification) REFERENCES justifications(tenant,domain,id),
 FOREIGN KEY(tenant,domain,claim) REFERENCES claims(tenant,domain,id)
);
CREATE INDEX premises_reverse ON premises(tenant,domain,claim);
CREATE TABLE justification_evidence (
 tenant text NOT NULL, domain text NOT NULL, justification text NOT NULL, evidence text NOT NULL,
 PRIMARY KEY(tenant,domain,justification,evidence),
 FOREIGN KEY(tenant,domain,justification) REFERENCES justifications(tenant,domain,id),
 FOREIGN KEY(tenant,domain,evidence) REFERENCES evidence(tenant,domain,id)
);
CREATE TABLE challenges (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 claim text NOT NULL, justification text, body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,claim) REFERENCES claims(tenant,domain,id),
 FOREIGN KEY(tenant,domain,justification) REFERENCES justifications(tenant,domain,id)
);
CREATE TABLE challenge_evidence (
 tenant text NOT NULL, domain text NOT NULL, challenge text NOT NULL, evidence text NOT NULL,
 PRIMARY KEY(tenant,domain,challenge,evidence),
 FOREIGN KEY(tenant,domain,challenge) REFERENCES challenges(tenant,domain,id),
 FOREIGN KEY(tenant,domain,evidence) REFERENCES evidence(tenant,domain,id)
);
CREATE TABLE reviews (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 justification text, challenge text, body jsonb NOT NULL,
 seq bigint GENERATED ALWAYS AS IDENTITY,
 CHECK ((justification IS NOT NULL)::int + (challenge IS NOT NULL)::int = 1),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,justification) REFERENCES justifications(tenant,domain,id),
 FOREIGN KEY(tenant,domain,challenge) REFERENCES challenges(tenant,domain,id)
);
CREATE TABLE lifecycle (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 claim text, evidence text, state text NOT NULL CHECK(state IN ('retracted','superseded','redacted')),
 reason text NOT NULL, seq bigint GENERATED ALWAYS AS IDENTITY,
 CHECK ((claim IS NOT NULL)::int + (evidence IS NOT NULL)::int = 1),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,claim) REFERENCES claims(tenant,domain,id),
 FOREIGN KEY(tenant,domain,evidence) REFERENCES evidence(tenant,domain,id)
);
CREATE TABLE assessments (
 tenant text NOT NULL, domain text NOT NULL, claim text NOT NULL, epoch bigint NOT NULL,
 body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,claim,epoch), FOREIGN KEY(tenant,domain,claim) REFERENCES claims(tenant,domain,id)
);
CREATE TABLE opportunities (
 tenant text NOT NULL, domain text NOT NULL, claim text NOT NULL, epoch bigint NOT NULL,
 reason text NOT NULL, PRIMARY KEY(tenant,domain,claim,epoch),
 FOREIGN KEY(tenant,domain,claim,epoch) REFERENCES assessments(tenant,domain,claim,epoch)
);
CREATE TABLE investigations (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE attempts (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 investigation text NOT NULL, body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,investigation) REFERENCES investigations(tenant,domain,id)
);
CREATE TABLE receipts (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 epoch bigint NOT NULL, body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE outcomes (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 attempt text NOT NULL, receipt text NOT NULL, body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), UNIQUE(tenant,domain,attempt),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,attempt) REFERENCES attempts(tenant,domain,id),
 FOREIGN KEY(tenant,domain,receipt) REFERENCES receipts(tenant,domain,id)
);
CREATE TABLE outcome_claims (
 tenant text NOT NULL, domain text NOT NULL, outcome text NOT NULL, claim text NOT NULL,
 PRIMARY KEY(tenant,domain,outcome,claim),
 FOREIGN KEY(tenant,domain,outcome) REFERENCES outcomes(tenant,domain,id),
 FOREIGN KEY(tenant,domain,claim) REFERENCES claims(tenant,domain,id)
);
CREATE TABLE commands (
 tenant text NOT NULL, domain text NOT NULL, author text NOT NULL, operation text NOT NULL,
 key text NOT NULL, digest text NOT NULL, response bytea NOT NULL,
 PRIMARY KEY(tenant,domain,author,operation,key), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE audit (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 operation text NOT NULL, outcome text NOT NULL, epoch bigint NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE outbox (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, epoch bigint NOT NULL,
 operation text NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), FOREIGN KEY(tenant,domain) REFERENCES domains(tenant,id)
);
CREATE TABLE deliveries (
 tenant text NOT NULL, domain text NOT NULL, consumer text NOT NULL, event text NOT NULL,
 PRIMARY KEY(tenant,domain,consumer,event), FOREIGN KEY(tenant,domain,event) REFERENCES outbox(tenant,domain,id)
);
CREATE FUNCTION immutable_record() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'append-only record'; END $$;

DO $$ DECLARE t text; BEGIN
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='holocron_app') THEN CREATE ROLE holocron_app LOGIN NOSUPERUSER NOBYPASSRLS; END IF;
 FOREACH t IN ARRAY ARRAY['claims','evidence','evidence_parents','justifications','premises','justification_evidence','challenges','challenge_evidence','reviews','lifecycle','assessments','opportunities','investigations','attempts','receipts','outcomes','outcome_claims','commands','audit','outbox','deliveries'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY boundary ON %I USING (tenant=current_setting(''holocron.tenant'',true) AND domain=current_setting(''holocron.domain'',true)) WITH CHECK (tenant=current_setting(''holocron.tenant'',true) AND domain=current_setting(''holocron.domain'',true))',t);
  EXECUTE format('CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION immutable_record()',t);
  EXECUTE format('GRANT SELECT,INSERT ON %I TO holocron_app',t);
 END LOOP;
END $$;
ALTER TABLE domains ENABLE ROW LEVEL SECURITY;
ALTER TABLE domains FORCE ROW LEVEL SECURITY;
CREATE POLICY boundary ON domains USING (tenant=current_setting('holocron.tenant',true) AND id=current_setting('holocron.domain',true));
GRANT USAGE ON SCHEMA holocron TO holocron_app;
GRANT SELECT,UPDATE ON domains TO holocron_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA holocron TO holocron_app;
GRANT EXECUTE ON FUNCTION authenticate(text) TO holocron_app;
-- Participant metadata is exposed only within the authenticated boundary.
CREATE FUNCTION participants() RETURNS TABLE(id text,owner text,role text,revoked boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=holocron,pg_temp AS $$
 SELECT id,owner,role,revoked OR expires_at<=clock_timestamp() FROM principals
 WHERE tenant=current_setting('holocron.tenant',true) AND domain=current_setting('holocron.domain',true)
$$;
REVOKE ALL ON FUNCTION participants() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION participants() TO holocron_app;
