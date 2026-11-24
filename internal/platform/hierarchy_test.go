package platform

import (
	"context"
	"encoding/json"
	"testing"
)

func node(t *testing.T, s *Service, category, kind, name string) string {
	return invoke(t, s, "curator", "node.create", NodeInput{Category: category, Kind: kind, Name: name})["id"].(string)
}
func edge(t *testing.T, s *Service, relation, from, to string) string {
	return invoke(t, s, "curator", "edge.add", EdgeInput{Relation: relation, From: from, To: to, Reason: "Test curated structure"})["id"].(string)
}
func associate(t *testing.T, s *Service, kind, claim, entity, target string) string {
	id := invoke(t, s, "worker-a", "association.propose", AssociationInput{Kind: kind, Claim: claim, Entity: entity, Target: target, Reason: "Test association"})["id"].(string)
	invoke(t, s, "curator", "association.review", AssociationReviewInput{ID: id, Verdict: "accepted", Reason: "Reviewed placement or subject"})
	return id
}
func TestHierarchyClassificationDoesNotChangeEvidence(t *testing.T) {
	s := fixture(t)
	c := claim(t, s, "Scoped memory observation "+NewID())
	support(t, s, c)
	before := assessment(t, s, c)
	org := node(t, s, "organization", "organization", "Engineering")
	project := node(t, s, "organization", "project", "Wadjet")
	repo := invoke(t, s, "curator", "node.create", NodeInput{Category: "organization", Kind: "repository", Name: "wadjet", Repository: scope().Repository})["id"].(string)
	edge(t, s, "contains", org, project)
	edge(t, s, "contains", project, repo)
	memory := node(t, s, "concept", "discipline", "Memory management")
	backend := node(t, s, "concept", "discipline", "Backend")
	associate(t, s, "entity_concept", "", repo, memory)
	associate(t, s, "claim_concept", c, "", memory)
	associate(t, s, "claim_concept", c, "", backend)
	after := assessment(t, s, c)
	if before["epoch"] != after["epoch"] || after["posture"] != "supported" {
		t.Fatal("classification changed assessment", before, after)
	}
	out := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), Working: &WorkingContext{Organization: org, Project: project, Repository: repo, Concepts: []string{memory, backend}}, ByteBudget: 64000})
	matches := 0
	for _, v := range out["cards"].([]any) {
		card := v.(map[string]any)
		if card["id"] == c {
			matches++
			if card["local_applicability"].(map[string]any)["state"] != "matched" {
				t.Fatal(card)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("classification paths duplicated or lost claim: %d", matches)
	}
	reject(t, s, "worker-a", "node.create", NodeInput{Category: "organization", Kind: "organization", Name: "unauthorized"})
	reject(t, s, "reviewer", "edge.add", EdgeInput{Relation: "contains", From: project, To: org, Reason: "reviewer is not curator"})
	reject(t, s, "curator", "edge.add", EdgeInput{Relation: "contains", From: repo, To: org, Reason: "cycle"})
}

func TestBroaderApplicabilityRequiresAllNamedContexts(t *testing.T) {
	s := fixture(t)
	a, b := scope(), scope()
	b.Repository = "second-repository"
	claimID := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "The named cases agree in two explicit contexts", Kind: "finite_cases", Limitations: "Only the two named contexts", Applicability: &ApplicabilityEnvelope{Version: 2, State: "enumerated", Contexts: []Scope{a, b}}})["id"].(string)
	e1 := evidence(t, s, "worker-a", "pass", NewID())
	e2 := evidence(t, s, "validator", "pass", NewID())
	j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: claimID, Evidence: []string{e1, e2}, Warrant: "Attempt incomplete coverage", Limitations: "Only A observed"})["id"].(string)
	reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "Missing B", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, claimID, "insufficient")
	for _, actor := range []string{"worker-a", "validator"} {
		data := invoke(t, s, actor, "evidence.get", IDInput{ID: e1})["manifest"]
		raw, _ := json.Marshal(data)
		var v EvidenceInput
		_ = json.Unmarshal(raw, &v)
		v.Scope = b
		v.Root = NewID()
		v.Artifact = "https://example.invalid/" + NewID()
		v.Digest = "sha256:" + Digest(NewID())
		id := invoke(t, s, actor, "evidence.submit", v)["id"].(string)
		if actor == "worker-a" {
			e1 = id
		} else {
			e2 = id
		}
	}
	a1 := evidence(t, s, "worker-a", "pass", NewID())
	a2 := evidence(t, s, "validator", "pass", NewID())
	j = invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: claimID, Evidence: []string{a1, a2, e1, e2}, Warrant: "Each enumerated context has a reproduced check", Limitations: "No extrapolation"})["id"].(string)
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "Both complete contexts inspected", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, claimID, "supported")
	if assessment(t, s, claimID)["policy"] != MatrixPolicy {
		t.Fatal("wrong policy for broader envelope")
	}
	unknown := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "Potential memory mechanism", Kind: "generalization", Assumptions: []string{"same mechanism"}, Limitations: "Unvalidated", Applicability: &ApplicabilityEnvelope{Version: 2, State: "unknown", Description: "Potential cross-project applicability"}})["id"].(string)
	posture(t, s, unknown, "unassessed")
	reject(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "missing dimensions", Kind: "finite_cases", Limitations: "none", Applicability: &ApplicabilityEnvelope{Version: 2, State: "enumerated", Contexts: []Scope{{Repository: "x"}}}})
}

func TestMetadataReceiptsAndRevokedReplay(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	key := NewID()
	body := encode(ContextInput{Scope: scope(), ByteBudget: 64000})
	first, err := s.Execute(ctx, "worker-b", "context.get", key, body)
	if err != nil {
		t.Fatal(err)
	}
	node(t, s, "concept", "topic", "Metadata edit")
	replay, err := s.Execute(ctx, "worker-b", "context.get", key, body)
	if err != nil || string(first) != string(replay) {
		t.Fatal("authorized snapshot replay changed", err)
	}
	// Revocation must be checked before replay even though the original command is stored.
	tx, err := s.begin(ctx, "worker-b")
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `UPDATE domains SET auth_epoch=auth_epoch+1 WHERE tenant=$1 AND id=$2`, tx.p.Tenant, tx.p.Domain)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(ctx, "worker-b", "context.get", key, body); err == nil {
		t.Fatal("old response escaped current authorization check")
	}
}
