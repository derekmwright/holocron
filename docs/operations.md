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

Schema migration requires an administrative connection able to create the schema and `holocron_app` role. The runtime refuses superuser, BYPASSRLS and table-owner/member connections. Numbered, checksum-verified migration 2 adds the hierarchy without editing migration 1. The current runtime requires schema version 2. Follow the [upgrade procedure](hierarchy-release.md#migration-and-compatibility) before replacing an existing runtime.

## Local service

For a persistent local development instance with a private Unix-socket PostgreSQL cluster, separate agent credentials, and a loopback server:

```sh
python3 scripts/local-dev.py start
python3 scripts/local-dev.py status
# When you want to stop it:
python3 scripts/local-dev.py stop
```

This keeps state under ignored `artifacts/local/` (directory mode 0700), with credentials in a mode-0600 file. It runs after the startup command exits; it is not an OS boot service. The local helper requires a Git checkout and embeds VCS provenance. Credentials expire after seven days. Existing state is retained across restarts; do not delete it to hide failed experiments.

To run one fresh, bounded real client session:

```sh
python3 scripts/live-agent.py codex publisher /path/to/prompt.txt
python3 scripts/live-agent.py claude validator /path/to/prompt.txt
```

The runner uses existing client authentication and configured models, a five-minute timeout, and a $3 reported-API-cost ceiling for each Claude run. Codex does not expose a comparable cost ceiling here. Logs and raw usage remain in ignored per-run directories. Codex receives an explicit role-specific Holocron tool allowlist and approval policy; no user-wide configuration is changed. Tool names contain dots, so CLI approval overrides use a literal-key TOML table rather than splitting the tool name as a configuration path. The probe uses a worker test subject separately from the submitting observer and checks `session.describe` against the pinned source/build scope.

These are opt-in live model invocations and consume the connected account's usage. A successful process exit alone is not a successful investigation; inspect tool outcomes and server assessments. See the pilot report for actual outcomes.

Create a dedicated PostgreSQL database and configure its authentication outside Holocron. Supply connection strings through environment variables, not command-line flags or tracked files:

```sh
export HOLOCRON_ADMIN_DATABASE_URL='postgresql://<migration-user>@localhost/holocron'
go run ./cmd/holocron migrate
# Pipe a provisioning JSON object through stdin; do not save secrets in the repo.
go run ./cmd/holocron provision < /secure/path/principal.json
export HOLOCRON_DATABASE_URL='postgresql://holocron_app@localhost/holocron'
go run ./cmd/holocron serve
```

Provisioning accepts `Tenant`, `Domain`, `ID`, `Owner`, `Role`, `Token`, RFC3339 `Expires`, and optional `Curator`. Roles are worker, validator, reviewer and admin. Curation is a separate explicit capability, not a reviewer privilege. Generate high-entropy tokens and give each participant its own credential; only digests are stored. Administrative `set-curation` changes metadata authority and fences old cached responses. Credential expiry blocks authentication without erasing history. Deliberate `revoke` is a principal quarantine that removes its evidence/reviews from eligible support; do not use it as a harmless routine token-expiry operation.

The default listener is `127.0.0.1:8080`; `HOLOCRON_LISTEN` overrides it. Use an authenticated TLS boundary before exposing it beyond a trusted local machine. The server itself does not configure TLS, organization SSO, rate limiting, or production secret distribution. Keep the administrative database credential out of the runtime environment.

Set `HOLOCRON_TELEMETRY_STDOUT=1` for OpenTelemetry traces and periodic metrics on stderr. These contain operation names, duration, counts, and status, without prompts, artifact contents, or credentials. Outcome exports carry reported token/time/cost data separately. `healthz` is liveness, not a database readiness probe.

## Agent adapters

Merge [the Codex example](../examples/codex.toml) into the intended profile, or use [the Claude Code MCP example](../examples/claude.mcp.json) as project MCP configuration. Export `HOLOCRON_TOKEN` in the client environment. Copy the participation instructions from [the example](../examples/agent-instructions.md) into the project's appropriate agent instructions. The examples are not installed into user configuration automatically.

The MCP SDK transport integration test covers initialization, tools discovery, authenticated calls, state shared with REST, and role/tenant denial. The [first live pilot](live-pilot.md) additionally exercised actual Codex 0.172.0 and Claude Code 2.1.330 sessions. Interruption, outage behavior, and wider workloads still need evaluation before claiming reliable adoption.

For hierarchical work, follow [Wadjet onboarding](wadjet-bootstrap.md) and the [single-domain release guide](hierarchy-release.md). The dedicated `live-agent.py` helper remains an HTTP-guard smoke harness; its automatically constructed Holocron scope is not a Wadjet execution scope. Normal Wadjet agents should run in the Wadjet checkout with the shared MCP connection and returned working IDs.

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
