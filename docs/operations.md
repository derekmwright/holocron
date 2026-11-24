# Running the prototype

Requirements: Go 1.25 or newer and PostgreSQL 17 (the tested version). Dependencies are pinned in `go.mod` and `go.sum`. No external inference, vector extension, or broker is needed.

Build with `go build -buildvcs=false ./cmd/holocron` for a source snapshot without usable Git metadata. In a normal Git checkout, VCS stamping may be enabled.

## Test

```sh
go test ./...
bash scripts/test-integration.sh
go vet ./...
```

The first command runs unit/compile checks and explicitly skips database tests unless `HOLOCRON_TEST_DATABASE_URL` is set. The integration script starts a disposable PostgreSQL cluster in a private temporary directory, migrates it, provisions test identities, and runs the real PostgreSQL/HTTP/MCP tests with Go's race detector. It requires permission to create local sockets. Set `PG_BIN` if PostgreSQL binaries are elsewhere. The test credentials in that script must never be used for a deployed service.

The schema migration requires an administrative connection able to create a schema and the `holocron_app` role. The runtime refuses superuser, BYPASSRLS, and schema-table-owner connections. Migration checksums prevent silently reusing a differently shaped schema. This is the initial migration; future changes require explicit upgrade migrations.

## Local service

Create a dedicated PostgreSQL database and configure its authentication outside Holocron. Supply connection strings through environment variables, not command-line flags or tracked files:

```sh
export HOLOCRON_ADMIN_DATABASE_URL='postgresql://<migration-user>@localhost/holocron'
go run ./cmd/holocron migrate
# Pipe a provisioning JSON object through stdin; do not save secrets in the repo.
go run ./cmd/holocron provision < /secure/path/principal.json
export HOLOCRON_DATABASE_URL='postgresql://holocron_app@localhost/holocron'
go run ./cmd/holocron serve
```

Provisioning accepts `Tenant`, `Domain`, `ID`, `Owner`, `Role`, `Token`, and RFC3339 `Expires`. Roles are worker, validator, reviewer, and admin. Generate high-entropy random tokens and give each participant its own expiring credential. Only SHA-256 credential digests are stored. Provisioning and revocation are administrative CLI operations, unavailable to workers through MCP or REST. To rotate a credential in this prototype, provision a new principal/run identity and revoke the old one; revocation conservatively removes its evidence from eligible support.

The default listener is `127.0.0.1:8080`; `HOLOCRON_LISTEN` overrides it. Use an authenticated TLS boundary before exposing it beyond a trusted local machine. The server itself does not configure TLS, organization SSO, rate limiting, or production secret distribution. Keep the administrative database credential out of the runtime environment.

Set `HOLOCRON_TELEMETRY_STDOUT=1` for OpenTelemetry traces and periodic metrics on stderr. These contain operation names, duration, counts, and status, without prompts, artifact contents, or credentials. Outcome exports carry reported token/time/cost data separately. `healthz` is liveness, not a database readiness probe.

## Agent adapters

Merge [the Codex example](../examples/codex.toml) into the intended profile, or use [the Claude Code MCP example](../examples/claude.mcp.json) as project MCP configuration. Export `HOLOCRON_TOKEN` in the client environment. Copy the participation instructions from [the example](../examples/agent-instructions.md) into the project's appropriate agent instructions. The examples are not installed into user configuration automatically.

The MCP SDK transport integration test covers initialization, tools discovery, authenticated calls, state shared with REST, and role/tenant denial. Installed CLI versions observed during development: Codex 0.172.0 and Claude Code 2.1.330. Version discovery is not a live model workflow test. End-to-end model participation and interruption behavior must be measured with fresh sessions before claiming reliable adoption.

Do not depend on a Stop hook to capture all results. An externally run completion gate uses server receipts and outcome dispositions; interruption can leave an attempt without an outcome. A later owner report can classify it abandoned. The service cannot stop independently operated workers or prove they read the context.

## Events

Events are transactional PostgreSQL outbox rows. A consumer calls `events.poll`, handles each event durably, and then calls `events.ack`. Use a stable consumer name and deduplicate by event ID. Poll with a new request key to obtain fresh results. A crash before acknowledgement causes redelivery; there is no exactly-once claim. Consumers reconcile authoritative records by revision/epoch instead of treating event order as a complete state snapshot. No background broker dispatcher is required in this increment.

## Retention and restore

Ordinary operations cannot mutate or delete evidence/history. `record.retract` withdraws support while retaining its payload. Privacy deletion is a distinct administrative maintenance operation:

```sh
printf '%s\n' '{"Tenant":"tenant-id","Domain":"domain-id"}' |
  go run ./cmd/holocron purge-domain
```

This deletes **the complete visibility domain**, including principals, evidence, derivations, assessment history, receipts, cached responses, work outcomes, and outbox rows. It is deliberately broader than individual-record deletion because the prototype cannot prove that all copied free text has been identified. Run it during a maintenance window; administrative DDL takes table locks. A minimal administrative ledger retains domain identifiers, operation, database actor, and time, but no evidence text. This boundary must be approved as appropriate for the organization's retention requirements.

Backups and external artifacts are separate copies. Retain the purge ledger outside ordinary database backups and replay purges into an offline restored database before serving it. The integration suite tests restoring a pre-deletion backup and replaying a purge. Automatic external backup deletion, fine-grained redaction, and artifact-store deletion are not implemented. Do not ingest data needing those controls until they exist.

## Experiment export

Call `experiment.export` as a reviewer and store the JSON. It includes immutable work contracts, attempts, and reported outcomes, capped at 1000 rows per collection. Use `scripts/analyze-experiment.py` with independently scored outcomes to summarize correctness and total resource cost by arm. It does not perform significance testing or claim causal improvement. The confirmatory sample size and practical thresholds remain the preregistration decisions in [the validation plan](validation.md).
