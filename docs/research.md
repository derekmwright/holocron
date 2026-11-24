# Research and architectural challenges

Research date: 2026-11-24. This is a bounded architectural investigation, not a production security audit or a systematic literature review. Repository observations below come from source inspection, not execution. Proposed adaptations are distinguished from demonstrated results.

## Recommendation

Build a small system for maintaining the reasons behind scoped conclusions. Make a finding an explainable assessment under a named verification policy, not a permanent truth label. Test it against ordinary artifact retrieval before building autonomous coordination or broad historical ingestion.

The largest risk is semantic: agents can write plausible claims, provenance, and validation reports that are wrong. Storage integrity cannot establish their truth. The system can enforce attribution, scope boundaries, separation of authority, and review obligations; it cannot eliminate the need for trustworthy observations and domain judgment.

## Lessons from the previous Holocron

Inspected repository tree at commit `d62b74bd00e0ad711eb05143ac2db4c0ec382fc1`, selected migrations, observation service, context assembler, pattern models, ADRs, and stop hook. This is a source sample, not a claim about every current code path.

| Source observation | Lesson for the new system |
| --- | --- |
| The [initial schema](https://github.com/derekmwright/holocron-cloud/blob/d62b74bd00e0ad711eb05143ac2db4c0ec382fc1/migrations/001_initial.sql) separates personal memories and team patterns; patterns have a canonical flag and Git coordinates. | Keep artifact references. Do not inherit a personal/team split as the knowledge ontology or equate canonicalization with evidence. |
| The [observation migration](https://github.com/derekmwright/holocron-cloud/blob/d62b74bd00e0ad711eb05143ac2db4c0ec382fc1/migrations/007_observations_lifecycle.sql) adds processed state and memory observation-ID arrays. | Processing is workflow state, not validation. Use enforced, revision-specific provenance relationships rather than an untyped array as the primary lineage model. |
| [Save and BatchSave](https://github.com/derekmwright/holocron-cloud/blob/d62b74bd00e0ad711eb05143ac2db4c0ec382fc1/internal/observations/service.go) suppress similar observations using a 0.8 similarity threshold; cleanup purges old or repeatedly skipped records. | Similar wording may be an independent replication or a subtly different counterexample. Deduplicate request identity, not observations by prose similarity. Retention must account for downstream use. |
| The [context assembler](https://github.com/derekmwright/holocron-cloud/blob/d62b74bd00e0ad711eb05143ac2db4c0ec382fc1/internal/memories/context.go) fills by domain and importance before query recall; recall allows a 20% overflow and token counts use text length divided by four. | Query applicability should precede diversity. Make budget behavior explicit and retain qualification text with each claim. Importance is not relevance. |
| The [stop hook](https://github.com/derekmwright/holocron-cloud/blob/d62b74bd00e0ad711eb05143ac2db4c0ec382fc1/hooks/holocron/scripts/stop.sh) sends truncated stdin as an observation and suppresses curl failure. | Automatic capture can lose structure, leak session material, and appear successful without persistence. Prefer intentional evidence manifests and durable acknowledgments. |
| The [ADRs](https://github.com/derekmwright/holocron-cloud/blob/d62b74bd00e0ad711eb05143ac2db4c0ec382fc1/_context/DECISIONS.md) favor pgx, agent-side synthesis, references, and embedded NATS. | Reuse the explicit SQL and inference-independent direction. References reduce duplication but do not eliminate confidentiality risks. Add messaging only when its delivery responsibility is necessary. |

No schema or packages should be copied wholesale. Source inspection provides reusable lessons, not evidence of operational reliability or product value.

## P vs NP protocol

The public [join protocol](https://pequalnp.com/join.txt) describes desk-assigned work, contextual failed approaches, renewable leases, staged duplicate assignment for hard steps, structured release reasons, exact-statement proof checking, and checked reductions into smaller obligations. It distinguishes checker infrastructure failure from rejected proof. Its late-submission guidance is inconsistent: one passage allows completed proofs after lease loss; another forbids submission after renewal returns 410. Server behavior was not tested.

Adopt bounded pull work, explicit failure outcomes, immutable acceptance targets, and separate execution from validation. Define late evidence intake independently from permission to complete a task. Defer speculative execution until its cost is measured. A checked proof is only as useful as its statement and assumptions; engineering generally has neither a universal checker nor complete formal specifications. The [project homepage](https://pequalnp.com/) describes checked mathematical steps, not evidence that the coordination strategy improves general engineering productivity. No agents were registered and no work was acquired during this investigation.

## Established techniques and their limits

| Approach and primary source | Established contribution | Proposed Holocron use and limit |
| --- | --- | --- |
| [Doyle, A Truth Maintenance System, 1979](https://www.sciencedirect.com/science/article/abs/pii/0004370279900080) | Records reasons for beliefs and revises assumptions when discoveries conflict. | Store justifications and recompute their usability after change. This maintains recorded reasons, not objective truth. |
| [de Kleer, An Assumption Based TMS, 1986](https://www.dekleer.org/Publications/An%20Assumption-Based%20TMS.pdf) | Tracks conclusions under sets of assumptions. | Preserve conditional support and alternative justifications. Defer full environment enumeration because combinations can grow sharply. |
| [AGM, On the Logic of Theory Change, 1985](https://www.cambridge.org/core/journals/journal-of-symbolic-logic/article/abs/on-the-logic-of-theory-change-partial-meet-contraction-and-revision-functions/7ED837BAD5FB6D9A7C77906D73527F9C) | Formalizes contraction and revision of theories. | Distinguish withdrawing support from asserting negation. Do not force all organizational disagreement into one consistent, deductively closed theory. |
| [Dung, On the Acceptability of Arguments, 1995](https://www.academia.edu/144184758/On_the_acceptability_of_arguments_and_its_fundamental_role_in_nonmonotonic_reasoning_logic_programming_and_n_person_games) | Defines argument acceptability using attack relations. | Represent rebuttals and attacks on methodology separately. Graph acceptability alone does not establish empirical correctness; no full argument solver in MVP. |
| [W3C PROV DM](https://www.w3.org/TR/prov-dm/) | Distinguishes entities, activities, agents, attribution, usage, and derivation. | Use compatible conceptual provenance. A lineage declaration remains attributable and challengeable; it is not proof of independence. |
| [Green, Karvounarakis, Tannen, provenance semirings, 2007 presentation](https://www.cis.upenn.edu/~plclub/propr/greg-slides.pdf) | Represents alternative and joint derivations of query results. | Store alternatives as separate justification sets and joint premises as members of one set. Avoid treating repeated derivation paths as independent observations. Full polynomial machinery is unnecessary initially. |
| [Evans, conditional independence](https://www.stats.ox.ac.uk/~evans/gms/_book/ci.html) | Specifies when joint distributions factorize conditionally. | Bayesian updates are defensible only with an explicit likelihood/dependence model. Different agent IDs supply no such model. |
| [Cochrane, studies versus reports](https://www.cochrane.org/authors/handbooks-and-manuals/handbook/current/chapter-04) | Distinguishes underlying studies from multiple reports of them. | Treat an experiment and its issue, summary, and ADR as one observation lineage, not four replications. This is an analogy, not adoption of a medical evidence hierarchy. |
| [Shannon, A Mathematical Theory of Communication, 1948](https://people.math.harvard.edu/~ctm/home/text/others/shannon/entropy/entropy.pdf) | Quantifies communication under probabilistic message models. | Token reduction is measurable; semantic usefulness per token is a workload outcome, not Shannon entropy without a defensible model. |

Scientific practice suggests explicit falsifiers, preserved negative results, prospective predictions, and independent reproduction. Holocron should encode experimental conditions and measurement limitations rather than implement a universal philosophy of science. A passing test can support a finite case; a universal claim may be defeated by one valid in-scope counterexample. Stochastic claims instead require comparison against their specified distribution or tolerance, not treating every surprising sample as refutation.

## Relevant existing systems

[Wikibase](https://www.mediawiki.org/wiki/Wikibase/DataModel) separates statements, qualifiers, references, and ranks. Its statement-oriented representation is useful prior art; rank should not become a probability or organizational authorization. [Zep and Graphiti](https://arxiv.org/abs/2501.13956) investigate temporal graph memory and report retrieval benchmark results. Temporal history is relevant, but conversational memory performance does not establish engineering correctness, provenance independence, or safe invalidation. Compare Holocron against both straightforward document retrieval and temporal memory concepts before claiming a novel benefit.

Git-backed ADRs, regression suites, issue history, and CI already form a partial organizational knowledge system. Holocron must earn its maintenance cost by linking and updating that evidence better than agents using those artifacts directly. A new graph is not itself an improvement.

## Wadjet evidence sample

Inspected tree at `f9a61994904c2d06dc0b112db048e8da762838ef` and selected ADR content. No Wadjet tests or benchmarks were run.

* [ADR 0013](https://github.com/derekmwright/wadjet/blob/f9a61994904c2d06dc0b112db048e8da762838ef/docs/adr/0013-correctness-gates-and-their-boundaries.md) distinguishes reference comparison, path agreement, and fuzzing, with different blind spots. It describes both-arms-wrong risk and legal nondeterminism. Import these as policy and reported lessons, not freshly reproduced findings.
* [ADR 0027](https://github.com/derekmwright/wadjet/blob/f9a61994904c2d06dc0b112db048e8da762838ef/docs/adr/0027-a-spill-gate-proves-it-spilled.md) describes spill suites that never exercised spilling and requires engagement measurements. This is a concrete reason to include coverage and trigger evidence in validation.
* [ADR 0011](https://github.com/derekmwright/wadjet/blob/f9a61994904c2d06dc0b112db048e8da762838ef/docs/adr/0011-performance-measurement-methodology.md) requires controlled benchmark comparisons and mechanism metrics. Its historical results need their run artifacts and environments before they can support new performance conclusions.

The available tree includes correctness fixtures, benchmark reports, and many ADRs. The issue archive, commit bodies, and coordinator/worker session corpus have not been comprehensively audited. Historical bootstrap is therefore a proposed bounded process, not a completed ingestion or exhaustive knowledge inventory.

## Assumptions to challenge

| Assumption | Failure mode | Design response |
| --- | --- | --- |
| Everything important can be an atomic claim. | Fragmentation loses causal narratives and makes authoring expensive. | Keep concise scoped claims plus linked explanations; measure authoring burden. |
| Independence is an agent property. | Shared tests, prompts, sources, models, or owners create correlated errors. | Record independence dimensions and unknowns at the activity and evidence level. |
| Contradiction can be detected mechanically from prose. | Different versions or quantifiers look contradictory. | Explicit proposals and review; automated candidates never resolve themselves. |
| Invalidating a premise makes every descendant false. | Alternative justification remains, or dependency was only contextual. | Recompute justification eligibility; mark affected conclusions for review. |
| A more general claim is more useful. | Unsupported extrapolation becomes authoritative context. | Require a separate revision or claim and a scope warrant with exceptions. |
| Immutable records make history trustworthy. | Fabrications become permanently recorded; sensitive data remains exposed. | Separate integrity from validity; preserve attribution with authorized purge and redaction. |
| A coordinator should always generate more work. | Endless research displaces delivery. | Tie opportunities to objectives, budgets, owners, and stopping conditions. |
| Enforced contribution improves coverage. | Agents manufacture low-value observations. | Gate a disposition and evidence manifest, allowing an empty contribution. |
| A summary of restricted evidence is safe to share. | The summary reveals the secret. | Derived visibility cannot automatically exceed source visibility. |
| Three agents demonstrate compounding intelligence. | A scripted example demonstrates mechanics only. | Run independent, randomized engineering workloads and include maintenance cost. |

## Unresolved research questions

1. How reliably can agents author useful scope without expensive review? Measure scope corrections and authoring time on the pilot corpus.
2. Which independence dimensions predict new information? Compare reruns, different oracles, independently written tests, and blinded investigations; do not presume model diversity suffices.
3. Can qualification-preserving context outperform ordinary retrieval at equal budgets? Test as a separate treatment.
4. How often do dependency links miss real impact or cause excessive revalidation? Seed code changes with known effects and measure both errors.
5. What policy is appropriately conservative without letting cheap challenges disable useful knowledge? Track challenge disposition and review latency; no reputation shortcut initially.
6. Which historical lessons transfer across versions? Benchmark held-out tasks and temporal cutoffs.
7. Can useful investigation opportunities be selected without server inference? Begin with rules and human prioritization; measure completed value, not queue size.
8. What is the break-even reuse horizon after curation, verification, and deletion costs? Report the full cost curve, including a negative result.

Established mechanisms are relational integrity, versioned records, explicit lineage, policy evaluation, leases, and dependency traversal. Automatic semantic contradiction resolution, calibrated cross-domain belief scores, reliable provenance independence detection, and autonomous investigation selection remain unvalidated research for this product.
