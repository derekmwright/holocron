# First live agent pilot

On 2026-11-24, twelve fresh real model sessions exercised a persistent local Holocron instance: seven Codex sessions configured with `gpt-6-astra` and five Claude Code sessions resolving the configured `fable` alias to `claude-fable-5-2`. Client versions were Codex 0.172.0 and Claude Code 2.1.330. These were actual client invocations and MCP calls, not the scripted integration-test participants.

The basic evidence workflow held, while the exercise exposed adoption and provenance gaps. The first reviewer declined to establish the initial claim. After correcting the measurement method and recording a successor, a separate reviewer accepted only the bounded finding. A fresh consumer reused it without rerunning the investigation and explicitly distinguished its own inference from the observed cases.

## Running instance

The service remains on `http://127.0.0.1:8080`, with `/healthz` and `/mcp`. State, credentials, observations, raw client logs, and usage records live in ignored `artifacts/local/`. The running build is source revision `872b0d702a58b9ddb7d8fa91d784f0394a4b4362`, with `modified=false`, and executable digest `sha256:6053fd3ae955ee5a4154c364d725772d1578bc4558ce5d4bfb9bc4c68fdd110a`. An operator check independently hashed the local executable and matched the service description.

The server is a background local process, not an OS boot service. Use `python3 scripts/local-dev.py status` to check it and `stop` when finished. Separate expiring credentials were provisioned for publisher, validator, reviewer, consumer, challenger, and an isolated tenant. All share the same human operator; this is not a claim of organizational independence.

## Observed sequence

1. Claude successfully discovered the MCP tools and retrieved empty context without manufacturing a finding.
2. Two Codex publisher attempts ran the HTTP checks but could not use MCP because approval was required under the noninteractive policy. Both reported that nothing had been published; the failed tool calls remain in the logs.
3. Supplying the authorized per-tool policy as a literal-name TOML table fixed Codex access. CLI dotted-key paths had not applied the settings for names such as `context.get`. The runner now uses a role-specific tool allowlist and leaves user-wide configuration unchanged. The supported policy fields are documented in the [Codex configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference).
4. Claude published the initial finite-case claim and one actual execution manifest. It did not claim verification. Because evidence submission alone creates no claim link, the validator needed an explicit coordinator handoff to find the manifest. Participation instructions now require a provisional justification immediately after evidence submission.
5. A separate Claude validation session executed the same checks, recorded a distinct artifact/root, disclosed shared method/oracle/environment/operator, and proposed a two-execution justification.
6. A Codex challenger found that the evidence did not establish the caller's claimed worker role. A separate Codex reviewer checked the artifacts and also could not establish the running-server build. It recorded an **inconclusive** review, leaving the claim insufficient and the methodology challenge pending.
7. Added authenticated `session.describe`, which exposes the current principal/role and the running binary's embedded source revision, dirty flag, and digest. The probe now records the worker test subject separately from the observing validator, the actual denial response, and the pinned build. Probe fixture claims use a separate case scope so they do not crowd the substantive context.
8. Codex published a successor claim with the new observations and a single-observation provisional justification. It remained insufficient. Claude independently reran the revised probe and proposed the two-execution alternative.
9. A fresh Codex reviewer inspected both observation files, hashes, scope, roles, build descriptions, and statuses. It accepted the two-execution justification. The single-observation alternative remained ineligible.
10. A fresh Claude consumer retrieved the supported revision and chose to reuse a key for the tested identical-payload retry and use a new key for a changed payload. It cited the evidence, did not rerun the probe, and explicitly warned that a lost-response or in-flight concurrent retry was not itself tested.

## Directly verified final state

| Record | Observed state at epoch 19 |
| --- | --- |
| Initial claim `25a244eda16cf446df2f53ffaf10584b` | Insufficient, revalidation required, superseded; methodology challenge remains pending. |
| Initial challenge `fbf0eb9cebfa8dfffc42255f4ea757ac` | Preserved against the original justification. No fabricated counterexample or actual authorization bypass was asserted. |
| Successor claim `3974e88db234bbf9127d52fba3ebc72e` | Supported, current, active, no pending challenge on this revision. |
| Accepted justification `ef255e6605162fb0a86096e57d1ca941` | Two eligible execution manifests: `12b1e1830c240c145c604eaf7cc709d2` and `b27b2e261f7bd7741be8366dd2631109`. |

The revised executions recorded statuses `200, 200, 409, 403`: identical request retries produced identical bodies, changed-payload reuse conflicted, and a worker review was denied. An additional operator test used an existing justification rather than a nonexistent target and still received `403`. A different tenant's request for the claim received `404`.

The new endpoint and existing invariants passed the real PostgreSQL race-enabled suite: 19 Go test functions plus their subcases, and two experiment-analysis tests. `go vet` passed. The server's final health response was `alive`.

## Limits and follow-up

This is a live workflow smoke test, not a controlled productivity experiment or a security audit. It covers three sequential REST cases and a small reasoning task. It does not validate generalized SQL behavior, Wadjet workload transfer, formal correctness, independent methodology, concurrency safety under load, or agent resistance to arbitrary stored prompt injection. Identity/build descriptions come from the server; they are not remote cryptographic attestation.

The fresh consumer spotted another authoring problem: the immutable claim prose still said it was not an established finding even after its structured assessment became supported. Participation guidance now keeps temporary assessment labels out of proposition text. Existing wording is preserved rather than silently rewritten. This instruction change still needs evaluation in subsequent authoring sessions.

Other observed adoption costs remain visible: two failed Codex launches, two Claude compound-shell permission denials followed by permitted commands, a manual evidence-ID handoff in the first round, and fixture noise before the probe scope was separated. These are costs to count, not discard as unsuccessful trials.

Raw per-run measurements are retained in `artifacts/local/run-summary.json`. Summed client elapsed time was 526.88 seconds across all twelve sessions, including overlaps and failed attempts; this is not wall-clock implementation time. Codex reported 996,857 cumulative input tokens, of which 833,408 were cached, across its seven sessions. These totals count context processed on successive model calls, not unique source text. Claude reported approximately $1.8462 in list-equivalent API cost across five sessions; that is not a claim about incremental subscription billing. Codex dollar cost was not available, and there is no baseline arm from which to infer savings.

The next useful step is a held-out Wadjet task with a frozen corpus, fixed models, independently scored correctness, and explicit acquisition/review cost. The present result supports further testing of the workflow, not scaling claims.
