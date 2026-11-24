# Wadjet round one interface and retrieval findings

The first externally coordinated Wadjet round exposed real adoption failures. This patch fixes the contract and retrieval mechanisms without promoting the captured observations into established findings or rewriting the workers' records.

## Observed failures

Workers could not discover allowed arm/outcome values, finite-case limits, artifact URI requirements or scope/envelope exclusivity from the original generated MCP types. Error messages combined unrelated failures or described the wrong cause. These failed calls are platform cost, not evidence of incapable workers.

The exact reported 6,000-byte query was:

```text
forced-drain aggregate gate: what it proves, reference taken disarmed, duplicate output rows
```

It supplied the investigation's execution scope but omitted discovery/working/orientation settings. The saved response contained one matched claim, an investigation, and `truncated: true`. Its 5,659 bytes were the entire bundle: approximately 3,740 bytes of cards and 1,573 bytes of investigation context.

Read-only inspection established three distinct problems. Omitted discovery selected legacy scope-only filtering, excluding all unknown-applicability ADRs before packing. The all-terms lexical interpretation matched none of the relevant claims. Finally, the default 20% orientation cap could exclude complete qualified source cards even after fixing discovery. The saved receipt did not explain these decisions.

## Corrections

* Explicit MCP enums, bounds, URI/digest patterns and conditional applicability shapes; field-specific recoverable errors. Source-only imports can omit execution scope and retain their pinned references under an explicitly unknown envelope.
* Text-query defaults now discover relevant source knowledge. Partial lexical matches are marked, and document-length normalization prevents repetition alone from dominating lexical rank. Explicit `legacy` mode remains available.
* Adaptive packing retains a complete relevant orientation card before a matched summary consumes the bundle. Explicit orientation caps are honored. Deferred navigation handles expose omitted candidates without presenting stripped assertions as facts.
* Retrieval diagnostics distinguish filtering from packing and work-context omissions. The receipt retains effective diagnostics, budget settings and a query digest; raw query text is not newly logged by default.
* Artifact references may be local `file://` URIs. They must name the actual capture, not a different production source file. Known origin is separate from having an expected result or oracle. Missing comparison still blocks establishment.
* Evidence profiles distinguish referenced experimental reports from no attached evidence. Unknown provenance, missing comparison, withdrawn/quarantined records and incomplete ancestry have different explanations. No numeric confidence or automatic promotion is added.

## Reproduction and limits

A private snapshot of the 32 claim inputs and original work context was captured using read-only transactions and replayed in a disposable PostgreSQL database. The source data is not committed. The explicit legacy control returned the original matched summary alone. At the same 6,000-byte budget, the corrected query returned the disarmed-reference rule and the matched summary in 5,839 bytes, plus a deferred reference. Diagnostics reported 18 candidates, 14 relevance exclusions, 16 full-card budget omissions and two omitted work items. The source rule retained unknown local applicability, its unassessed posture, limitations and pinned source reference.

This demonstrates recovery of the reported case, not a general relevance improvement. Term-overlap recall can reduce precision, and adaptive allocation can displace work context. Evaluate both on held-out questions at fixed budgets. The round had no curated hierarchy nodes at the inspected snapshot, so this is a flat discovery/packing failure, not a hierarchy treatment comparison.

The first empty retrieval also reflects cold-start timing: claims not yet published cannot be reused. A sequential fresh-consumer task after capture is needed to assess transfer, alongside the parallel acquisition round. Do not count only `applied` entries as useful knowledge: qualified guidance may properly be `considered` without being an established local premise.

All five inspected new evidence records were authenticated as `consumer`, declared unknown provenance, and omitted expected digests. Different model names or observation roots do not supply distinct authenticated reproduction. A reviewer token authorizes review but cannot waive missing comparison or independence. An appropriate validator must independently reproduce a defined method under its own identity.

## Rollout and evidence repair

Reconnect Claude's Holocron MCP server or start a fresh session after deploying the new binary so the client loads the revised schemas. A reviewer credential is configured in the client header/environment, not selected by restarting Holocron. Use a fresh idempotency key for retrieval experiments; replaying the old key returns the historical snapshot by design.

Do not silently rewrite the round's evidence. If a GitHub source URL was used for unrelated scratch output, preserve the actual output in durable customer-controlled storage and submit corrected metadata pointing to those bytes. Keep the original observation root so the correction is not counted as a new experiment; explain which record it corrects. Retract the misleading record through an authorized operation if appropriate. Do not fabricate a golden result or relabel provenance automatically. A defined expected property/metamorphic relation may support a new validation protocol, but it must be explicit and independently checked.

Failures rejected by SDK schema validation may occur before the database audit path. Retain client traces when measuring interface friction; database rejection counts alone are not a complete metric.
