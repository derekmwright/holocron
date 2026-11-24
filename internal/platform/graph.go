package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"time"
)

type graph struct {
	claims         map[string]claimRecord
	evidence       map[string]evidenceRecord
	justifications map[string]justificationRecord
	challenges     map[string]challengeRecord
	reviews        map[string]reviewRecord
	lifecycle      map[string]string
	participants   map[string]Participant
	epoch          int64
	now            time.Time
	assessments    map[string]Assessment
	visiting       map[string]bool
}

func (t *transaction) graph(ctx context.Context) (*graph, error) {
	g := &graph{claims: map[string]claimRecord{}, evidence: map[string]evidenceRecord{}, justifications: map[string]justificationRecord{}, challenges: map[string]challengeRecord{}, reviews: map[string]reviewRecord{}, lifecycle: map[string]string{}, participants: map[string]Participant{}, epoch: t.epoch, now: time.Now(), assessments: map[string]Assessment{}, visiting: map[string]bool{}}
	count := 0
	for _, table := range []string{"claims", "evidence", "justifications", "challenges"} {
		rows, err := t.Query(ctx, "SELECT id,author,body FROM "+table+" WHERE tenant=$1 AND domain=$2 LIMIT $3", t.p.Tenant, t.p.Domain, t.service.MaxRecords+1)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			count++
			if count > t.service.MaxRecords {
				rows.Close()
				return nil, problem("limit", "graph record bound exceeded")
			}
			var id, author string
			var b []byte
			if err = rows.Scan(&id, &author, &b); err != nil {
				rows.Close()
				return nil, err
			}
			switch table {
			case "claims":
				var v ClaimInput
				err = json.Unmarshal(b, &v)
				g.claims[id] = claimRecord{v, id, author}
			case "evidence":
				var v EvidenceInput
				err = json.Unmarshal(b, &v)
				g.evidence[id] = evidenceRecord{v, id, author}
			case "justifications":
				var v JustificationInput
				err = json.Unmarshal(b, &v)
				g.justifications[id] = justificationRecord{v, id, author}
			case "challenges":
				var v ChallengeInput
				err = json.Unmarshal(b, &v)
				g.challenges[id] = challengeRecord{v, id, author}
			}
			if err != nil {
				rows.Close()
				return nil, err
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	if len(g.claims) > t.service.MaxClaims {
		return nil, problem("limit", "claim bound exceeded")
	}
	rows, err := t.Query(ctx, `SELECT author,body FROM reviews WHERE tenant=$1 AND domain=$2 ORDER BY seq`, t.p.Tenant, t.p.Domain)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var author string
		var b []byte
		if err = rows.Scan(&author, &b); err != nil {
			rows.Close()
			return nil, err
		}
		var v ReviewInput
		if err = json.Unmarshal(b, &v); err != nil {
			rows.Close()
			return nil, err
		}
		g.reviews[v.TargetKind+":"+v.Target] = reviewRecord{v, author}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = t.Query(ctx, `SELECT coalesce(claim,evidence),state FROM lifecycle WHERE tenant=$1 AND domain=$2 ORDER BY seq`, t.p.Tenant, t.p.Domain)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, state string
		if err = rows.Scan(&id, &state); err != nil {
			rows.Close()
			return nil, err
		}
		g.lifecycle[id] = state
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = t.Query(ctx, `SELECT id,owner,role,revoked FROM participants()`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		var p Participant
		if err = rows.Scan(&id, &p.Owner, &p.Role, &p.Revoked); err != nil {
			rows.Close()
			return nil, err
		}
		g.participants[id] = p
	}
	err = rows.Err()
	rows.Close()
	return g, err
}
func (g *graph) review(kind, id string) (reviewRecord, bool) {
	r, ok := g.reviews[kind+":"+id]
	p := g.participants[r.Author]
	return r, ok && !p.Revoked && (p.Role == "reviewer" || p.Role == "admin")
}
func (g *graph) evidencePair(ids []string, s Scope, outcome string) []string {
	reasons := []string{}
	valid := []evidenceRecord{}
	for _, id := range ids {
		e, ok := g.evidence[id]
		if !ok {
			reasons = append(reasons, "missing evidence "+id)
			continue
		}
		p := g.participants[e.Author]
		if g.lifecycle[id] != "" || p.Revoked || !g.originAvailable(id, map[string]bool{}) {
			reasons = append(reasons, "unavailable evidence "+id)
			continue
		}
		if e.Provenance != "known" || e.Root == "" || e.Digest == "" || e.OracleSource == "" {
			reasons = append(reasons, "unknown provenance "+id)
			continue
		}
		if e.Outcome != outcome {
			reasons = append(reasons, "outcome does not qualify "+id)
			continue
		}
		if e.ResultDigest == "" || e.ExpectedDigest == "" || (outcome == "pass" && e.ResultDigest != e.ExpectedDigest) || (outcome == "fail" && e.ResultDigest == e.ExpectedDigest) {
			reasons = append(reasons, "result digest does not substantiate outcome "+id)
			continue
		}
		if e.TriggerCount <= 0 {
			reasons = append(reasons, "required trigger not engaged "+id)
			continue
		}
		if !e.Scope.sameEnvironment(s) {
			reasons = append(reasons, "scope mismatch "+id)
			continue
		}
		covered := Scope{Repository: s.Repository, Commit: s.Commit, Environment: s.Environment, Configuration: s.Configuration, Dataset: s.Dataset, Oracle: s.Oracle, Cases: e.Cases}
		if outcome == "pass" && !covered.covers(s) {
			reasons = append(reasons, "incomplete case coverage "+id)
			continue
		}
		if outcome == "fail" {
			overlap := false
			for _, c := range e.Cases {
				if slices.Contains(s.Cases, c) {
					overlap = true
				}
			}
			if !overlap {
				reasons = append(reasons, "counterexample outside cases "+id)
				continue
			}
		}
		valid = append(valid, e)
	}
	paired := false
	for i, a := range valid {
		for j, b := range valid {
			if i == j {
				continue
			}
			pb := g.participants[b.Author]
			// Distinct executions and artifacts are necessary, not proof of statistical independence.
			if a.Author != b.Author && !g.sharedOrigin(a.ID, b.ID) && a.Artifact != b.Artifact && a.Digest != b.Digest && (pb.Role == "validator" || pb.Role == "reviewer" || pb.Role == "admin") && a.Method == b.Method && a.OracleSource == b.OracleSource {
				if outcome == "fail" {
					common := false
					for _, c := range a.Cases {
						if slices.Contains(b.Cases, c) && slices.Contains(s.Cases, c) {
							common = true
						}
					}
					if !common {
						continue
					}
				}
				paired = true
			}
		}
	}
	if !paired {
		reasons = append(reasons, "missing distinct authorized reproduction with matching method and oracle")
	}
	return reasons
}
func (g *graph) originAvailable(id string, seen map[string]bool) bool {
	if seen[id] {
		return false
	}
	seen[id] = true
	defer delete(seen, id)
	e, ok := g.evidence[id]
	if !ok || g.lifecycle[id] != "" || g.participants[e.Author].Revoked || e.Provenance != "known" {
		return false
	}
	for _, parent := range e.DerivedFrom {
		if !g.originAvailable(parent, seen) {
			return false
		}
	}
	return true
}
func (g *graph) sharedOrigin(a, b string) bool {
	origins := func(id string) map[string]bool {
		seen := map[string]bool{}
		roots := map[string]bool{}
		var visit func(string)
		visit = func(id string) {
			if seen[id] {
				return
			}
			seen[id] = true
			e := g.evidence[id]
			roots[e.Root] = true
			roots["digest:"+e.Digest] = true
			for _, p := range e.DerivedFrom {
				visit(p)
			}
		}
		visit(id)
		return roots
	}
	left, right := origins(a), origins(b)
	for root := range left {
		if right[root] {
			return true
		}
	}
	return false
}
func (g *graph) assess(id string) (Assessment, error) {
	if a, ok := g.assessments[id]; ok {
		return a, nil
	}
	c, ok := g.claims[id]
	if !ok {
		return Assessment{}, problem("not_found", "claim not found")
	}
	if g.visiting[id] {
		return Assessment{}, problem("invalid", "cyclic justification derivation")
	}
	g.visiting[id] = true
	defer delete(g.visiting, id)
	a := Assessment{Claim: id, Policy: Policy, Epoch: g.epoch, Posture: "unassessed", Health: "current", Lifecycle: "active", Justifications: []Explanation{}, Challenges: []ChallengeExplanation{}, Reasons: []string{}}
	if state := g.lifecycle[id]; state != "" {
		a.Lifecycle = state
	}
	disabled := map[string]bool{}
	counter := false
	anyAssessment := false
	for _, chid := range sortedKeys(g.challenges) {
		ch := g.challenges[chid]
		if ch.Claim != id {
			continue
		}
		r, reviewed := g.review("challenge", chid)
		verdict := "pending"
		if reviewed {
			verdict = r.Verdict
			anyAssessment = true
		} else {
			a.Pending = true
		}
		if verdict == "inconclusive" {
			a.Pending = true
		}
		if verdict == "accepted" {
			if ch.Kind == "methodology" {
				disabled[ch.Justification] = true
			} else if r.CoverageChecked && r.ProvenanceChecked && len(g.evidencePair(ch.Evidence, c.Scope, "fail")) == 0 {
				counter = true
			} else {
				verdict = "inconclusive"
				a.Pending = true
			}
		}
		a.Challenges = append(a.Challenges, ChallengeExplanation{chid, ch.Kind, verdict, ch.Reason, ch.Evidence})
	}
	adequate := false
	affected := false
	for _, jid := range sortedKeys(g.justifications) {
		j := g.justifications[jid]
		if j.Claim != id {
			continue
		}
		e := Explanation{Justification: jid, Evidence: j.Evidence, Premises: j.Premises, Reasons: []string{}}
		r, reviewed := g.review("justification", jid)
		if !reviewed || r.Verdict != "accepted" {
			e.Reasons = append(e.Reasons, "scope warrant not accepted by active reviewer")
		} else {
			anyAssessment = true
			if !r.CoverageChecked || !r.ProvenanceChecked {
				e.Reasons = append(e.Reasons, "coverage or provenance review missing")
			}
		}
		if reviewed {
			anyAssessment = true
		}
		if disabled[jid] {
			e.Reasons = append(e.Reasons, "accepted methodological challenge")
		}
		if c.Kind != "finite_cases" {
			e.Reasons = append(e.Reasons, "policy covers finite cases only")
		}
		e.Reasons = append(e.Reasons, g.evidencePair(j.Evidence, c.Scope, "pass")...)
		for _, premise := range j.Premises {
			p, err := g.assess(premise)
			if err != nil {
				return Assessment{}, err
			}
			if p.Posture != "supported" || p.Health != "current" || p.Lifecycle != "active" {
				e.Reasons = append(e.Reasons, "premise requires revalidation: "+premise)
				affected = true
			}
			if p.Pending {
				a.Pending = true
				a.Reasons = append(a.Reasons, "mandatory premise has unresolved challenge: "+premise)
			}
		}
		e.Eligible = len(e.Reasons) == 0
		if reviewed && !e.Eligible {
			affected = true
		}
		if e.Eligible {
			adequate = true
		}
		a.Justifications = append(a.Justifications, e)
	}
	if len(a.Justifications) > 0 || anyAssessment {
		a.Posture = "insufficient"
	}
	if adequate {
		a.Posture = "supported"
	}
	if counter {
		if adequate {
			a.Posture = "contested"
		} else {
			a.Posture = "refuted"
		}
	}
	if affected && !adequate {
		a.Health = "revalidation_required"
	}
	if a.Lifecycle != "active" {
		a.Health = "revalidation_required"
		a.Reasons = append(a.Reasons, "claim is "+a.Lifecycle)
	}
	if c.ReviewAfter != nil && !g.now.Before(*c.ReviewAfter) {
		a.Health = "revalidation_required"
		a.Reasons = append(a.Reasons, "review deadline passed")
	}
	if a.Pending {
		a.Reasons = append(a.Reasons, "unresolved challenge; inspect before application")
	}
	g.assessments[id] = a
	return a, nil
}
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (t *transaction) recompute(ctx context.Context) error {
	g, err := t.graph(ctx)
	if err != nil {
		return err
	}
	t.service.propagation.Record(ctx, int64(len(g.claims)))
	for _, id := range sortedKeys(g.claims) {
		a, err := g.assess(id)
		if err != nil {
			return err
		}
		_, err = t.Exec(ctx, `INSERT INTO assessments(tenant,domain,claim,epoch,body) VALUES($1,$2,$3,$4,$5)`, t.p.Tenant, t.p.Domain, id, t.epoch, encode(a))
		if err != nil {
			return err
		}
		if a.Health != "current" || a.Posture == "contested" {
			_, err = t.Exec(ctx, `INSERT INTO opportunities(tenant,domain,claim,epoch,reason) VALUES($1,$2,$3,$4,$5)`, t.p.Tenant, t.p.Domain, id, t.epoch, fmt.Sprintf("%s / %s", a.Posture, a.Health))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
