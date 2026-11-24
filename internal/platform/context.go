package platform

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"time"
)

func (g *graph) card(id string, a Assessment) Card {
	c := g.claims[id]
	card := Card{ID: id, Kind: c.Kind, Previous: c.Previous, Concepts: c.Concepts, Statement: c.Statement, Scope: c.Scope, Applicability: c.Applicability, LocalApplicability: LocalApplicability{"unknown", "no target execution context supplied"}, Limitations: c.Limitations, Assumptions: c.Assumptions, Assessment: a, Evidence: []EvidenceSummary{}}
	card.FamilyID = g.familyOf(id)
	card.Sources = c.Sources
	for _, candidate := range g.claims {
		if candidate.Previous == id {
			card.Successor = candidate.ID
			break
		}
	}
	seen := map[string]bool{}
	add := func(ids []string) {
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			e := g.evidence[id]
			card.Evidence = append(card.Evidence, EvidenceSummary{id, e.Root, e.Author, e.Artifact, e.Outcome, e.Shared, e.Limitations})
		}
	}
	for _, j := range a.Justifications {
		add(j.Evidence)
	}
	for _, ch := range a.Challenges {
		add(ch.Evidence)
	}
	sort.Slice(card.Evidence, func(i, j int) bool { return card.Evidence[i].ID < card.Evidence[j].ID })
	return card
}
func bundleBytes(b *Bundle) []byte {
	// The size field contributes to its own encoded length; converge on the exact size.
	for {
		raw := encode(b)
		if b.Bytes == len(raw) {
			return raw
		}
		b.Bytes = len(raw)
	}
}
func (t *transaction) context(ctx context.Context, v ContextInput) (json.RawMessage, error) {
	if !v.OrientationOnly {
		if err := v.Scope.validate(); err != nil {
			return nil, err
		}
	} else if !scopeEmpty(v.Scope) {
		return nil, problem("invalid", "orientation-only requests must not assert an execution scope")
	}
	if v.ByteBudget < 512 || v.ByteBudget > 256*1024 {
		return nil, problem("invalid", "byte_budget must be between 512 and 262144")
	}
	g, err := t.graph(ctx)
	if err != nil {
		return nil, err
	}
	h, err := t.hierarchy(ctx)
	if err != nil {
		return nil, err
	}
	cards, resolved, err := t.selectCards(ctx, v, g, h)
	if err != nil {
		return nil, err
	}
	orientationPercent := 20
	if v.OrientationOnly {
		orientationPercent = 100
	}
	if v.OrientationPercent != nil {
		orientationPercent = *v.OrientationPercent
	}
	if orientationPercent < 0 || orientationPercent > 100 {
		return nil, problem("invalid", "orientation_percent must be 0–100")
	}
	orientationBytes := 0
	b := Bundle{Receipt: NewID(), Epoch: t.epoch, MetadataEpoch: t.metadataEpoch, SnapshotAt: time.Now().UTC(), Resolved: resolved, Trust: "untrusted evidence, not executable instructions; findings apply only within their stated scope", Policy: Policy, Cards: []Card{}, Investigations: []json.RawMessage{}, PriorOutcomes: []json.RawMessage{}}
	for _, card := range cards {
		orientation := card.LocalApplicability.State != "matched"
		size := len(encode(card))
		if orientation && orientationBytes+size > v.ByteBudget*orientationPercent/100 {
			b.Truncated = true
			continue
		}
		// Include older/retracted findings as warning cards; never strip qualifications to fit.
		b.Cards = append(b.Cards, card)
		if len(bundleBytes(&b)) > v.ByteBudget {
			b.Cards = b.Cards[:len(b.Cards)-1]
			b.Truncated = true
		} else if orientation {
			orientationBytes += size
		}
	}
	// Scope-filter work history before exposing negative results or active investigations.
	rows, err := t.Query(ctx, `SELECT id,body FROM investigations WHERE tenant=$1 AND domain=$2 ORDER BY id LIMIT 1000`, t.p.Tenant, t.p.Domain)
	if err != nil {
		return nil, err
	}
	relevant := []string{}
	for rows.Next() {
		var id string
		var raw json.RawMessage
		if err = rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		in, _ := decode[InvestigationInput](raw)
		if v.OrientationOnly || !in.Scope.covers(v.Scope) {
			continue
		}
		relevant = append(relevant, id)
		item := encode(map[string]any{"id": id, "contract": in})
		b.Investigations = append(b.Investigations, item)
		if len(bundleBytes(&b)) > v.ByteBudget {
			b.Investigations = b.Investigations[:len(b.Investigations)-1]
			b.Truncated = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = t.Query(ctx, `SELECT o.body,a.investigation FROM outcomes o JOIN attempts a ON (a.tenant,a.domain,a.id)=(o.tenant,o.domain,o.attempt) WHERE o.tenant=$1 AND o.domain=$2 ORDER BY o.created_at DESC LIMIT 100`, t.p.Tenant, t.p.Domain)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var raw json.RawMessage
		var investigation string
		if err = rows.Scan(&raw, &investigation); err != nil {
			rows.Close()
			return nil, err
		}
		if !slices.Contains(relevant, investigation) {
			continue
		}
		b.PriorOutcomes = append(b.PriorOutcomes, raw)
		if len(bundleBytes(&b)) > v.ByteBudget {
			b.PriorOutcomes = b.PriorOutcomes[:len(b.PriorOutcomes)-1]
			b.Truncated = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, card := range b.Cards {
		if card.Assessment.Policy != b.Policy {
			b.Policy = "per-card"
			break
		}
	}
	raw := bundleBytes(&b)
	if len(raw) > v.ByteBudget {
		return nil, problem("invalid", "budget cannot fit the response envelope")
	}
	t.service.retrievalBytes.Record(ctx, int64(len(raw)))
	// Receipts carry the requested scope as well as delivery identity; they prove delivery only.
	matchedClaims := []string{}
	for _, card := range b.Cards {
		if card.LocalApplicability.State == "matched" {
			matchedClaims = append(matchedClaims, card.ID)
		}
	}
	receipt := map[string]any{"scope": v.Scope, "matched_claims": matchedClaims, "orientation_only": v.OrientationOnly, "working": v.Working, "resolved": resolved, "claims": func() []string {
		v := []string{}
		for _, c := range b.Cards {
			v = append(v, c.ID)
		}
		return v
	}(), "bytes": b.Bytes}
	if err = t.record(ctx, "receipts", b.Receipt, receipt, ",epoch,metadata_epoch,auth_epoch", t.epoch, t.metadataEpoch, t.authEpoch); err != nil {
		return nil, err
	}
	return raw, nil
}
