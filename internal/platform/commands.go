package platform

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"
)

var Operations = []string{"session.describe", "claim.publish", "evidence.submit", "justification.propose", "challenge.submit", "review.record", "record.retract", "assessment.explain", "assessment.history", "evidence.get", "claim.get", "context.get", "investigation.open", "attempt.start", "attempt.finish", "work.list", "experiment.export", "events.poll", "events.ack"}

func init() {
	Operations = append(Operations, "node.create", "node.revise", "edge.add", "edge.retract", "association.propose", "association.review", "placement.transfer", "hierarchy.get", "context.resolve")
}

func knownOperation(op string) bool { return slices.Contains(Operations, op) }
func (t *transaction) dispatch(ctx context.Context, op string, b json.RawMessage) (json.RawMessage, bool, error) {
	id := NewID()
	result := encode(map[string]string{"id": id})
	switch op {
	case "node.create", "node.revise", "edge.add", "edge.retract", "association.propose", "association.review", "placement.transfer", "hierarchy.get", "context.resolve":
		return t.hierarchyCommand(ctx, op, b, id)
	case "session.describe":
		if _, err := decode[struct{}](b); err != nil {
			return nil, false, err
		}
		var curate bool
		if err := t.QueryRow(ctx, `SELECT can_curate($1)`, t.p.ID).Scan(&curate); err != nil {
			return nil, false, err
		}
		return encode(map[string]any{"identity": map[string]string{"id": t.p.ID, "role": t.p.Role, "owner": t.p.Owner, "tenant": t.p.Tenant, "domain": t.p.Domain}, "capabilities": map[string]bool{"curate": curate}, "build": t.service.build, "policy": Policy, "verification_policies": []string{Policy, MatrixPolicy}, "schema_version": SchemaVersion}), false, nil
	case "claim.publish":
		v, err := decode[ClaimInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = v.validateApplicability(); err != nil {
			return nil, false, err
		}
		if err = validateSources(v.Sources); err != nil {
			return nil, false, err
		}
		if len(v.Assumptions) > 64 {
			return nil, false, problem("invalid", "too many assumptions")
		}
		if err = unique(v.Assumptions); err != nil {
			return nil, false, err
		}
		if err = required(v.Statement, v.Kind, v.Limitations); err != nil {
			return nil, false, err
		}
		if len(v.Statement) > 8192 || !slices.Contains([]string{"finite_cases", "hypothesis", "generalization"}, v.Kind) {
			return nil, false, problem("invalid", "unsupported claim kind or excessive statement")
		}
		if len(v.Concepts) > 20 {
			return nil, false, problem("invalid", "too many concept labels")
		}
		if v.Previous != "" {
			_, author, err := t.get(ctx, "claims", v.Previous)
			if err != nil {
				return nil, false, err
			}
			if author != t.p.ID && t.reviewer() != nil {
				return nil, false, problem("forbidden", "only author or reviewer may supersede a claim")
			}
			var active bool
			err = t.QueryRow(ctx, `SELECT NOT EXISTS(SELECT FROM lifecycle WHERE tenant=$1 AND domain=$2 AND claim=$3)`, t.p.Tenant, t.p.Domain, v.Previous).Scan(&active)
			if err != nil {
				return nil, false, err
			}
			if !active {
				return nil, false, problem("conflict", "prior revision is not active")
			}
			if _, err = t.Exec(ctx, `INSERT INTO lifecycle(tenant,domain,id,author,claim,state,reason) VALUES($1,$2,$3,$4,$5,'superseded','successor revision published')`, t.p.Tenant, t.p.Domain, NewID(), t.p.ID, v.Previous); err != nil {
				return nil, false, err
			}
		}
		var previous any
		if v.Previous != "" {
			previous = v.Previous
		}
		return result, true, t.record(ctx, "claims", id, v, ",statement,previous", v.Statement, previous)
	case "evidence.submit":
		v, err := decode[EvidenceInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = v.Scope.validate(); err != nil {
			return nil, false, err
		}
		if err = required(v.Root, v.Artifact, v.Method, v.Command, v.Outcome, v.Provenance, v.Limitations); err != nil {
			return nil, false, err
		}
		if !slices.Contains([]string{"pass", "fail", "inconclusive", "infrastructure_error", "exhausted"}, v.Outcome) || !slices.Contains([]string{"known", "unknown"}, v.Provenance) {
			return nil, false, problem("invalid", "invalid evidence classification")
		}
		if v.ObservedAt.IsZero() || v.ObservedAt.After(time.Now().Add(5*time.Minute)) || v.TriggerCount < 0 {
			return nil, false, problem("invalid", "invalid observation time or trigger count")
		}
		if err = unique(v.Cases); err != nil {
			return nil, false, err
		}
		for _, c := range v.Cases {
			if !slices.Contains(v.Scope.Cases, c) {
				return nil, false, problem("invalid", "observed cases must belong to evidence scope")
			}
		}
		parsed, err := url.Parse(v.Artifact)
		if err != nil || parsed.Scheme == "" {
			return nil, false, problem("invalid", "versioned artifact URI required")
		}
		if v.Provenance == "known" {
			for _, value := range []string{v.Digest, v.ResultDigest, v.ExpectedDigest} {
				digest := strings.TrimPrefix(value, "sha256:")
				_, e := hex.DecodeString(digest)
				if e != nil || len(digest) != 64 || !strings.HasPrefix(value, "sha256:") {
					return nil, false, problem("invalid", "known provenance requires manifest, result and expected sha256 digests")
				}
			}
			if v.OracleSource == "" {
				return nil, false, problem("invalid", "oracle lineage is required")
			}
			if (v.Outcome == "pass" && v.ResultDigest != v.ExpectedDigest) || (v.Outcome == "fail" && v.ResultDigest == v.ExpectedDigest) {
				return nil, false, problem("invalid", "outcome contradicts recorded result digests")
			}
		}
		if len(v.DerivedFrom) > 20 {
			return nil, false, problem("invalid", "too many derivation inputs")
		}
		if err = unique(v.DerivedFrom); err != nil {
			return nil, false, err
		}
		for _, parent := range v.DerivedFrom {
			if _, _, err = t.get(ctx, "evidence", parent); err != nil {
				return nil, false, err
			}
		}
		if err = t.record(ctx, "evidence", id, v, ",root", v.Root); err != nil {
			return nil, false, err
		}
		for _, parent := range v.DerivedFrom {
			if err = t.link(ctx, "evidence_parents", "evidence", "parent", id, parent); err != nil {
				return nil, false, err
			}
		}
		return result, true, nil
	case "justification.propose":
		v, err := decode[JustificationInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.Claim, v.Warrant, v.Limitations); err != nil {
			return nil, false, err
		}
		if v.InferenceKind != "" && !slices.Contains([]string{"reproduction", "deduction", "analogy", "bounded_generalization"}, v.InferenceKind) {
			return nil, false, problem("invalid", "unknown inference kind")
		}
		if len(v.Evidence) < 1 || len(v.Evidence) > 20 || len(v.Premises) > 20 {
			return nil, false, problem("invalid", "bounded evidence and premise sets required")
		}
		if err = unique(v.Evidence); err != nil {
			return nil, false, err
		}
		if err = unique(v.Premises); err != nil {
			return nil, false, err
		}
		if _, _, err = t.get(ctx, "claims", v.Claim); err != nil {
			return nil, false, err
		}
		for _, p := range v.Premises {
			if _, _, err = t.get(ctx, "claims", p); err != nil {
				return nil, false, err
			}
		}
		for _, e := range v.Evidence {
			if _, _, err = t.get(ctx, "evidence", e); err != nil {
				return nil, false, err
			}
		}
		if err = t.record(ctx, "justifications", id, v, ",claim", v.Claim); err != nil {
			return nil, false, err
		}
		for _, p := range v.Premises {
			if err = t.link(ctx, "premises", "justification", "claim", id, p); err != nil {
				return nil, false, err
			}
		}
		for _, e := range v.Evidence {
			if err = t.link(ctx, "justification_evidence", "justification", "evidence", id, e); err != nil {
				return nil, false, err
			}
		}
		return result, true, nil // recompute rejects cycles before transaction commit
	case "challenge.submit":
		v, err := decode[ChallengeInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.Claim, v.Reason); err != nil {
			return nil, false, err
		}
		if v.Kind != "methodology" && v.Kind != "counterexample" {
			return nil, false, problem("invalid", "challenge kind must be methodology or counterexample")
		}
		if len(v.Evidence) > 20 {
			return nil, false, problem("invalid", "too many evidence references")
		}
		if err = unique(v.Evidence); err != nil {
			return nil, false, err
		}
		if _, _, err = t.get(ctx, "claims", v.Claim); err != nil {
			return nil, false, err
		}
		var justification any
		if v.Kind == "methodology" {
			jb, _, e := t.get(ctx, "justifications", v.Justification)
			if e != nil {
				return nil, false, e
			}
			j, _ := decode[JustificationInput](jb)
			if j.Claim != v.Claim {
				return nil, false, problem("invalid", "justification belongs to a different claim")
			}
			justification = v.Justification
		} else if v.Justification != "" {
			return nil, false, problem("invalid", "counterexample targets a claim")
		}
		for _, e := range v.Evidence {
			if _, _, err = t.get(ctx, "evidence", e); err != nil {
				return nil, false, err
			}
		}
		if err = t.record(ctx, "challenges", id, v, ",claim,justification", v.Claim, justification); err != nil {
			return nil, false, err
		}
		for _, e := range v.Evidence {
			if err = t.link(ctx, "challenge_evidence", "challenge", "evidence", id, e); err != nil {
				return nil, false, err
			}
		}
		return result, true, nil
	case "review.record":
		if err := t.reviewer(); err != nil {
			return nil, false, err
		}
		v, err := decode[ReviewInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.Target, v.Reason); err != nil {
			return nil, false, err
		}
		if !slices.Contains([]string{"accepted", "rejected", "inconclusive"}, v.Verdict) {
			return nil, false, problem("invalid", "invalid review verdict")
		}
		if v.TargetKind != "justification" && v.TargetKind != "challenge" {
			return nil, false, problem("invalid", "invalid review target kind")
		}
		table := v.TargetKind + "s"
		_, author, err := t.get(ctx, table, v.Target)
		if err != nil {
			return nil, false, err
		}
		if author == t.p.ID {
			return nil, false, problem("forbidden", "reviewer cannot approve own submission")
		}
		if v.Verdict == "accepted" && (!v.CoverageChecked || !v.ProvenanceChecked) {
			return nil, false, problem("invalid", "acceptance requires coverage and provenance review")
		}
		if v.Verdict == "accepted" {
			g, err := t.graph(ctx)
			if err != nil {
				return nil, false, err
			}
			if v.TargetKind == "justification" {
				j := g.justifications[v.Target]
				for _, eid := range j.Evidence {
					if g.evidence[eid].Author == t.p.ID {
						return nil, false, problem("forbidden", "reviewer cannot approve their own evidence")
					}
				}
				c := g.claims[j.Claim]
				if reasons := g.supportReasons(j, c); len(reasons) > 0 {
					return nil, false, problem("invalid", strings.Join(reasons, "; "))
				}
			}
			if v.TargetKind == "challenge" {
				ch := g.challenges[v.Target]
				for _, eid := range ch.Evidence {
					if g.evidence[eid].Author == t.p.ID {
						return nil, false, problem("forbidden", "reviewer cannot approve their own evidence")
					}
				}
				if ch.Kind == "counterexample" {
					if reasons := g.counterReasons(ch.Evidence, g.claims[ch.Claim]); len(reasons) > 0 {
						return nil, false, problem("invalid", strings.Join(reasons, "; "))
					}
				}
			}
		}
		return result, true, t.record(ctx, "reviews", id, v, ","+v.TargetKind, v.Target)
	case "record.retract":
		v, err := decode[LifecycleInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.Target, v.Reason); err != nil {
			return nil, false, err
		}
		if v.TargetKind != "claim" && v.TargetKind != "evidence" {
			return nil, false, problem("invalid", "invalid lifecycle target")
		}
		table := "claims"
		if v.TargetKind == "evidence" {
			table = "evidence"
		}
		_, author, err := t.get(ctx, table, v.Target)
		if err != nil {
			return nil, false, err
		}
		if author != t.p.ID && t.reviewer() != nil {
			return nil, false, problem("forbidden", "only author or reviewer can retract")
		}
		_, err = t.Exec(ctx, "INSERT INTO lifecycle(tenant,domain,id,author,"+v.TargetKind+",state,reason) VALUES($1,$2,$3,$4,$5,'retracted',$6)", t.p.Tenant, t.p.Domain, id, t.p.ID, v.Target, v.Reason)
		return result, true, err
	case "assessment.history":
		v, err := decode[IDInput](b)
		if err != nil {
			return nil, false, err
		}
		if _, _, err = t.get(ctx, "claims", v.ID); err != nil {
			return nil, false, err
		}
		rows, err := t.Query(ctx, `SELECT body FROM assessments WHERE tenant=$1 AND domain=$2 AND claim=$3 ORDER BY epoch DESC LIMIT 100`, t.p.Tenant, t.p.Domain, v.ID)
		if err != nil {
			return nil, false, err
		}
		defer rows.Close()
		items := []json.RawMessage{}
		for rows.Next() {
			var raw json.RawMessage
			if err = rows.Scan(&raw); err != nil {
				return nil, false, err
			}
			items = append(items, raw)
		}
		return encode(map[string]any{"assessments": items, "limit": 100}), false, rows.Err()
	case "evidence.get":
		v, err := decode[IDInput](b)
		if err != nil {
			return nil, false, err
		}
		raw, author, err := t.get(ctx, "evidence", v.ID)
		if err != nil {
			return nil, false, err
		}
		g, err := t.graph(ctx)
		if err != nil {
			return nil, false, err
		}
		return encode(map[string]any{"id": v.ID, "author": author, "manifest": raw, "lifecycle": g.lifecycle[v.ID], "trust": "untrusted evidence"}), false, nil
	case "assessment.explain", "claim.get":
		v, err := decode[IDInput](b)
		if err != nil {
			return nil, false, err
		}
		g, err := t.graph(ctx)
		if err != nil {
			return nil, false, err
		}
		a, err := g.assess(v.ID)
		if err != nil {
			return nil, false, err
		}
		if op == "assessment.explain" {
			return encode(a), false, nil
		}
		h, err := t.hierarchy(ctx)
		if err != nil {
			return nil, false, err
		}
		card := g.card(v.ID, a)
		card.MetadataEpoch = t.metadataEpoch
		h.decorate(&card, g)
		return encode(card), false, nil
	case "context.get":
		v, err := decode[ContextInput](b)
		if err != nil {
			return nil, false, err
		}
		out, err := t.context(ctx, v)
		return out, false, err
	case "investigation.open", "attempt.start", "attempt.finish", "work.list", "experiment.export":
		return t.work(ctx, op, b, id)
	case "events.poll", "events.ack":
		return t.events(ctx, op, b)
	}
	return nil, false, problem("invalid", "unknown operation")
}
