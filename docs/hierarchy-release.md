# Single-domain hierarchy release

Implemented on 2026-11-24 within the founder-approved boundary: hierarchical knowledge organization, explicitly bounded broader applicability, and qualified retrieval inside one visibility domain. [ADR 018](hierarchy-decisions.md#adr-018-single-domain-implementation-boundaries) records the concrete choices. [Wadjet onboarding](wadjet-bootstrap.md) is the next practical step; this release makes no claim that hierarchy improves agent performance.

## Implemented capabilities

* Stable organizational entities and concepts with tenant-defined kinds. A new kind is data, not a migration. Names are not identifiers.
* Multiple-parent organizational containment and broader-concept DAGs, plus non-transitive related/affiliated links. Cycles are rejected transactionally; diamond paths deduplicate.
* Immutable node revision and relationship history. `edge.add` with `replace` moves a child atomically; renames preserve identity. `hierarchy.get` exposes initial metadata, node revisions, and retired containment edges.
* Proposed and curator-reviewed entity classifications, claim classifications, and placements. Curation is an explicit administrative grant, independent of the evidence-reviewer role. Classification never changes evidence posture.
* One accepted home per claim revision, with an explicit steward and atomic transfer. `family_id` derives from the preserved revision chain, but a successor does not inherit an accepted placement/classification. This prevents borrowing approval for a newly broadened assertion.
* Applicability envelope v2: either explicitly unknown or a bounded list of complete execution contexts. Missing dimensions are never wildcards. Existing exact scope remains supported unchanged as envelope v1.
* `software-context-matrix/v1` for positively established finite multi-context claims. Every named context needs qualifying reproduction. A bounded generalization additionally needs explicit assumptions, premise claims, an inference kind and a reviewed warrant. Universal principles and analogy-only support remain unestablished.
* Qualified hierarchy, concept-only/lateral, and flat retrieval. Cards retain posture, conditions, limitations, source references, challenges, local applicability and retrieval reasons. Multiple paths yield one revision, not more support.
* Separate knowledge, metadata and authorization versions. Organizational edits do not create evidence assessments. Metadata changes require fresh completion context; authorization changes fence saved-response replay without repeating a mutation.
* Referenced historical sources can be retained on hypotheses with a pinned revision, fragment and digest. Source references alone never count as validated execution evidence.

Organizational selectors are discovery context. A component association is not evidence that arbitrary operations on that component are correct. `local_applicability: matched` means only that the explicitly supplied finite conditions fit a declared envelope; evidence posture and dependency health still determine whether that knowledge is supported.

## API additions

All operations use the existing REST/MCP authentication and idempotency protocol. Credentials still select exactly one tenant/domain.

| Operation | Behavior |
| --- | --- |
| `node.create` | Curator creates category `organization` or `concept`, custom `kind`, name, description, optional repository identity. |
| `node.revise` | Curator changes descriptive metadata using `expected_version`; identity/category/kind remain stable. |
| `edge.add` | Curator adds `contains`, `broader`, `related`, or `affiliated`. Optional `replace` retires an edge for the same relation and child atomically. |
| `edge.retract` | Curator retires an edge with a reason. |
| `association.propose` | Any authenticated contributor proposes `claim_concept`, `entity_concept`, or `placement`; no curated retrieval authority yet. |
| `association.review` | Explicit curator accepts/rejects the proposal. Evidence reviewer authority alone is insufficient. |
| `placement.transfer` | Curator changes the accepted home/steward inside the same compartment; no grant or factual scope is changed. |
| `hierarchy.get` | Bounded current metadata and node/edge history, with limit/truncation reporting. |
| `context.resolve` | Validates supplied organizational IDs, containment, conceptual IDs and repository identity. Does not grant access or establish applicability. |

`session.describe` reports `capabilities.curate`, schema version and available verification policies. `claim.get` exposes the family ID, revision-specific classifications, accepted home and metadata version.

Example working context after bootstrap (values must be returned IDs):

```json
{
  "orientation_only": true,
  "working": {
    "organization": "organization-node-id",
    "project": "wadjet-project-node-id",
    "repository": "wadjet-repository-node-id",
    "components": ["hash-aggregation-node-id"],
    "concepts": ["memory-concept-id", "spill-concept-id"]
  },
  "query": "hash aggregation spill correctness",
  "byte_budget": 16000
}
```

Do not invent an execution environment merely to retrieve orientation. Omit `scope` and use `orientation_only: true`; its receipt cannot complete an execution-scoped attempt. Once actual conditions are known, provide the complete legacy `scope` and the working context. A concept-only working context permits lateral discovery without organizational ancestry.

`discovery` can be `hierarchical`, `flat`, or `legacy`. Working identifiers or orientation-only requests default to hierarchical. Text queries, objectives, or explicit claim references default to flat discovery even without working identifiers. Only scope-only requests retain the legacy default; callers can still explicitly request `legacy`. Natural-language queries use all-term matching plus length-normalized term overlap, labeled as partial matches where appropriate. This changes discovery, not evidence strength or applicability.

Default packing is adaptive: explicit requested revisions first, then space for one complete relevant orientation card, then matched cards, and remaining qualified orientation. Unused matched capacity can be borrowed. An explicitly supplied `orientation_percent` remains a hard cap, including zero. Full cards are never shortened by stripping qualifications. Small navigation-only `deferred` references identify omitted claims and require `claim.get`; they are not delivered claims for `applied`/`considered` reporting. Retrieve a fresh full-card context receipt before reporting their use.

`diagnostics` reports effective discovery, candidate count, scope/relevance exclusions, budget/orientation omissions and omitted work context. `truncated` alone does not identify which category was omitted. Counts cover only the authenticated compartment. Too small a budget produces an explicit envelope-size error rather than an invalid receipt. The limit still counts serialized Holocron JSON, not the MCP wrapper, tool catalog or vendor-specific model tokens.

For completion, `applied` IDs must have been delivered as locally matched. Orientation or analogical candidates can be recorded as `considered`; they cannot masquerade as locally established premises. This checks declared usage, not the agent's private reasoning. Independent task correctness remains externally adjudicated.

## Broader applicability and generalization

Use `applicability` instead of the legacy `scope` when publishing an envelope-v2 claim:

```json
{
  "version": 2,
  "state": "unknown",
  "description": "Potential applicability beyond the observed repository; not established"
}
```

Unknown applicability is permitted on hypotheses/generalizations, not on a `finite_cases` assertion. An `enumerated` envelope has 1–16 complete `contexts`, each using the existing repository/commit/environment/configuration/dataset/oracle/cases fields. All must be fully specified and distinct. Explicitly unrestricted predicates are not implemented.

For a supported bounded generalization, use claim kind `generalization`, nonempty assumptions and limitations, and a justification with `inference_kind: bounded_generalization`, actual premise IDs, evidence and a scope warrant. Every enumerated context must have a distinct authorized reproduction; classification and the number of projects provide no evidential credit. `analogy` is explicit and cannot establish support by itself.

An aggregate failing result that also tested cases outside the target envelope is insufficient to refute that envelope. Isolate the failure inside the claimed cases. Reproduction must agree on the tested case set and normalized result/expected digests. These checks enforce the existing in-scope falsification requirement; all original records and historical assessments remain retained.

## Migration and compatibility

Migration 1 remains byte-for-byte unchanged. `002_hierarchy.sql` adds metadata tables, separate versions, curation grants, receipt/replay guards and indexes. Existing claim/evidence bodies, IDs, assessments and cached response bytes are not rewritten. Embedded legacy concept strings remain annotations; curators can explicitly convert them into graph associations, without automatic same-name merging.

The new runtime requires schema version 2 and refuses to start against version 1. Migration checksums are verified, repeat migration is a no-op, and a newer unknown schema is rejected. This is data compatibility, not permission to mix old and new writers: stop the old process during upgrade.

Upgrade procedure:

1. Back up the database and retain the pre-upgrade binary/source revision. Protect backups like the original data.
2. Stop the application. Keep PostgreSQL available for migration, or let the local helper restart its private cluster.
3. Run `holocron migrate` with the administrative DSN; then start the new runtime with the non-owner application DSN.
4. Retrieve fresh context. Existing domains receive a metadata-version bump so pre-upgrade completion receipts require refresh.
5. Check the old claim histories and credentials, then provision a dedicated curator for the new metadata.

There is no destructive down-migration. Rollback means restoring the pre-upgrade database and binary in a controlled window; preserve any post-upgrade records for reconciliation. Restore tests include new metadata tables and deletion-ledger replay. Whole-domain purge covers the new associations and history.

The local upgrade exposed a Python credential-file write error after provisioning the new curator. It was corrected with atomic mode-0600 writes, a private pending-provision journal, and retry-safe identity creation. A provisioning retry cannot replace credentials or re-enable a revoked curation grant. Recovery of the unused local curator credential was audited; no knowledge records were changed by that recovery.

An unchanged authorized retry still returns its original bytes. After an authorization-version change, the server returns a conflict instead of old payload bytes; the original mutation remains committed once. Do not blindly submit that mutation with a new key. Reconcile its known ID with current read operations. This release conservatively fences all cached responses in that visibility domain after an authority change; selective grants and multi-domain replay are deferred.

Credential expiry now prevents new authentication without retracting historically valid observations/reviews. Administrative `revoke` remains an explicit principal quarantine affecting support. `set-curation` changes metadata authority only and bumps the authorization version. It does not invalidate facts or erase previously authorized classification work.

## Agent configuration and authority

Use the existing [Codex MCP example](../examples/codex.toml) and [Claude Code example](../examples/claude.mcp.json), then supply the [updated participation instructions](../examples/agent-instructions.md). Do not give normal workers a curator or reviewer credential to make tool errors disappear.

Provision curators with `Role: worker` and `Curator: true` through the administrative CLI. Existing principals gain or lose that capability through `set-curation` with tenant, domain, ID and `Enabled`. A curation grant is domain-wide in this release; per-node delegated stewardship and policy-authority management are not implemented. Curators therefore represent the authorized metadata authority for both ends of a placement transfer.

For Codex unattended use, [the worker profile](../examples/codex-worker.toml) limits the visible tools and supplies literal-name approval tables. It is an opt-in configuration for the selected local service, not a user-wide installation. The server still enforces roles/grants. Claude can use the same authenticated endpoint with the appropriate MCP tool permissions. Validate `session.describe` first and save working IDs from `context.resolve`; names and concept labels are not permissions.

## Validation and explicit limitations

Validation completed with 36 Go test functions and their adversarial subcases against PostgreSQL 17 under `-race`, plus two experiment-analysis tests and a private-credential persistence regression. Build and `go vet ./...` passed. Focused regressions were rerun after the final scope-isolation and provisioning-retry changes. Documentation links, agent TOML, Python syntax and source credential-pattern checks also passed.

The release acceptance suite covers:

| Area | Tests |
| --- | --- |
| Classification neutrality and deduplication | `TestHierarchyClassificationDoesNotChangeEvidence`, `TestHierarchyProposalsAndMultiplePathsDoNotAmplify` |
| Positive broader coverage and scoped falsification | `TestBroaderApplicabilityRequiresAllNamedContexts`, `TestMatrixCasePartitionAndOutOfScopeCounterexample` |
| Generalization dependencies and alternatives | `TestBoundedGeneralizationRevalidatesThroughPremises`, existing alternative-support tests |
| Orientation, lateral discovery and qualified budgets | `TestHierarchyOrientationLateralTransferAndBudget`, `TestUnknownApplicabilityCannotCompleteAsLocalSupport` |
| Cycles, moves and history | `TestConcurrentHierarchyCyclesAndMetadataMove`, `TestSuccessorRequiresExplicitCuratedPlacement` |
| Replay, authorization and receipts | `TestMetadataReceiptsAndRevokedReplay`, `TestHierarchyAuthorizationAndRevokedCuration`, `TestMetadataMutationFencesCompletionAndBoundsRollback` |
| Import provenance and hidden metadata | `TestImportedSourcesRemainUnassessedAndHiddenMetadataIsInvisible` |
| Expiry versus distrust | `TestCredentialExpiryDoesNotEraseHistoricalEvidence` |
| Migration and retention | `TestMigrationPreservesV1RecordsAndReplay`, extended `TestDomainPurgeAndRestoreLedger` |
| Actual transports and bootstrap | `TestHierarchyAndUnknownEnvelopeThroughMCP`, `TestWadjetBootstrapIsMetadataOnlyAndRetrySafe`, existing REST/MCP tests |

The broader acceptance specification's multi-domain release, hidden-path federation, organizational-policy conflicts/exceptions and autonomous work-generation cases remain deferred because those features are absent. Cross-domain references are denied rather than partially implemented. There is no policy-inheritance engine: an organization-level factual claim or node named “policy” is not an authorized instruction.

Other limitations to observe during Wadjet use:

* Lexical/concept routing is deterministic but not empirically tuned. Concept associations can be wrong, and changing a concept definition needs curator attention. A factual definition used in reasoning must be a claim premise; node descriptions do not become logical premises.
* Domain locking serializes operations. Current bounds are 1000 claims, 10000 core graph records and 5000 hierarchy records. Large histories or excessive proposals can require operational cleanup; this is not an unlimited ingestion service.
* Receipt invalidation is conservative, including unrelated metadata edits within the same compartment. Measure refresh/curation cost before relaxing it.
* Node/edge history is retained, but arbitrary historical topology queries, concept retirement/merging, per-node ACLs and multi-domain identities are not implemented.
* References and execution manifests are attributed declarations until inspected. The server never fetches or executes their artifacts. A source digest does not establish correctness.
* No guarantee is made about revoking bytes already delivered to an independently operated agent. Reads committed before revocation may finish delivery; subsequent operations and completion checks use the new authority state.
* No live Wadjet productivity result is claimed. The bootstrap test checks real protocol behavior and idempotency, not whether the seeded concepts improve engineering outcomes.
* The local development database uses a private Unix socket. It is not a security boundary against another process running as the same OS user; isolate untrusted worker environments and credentials separately.
