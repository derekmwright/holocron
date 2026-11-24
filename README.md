# Holocron

Holocron is an experimental platform for durable organizational knowledge supported by evidence and open to correction.

The foundational design was approved on 2026-11-24. The repository now contains a Go/PostgreSQL prototype, REST and MCP interfaces, and real-database integration/adversarial tests. Its finite-case verification policy keeps provenance, contradictions, revisions, and dependency revalidation explicit.

```sh
go build -buildvcs=false ./cmd/holocron
bash scripts/test-integration.sh
bash scripts/demo.sh
```

The integration script creates and removes a disposable PostgreSQL cluster and runs tests with the race detector. PostgreSQL 17 binaries and local socket access are required. The demo uses scripted participant identities; it does not demonstrate improved LLM performance.

Start with [running the prototype](docs/operations.md), [the API](docs/api.md), and [implementation status and limits](docs/implementation.md).

Read in this order:

1. [Research and architectural challenges](docs/research.md)
2. [Foundational design](docs/design.md)
3. [Architectural decision records](docs/decisions.md)
4. [MVP and experimental validation](docs/validation.md)

The system is a Go modular monolith with PostgreSQL, immutable claim revisions and evidence records, explicit justifications, policy-specific assessments, and conservative dependency revalidation. It requires no server-side inference, vector database, or message broker.

The central experiment is whether curated, revisable organizational knowledge improves subsequent independent engineering work enough to pay for its acquisition and maintenance. The design does not assume that this has already been demonstrated.
