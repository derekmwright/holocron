package platform

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// These tests deliberately require a real, non-owner PostgreSQL connection.
func fixture(t *testing.T) *Service {
	t.Helper()
	url := os.Getenv("HOLOCRON_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("run scripts/test-integration.sh for real PostgreSQL acceptance tests")
	}
	s, err := Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}
func invoke(t *testing.T, s *Service, token, op string, body any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	out, err := s.Execute(context.Background(), token, op, NewID(), b)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	var result map[string]any
	if err = json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func reject(t *testing.T, s *Service, token, op string, body any) {
	t.Helper()
	b, _ := json.Marshal(body)
	if _, err := s.Execute(context.Background(), token, op, NewID(), b); err == nil {
		t.Fatalf("%s unexpectedly accepted", op)
	}
}
func scope() Scope {
	return Scope{Repository: "wadjet-fixture", Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Environment: "linux-fixture-v1", Configuration: "forced-spill-v1", Dataset: "fixture-v1", Oracle: "oracle-v1", Cases: []string{"group-null", "group-decimal"}}
}
func claim(t *testing.T, s *Service, title string) string {
	return invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: title, Scope: scope(), Kind: "finite_cases", Limitations: "Only the named fixtures; no general correctness claim."})["id"].(string)
}
func evidence(t *testing.T, s *Service, token, outcome, root string) string {
	in := EvidenceInput{Scope: scope(), Root: root, Artifact: "https://example.invalid/artifacts/" + NewID(), Digest: "sha256:" + Digest(NewID()), ResultDigest: "sha256:" + Digest("expected"), ExpectedDigest: "sha256:" + Digest("expected"), Method: "spill-check/v1", Command: "go test ./fixture", OracleSource: "git:oracle-v1", Outcome: outcome, ObservedAt: time.Now().UTC(), Cases: scope().Cases, TriggerCount: 1, Provenance: "known", Limitations: "Shared fixture and oracle", Shared: []string{"test", "oracle", "model", "environment"}}
	if outcome == "fail" {
		in.ResultDigest = "sha256:" + Digest("wrong")
	}
	return invoke(t, s, token, "evidence.submit", in)["id"].(string)
}
func support(t *testing.T, s *Service, c string, premises ...string) string {
	e := evidence(t, s, "worker-a", "pass", NewID())
	r := evidence(t, s, "validator", "pass", NewID())
	j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: []string{e, r}, Premises: premises, Warrant: "Finite-case reproduction with the exact scoped oracle", Limitations: "Shared test and oracle"})["id"].(string)
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "Inspected oracle lineage, coverage, separate execution and scope", CoverageChecked: true, ProvenanceChecked: true})
	return j
}
func assessment(t *testing.T, s *Service, c string) map[string]any {
	return invoke(t, s, "worker-b", "assessment.explain", IDInput{ID: c})
}
func posture(t *testing.T, s *Service, c, want string) {
	t.Helper()
	got := assessment(t, s, c)
	if got["posture"] != want {
		t.Fatalf("want %s: %#v", want, got)
	}
}

func TestWalkingScenario(t *testing.T) {
	s := fixture(t)
	c := claim(t, s, "Forced spill matches oracle "+NewID())
	j := support(t, s, c)
	posture(t, s, c, "supported")
	t.Log("A publishes scoped evidence; independent execution manifest and authorized review yield supported finite cases")
	dependent := claim(t, s, "Optimization prerequisite "+NewID())
	support(t, s, dependent, c)
	posture(t, s, dependent, "supported")
	contract := invoke(t, s, "worker-b", "investigation.open", InvestigationInput{Objective: "Apply existing finite-case knowledge", Question: "Is the prerequisite current?", Scope: scope(), Criteria: "Apply only supported scoped findings", BudgetTokens: 1000, Deadline: time.Now().Add(time.Hour), RequireSupported: true})["id"].(string)
	attempt := invoke(t, s, "worker-b", "attempt.start", AttemptInput{Investigation: contract, Model: "scripted-fixture", Harness: "integration-test", Arm: "demo", TaskFamily: "spill"})["id"].(string)
	bundle := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), Query: "Forced spill matches oracle", ByteBudget: 64000})
	if len(bundle["cards"].([]any)) == 0 {
		t.Fatal("no context")
	}
	t.Log("B retrieves scope-qualified knowledge with evidence and a context receipt")
	invoke(t, s, "worker-b", "attempt.finish", FinishInput{Attempt: attempt, Receipt: bundle["receipt"].(string), Outcome: "completed", Disposition: "Applied the finite-case prerequisite", Applied: []string{c}, Findings: []string{}, Tokens: 42, ElapsedMS: 10})
	bad := evidence(t, s, "worker-c", "fail", NewID())
	replication := evidence(t, s, "validator", "fail", NewID())
	ch := invoke(t, s, "worker-c", "challenge.submit", ChallengeInput{Claim: c, Kind: "counterexample", Evidence: []string{bad, replication}, Reason: "Named scoped case differs from oracle"})["id"].(string)
	posture(t, s, c, "supported")
	t.Log("C files counterevidence; unreviewed challenge remains pending, not a refutation")
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "challenge", Target: ch, Verdict: "accepted", Reason: "Reproduced mismatch; coverage and provenance checked", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, c, "contested")
	posture(t, s, dependent, "insufficient")
	t.Log("Accepted counterevidence makes the claim contested and its dependent revalidation-required")
	if assessment(t, s, dependent)["health"] != "revalidation_required" {
		t.Fatal("dependent must be revalidated")
	}
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "rejected", Reason: "Original engagement counter measured the wrong operator"})
	posture(t, s, c, "refuted")
	updated := invoke(t, s, "worker-d", "claim.get", IDInput{ID: c})
	if updated["assessment"].(map[string]any)["posture"] != "refuted" {
		t.Fatal("subsequent participant received stale knowledge")
	}
	t.Log("Invalid original justification is retained in history; subsequent retrieval explains refutation")
	reject(t, s, "outsider", "assessment.explain", IDInput{ID: c})
	reject(t, s, "worker-a", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "self approval"})
}

func TestIdempotencyAndImmutableRevision(t *testing.T) {
	s := fixture(t)
	in := ClaimInput{Statement: "Pinned claim " + NewID(), Scope: scope(), Kind: "finite_cases", Limitations: "finite"}
	b, _ := json.Marshal(in)
	key := NewID()
	a, e := s.Execute(context.Background(), "worker-a", "claim.publish", key, b)
	if e != nil {
		t.Fatal(e)
	}
	retry, e := s.Execute(context.Background(), "worker-a", "claim.publish", key, b)
	if e != nil || string(a) != string(retry) {
		t.Fatal("retry changed result", e)
	}
	in.Statement = "changed"
	b, _ = json.Marshal(in)
	if _, e = s.Execute(context.Background(), "worker-a", "claim.publish", key, b); e == nil {
		t.Fatal("key reuse accepted")
	}
	var r map[string]any
	_ = json.Unmarshal(a, &r)
	old := r["id"].(string)
	in.Previous = old
	newClaim := invoke(t, s, "worker-a", "claim.publish", in)["id"].(string)
	posture(t, s, newClaim, "unassessed")
	if assessment(t, s, old)["lifecycle"] != "superseded" {
		t.Fatal("prior revision still active")
	}
	reject(t, s, "worker-a", "claim.publish", in)
}
