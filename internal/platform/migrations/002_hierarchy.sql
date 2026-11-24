SET search_path=holocron,public;
ALTER TABLE domains ADD COLUMN metadata_epoch bigint NOT NULL DEFAULT 0;
ALTER TABLE domains ADD COLUMN auth_epoch bigint NOT NULL DEFAULT 0;
ALTER TABLE commands ADD COLUMN auth_epoch bigint NOT NULL DEFAULT 0;
ALTER TABLE receipts ADD COLUMN metadata_epoch bigint NOT NULL DEFAULT 0;
ALTER TABLE receipts ADD COLUMN auth_epoch bigint NOT NULL DEFAULT 0;
ALTER TABLE outbox ADD COLUMN metadata_epoch bigint NOT NULL DEFAULT 0;

CREATE TABLE curation_grants (
 tenant text NOT NULL, domain text NOT NULL, principal text NOT NULL,
 enabled boolean NOT NULL DEFAULT true,
 PRIMARY KEY(tenant,domain,principal),
 FOREIGN KEY(tenant,domain,principal) REFERENCES principals(tenant,domain,id)
);
CREATE FUNCTION can_curate(actor text) RETURNS boolean
LANGUAGE sql SECURITY DEFINER SET search_path=holocron,pg_temp AS $$
 SELECT EXISTS(SELECT FROM curation_grants g JOIN principals p
 ON (p.tenant,p.domain,p.id)=(g.tenant,g.domain,g.principal)
 WHERE g.tenant=current_setting('holocron.tenant',true)
 AND g.domain=current_setting('holocron.domain',true)
 AND g.principal=actor AND g.enabled AND NOT p.revoked AND p.expires_at>clock_timestamp())
$$;
REVOKE ALL ON FUNCTION can_curate(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION can_curate(text) TO holocron_app;

CREATE TABLE node_kinds (
 tenant text NOT NULL, domain text NOT NULL, category text NOT NULL CHECK(category IN ('organization','concept')),
 kind text NOT NULL, PRIMARY KEY(tenant,domain,category,kind),
 FOREIGN KEY(tenant,domain) REFERENCES domains(tenant,id)
);
CREATE TABLE knowledge_nodes (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 category text NOT NULL, kind text NOT NULL, body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id),
 FOREIGN KEY(tenant,domain,category,kind) REFERENCES node_kinds(tenant,domain,category,kind),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE node_revisions (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 node text NOT NULL, version integer NOT NULL CHECK(version>=2), body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), UNIQUE(tenant,domain,node,version),
 FOREIGN KEY(tenant,domain,node) REFERENCES knowledge_nodes(tenant,domain,id),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE knowledge_edges (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 relation text NOT NULL CHECK(relation IN ('contains','broader','related','affiliated')),
 source text NOT NULL, target text NOT NULL, body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), CHECK(source<>target),
 FOREIGN KEY(tenant,domain,source) REFERENCES knowledge_nodes(tenant,domain,id),
 FOREIGN KEY(tenant,domain,target) REFERENCES knowledge_nodes(tenant,domain,id),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE INDEX knowledge_edges_forward ON knowledge_edges(tenant,domain,relation,source);
CREATE INDEX knowledge_edges_reverse ON knowledge_edges(tenant,domain,relation,target);
CREATE TABLE edge_retractions (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 edge text NOT NULL, body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id), UNIQUE(tenant,domain,edge),
 FOREIGN KEY(tenant,domain,edge) REFERENCES knowledge_edges(tenant,domain,id),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE TABLE knowledge_associations (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 kind text NOT NULL CHECK(kind IN ('claim_concept','entity_concept','placement')),
 claim text, entity text, target text NOT NULL, steward text, body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK((claim IS NOT NULL)::int+(entity IS NOT NULL)::int=1),
 CHECK((kind='entity_concept')=(entity IS NOT NULL)),
 CHECK((kind='placement')=(steward IS NOT NULL)),
 PRIMARY KEY(tenant,domain,id),
 FOREIGN KEY(tenant,domain,claim) REFERENCES claims(tenant,domain,id),
 FOREIGN KEY(tenant,domain,entity) REFERENCES knowledge_nodes(tenant,domain,id),
 FOREIGN KEY(tenant,domain,target) REFERENCES knowledge_nodes(tenant,domain,id),
 FOREIGN KEY(tenant,domain,steward) REFERENCES principals(tenant,domain,id),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE INDEX associations_claim ON knowledge_associations(tenant,domain,claim);
CREATE INDEX associations_concept ON knowledge_associations(tenant,domain,target);
CREATE TABLE association_reviews (
 tenant text NOT NULL, domain text NOT NULL, id text NOT NULL, author text NOT NULL,
 association text NOT NULL, body jsonb NOT NULL, seq bigint GENERATED ALWAYS AS IDENTITY,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant,domain,id),
 FOREIGN KEY(tenant,domain,association) REFERENCES knowledge_associations(tenant,domain,id),
 FOREIGN KEY(tenant,domain,author) REFERENCES principals(tenant,domain,id)
);
CREATE INDEX association_review_latest ON association_reviews(tenant,domain,association,seq DESC);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['node_kinds','knowledge_nodes','node_revisions','knowledge_edges','edge_retractions','knowledge_associations','association_reviews'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY boundary ON %I USING (tenant=current_setting(''holocron.tenant'',true) AND domain=current_setting(''holocron.domain'',true)) WITH CHECK (tenant=current_setting(''holocron.tenant'',true) AND domain=current_setting(''holocron.domain'',true))',t);
  EXECUTE format('CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION immutable_record()',t);
  EXECUTE format('GRANT SELECT,INSERT ON %I TO holocron_app',t);
 END LOOP;
END $$;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA holocron TO holocron_app;
CREATE FUNCTION runtime_schema_version() RETURNS TABLE(schema_version integer)
LANGUAGE sql SECURITY DEFINER SET search_path=holocron,pg_temp AS $$ SELECT max(version) FROM schema_version $$;
REVOKE ALL ON FUNCTION runtime_schema_version() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION runtime_schema_version() TO holocron_app;

-- Authentication expiry prevents new calls; it does not retract observations
-- made while authenticated. Explicit principal revocation remains quarantine.
CREATE OR REPLACE FUNCTION participants() RETURNS TABLE(id text,owner text,role text,revoked boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=holocron,pg_temp AS $$
 SELECT p.id,p.owner,p.role,p.revoked FROM principals p
 WHERE p.tenant=current_setting('holocron.tenant',true) AND p.domain=current_setting('holocron.domain',true)
$$;

-- Preserve old responses as historical snapshots, but require fresh context for
-- completion after the metadata/applicability evaluator upgrade.
UPDATE domains SET metadata_epoch=metadata_epoch+1;
INSERT INTO administrative_audit(tenant,domain,operation,target)
 SELECT tenant,id,'schema.migrate','2' FROM domains;
