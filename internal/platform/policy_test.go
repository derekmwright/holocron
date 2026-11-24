package platform

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestSessionDescriptionUsesAuthenticatedIdentity(t *testing.T) {
	s := fixture(t)
	worker := invoke(t, s, "worker-a", "session.describe", struct{}{})
	identity := worker["identity"].(map[string]any)
	if identity["id"] != "worker-a" || identity["role"] != "worker" || identity["tenant"] != "demo" {
		t.Fatalf("wrong identity: %#v", identity)
	}
	build := worker["build"].(map[string]any)
	if len(build["binary_digest"].(string)) != 71 || build["go_version"] == "" {
		t.Fatalf("missing build identity: %#v", build)
	}
	reviewer := invoke(t, s, "reviewer", "session.describe", struct{}{})
	if reviewer["identity"].(map[string]any)["role"] != "reviewer" {
		t.Fatal("role not authenticated")
	}
	reject(t, s, "worker-a", "session.describe", map[string]string{"role": "reviewer"})
}

func TestDerivedReportsAndSameArtifactsDoNotReplicate(t *testing.T) {
	s := fixture(t)
	for _, mode := range []string{"derived", "same-digest"} {
		t.Run(mode, func(t *testing.T) {
			c := claim(t, s, "derived report "+NewID())
			original := evidence(t, s, "worker-a", "pass", NewID())
			manifest := invoke(t, s, "worker-a", "evidence.get", IDInput{ID: original})["manifest"]
			raw, _ := json.Marshal(manifest)
			var in EvidenceInput
			_ = json.Unmarshal(raw, &in)
			in.Root = NewID()
			in.Artifact = "https://example.invalid/report/" + NewID()
			if mode == "derived" {
				in.Digest = "sha256:" + Digest(NewID())
				in.DerivedFrom = []string{original}
			}
			report := invoke(t, s, "validator", "evidence.submit", in)["id"].(string)
			j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: []string{original, report}, Warrant: "reports are not replications", Limitations: "shared lineage"})["id"].(string)
			reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "attempted amplification", CoverageChecked: true, ProvenanceChecked: true})
		})
	}
}
func TestPendingPremiseAndReadConsistency(t *testing.T) {
	s := fixture(t)
	c := claim(t, s, "parent "+NewID())
	support(t, s, c)
	d := claim(t, s, "child "+NewID())
	support(t, s, d, c)
	invoke(t, s, "worker-c", "challenge.submit", ChallengeInput{Claim: c, Kind: "counterexample", Reason: "unverified potential failure"})
	posture(t, s, d, "supported")
	if !assessment(t, s, d)["pending_challenge"].(bool) {
		t.Fatal("pending premise warning disappeared")
	}
	// The writer holds the domain lock. Concurrent context must wait and then see
	// the committed lifecycle plus reassessed descendants, not a torn snapshot.
	ctx := context.Background()
	tx, err := s.begin(ctx, "worker-a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO lifecycle(tenant,domain,id,author,claim,state,reason) VALUES($1,$2,$3,$4,$5,'retracted','coverage corrected')`, tx.p.Tenant, tx.p.Domain, NewID(), tx.p.ID, c)
	if err != nil {
		t.Fatal(err)
	}
	tx.epoch++
	_, err = tx.Exec(ctx, `UPDATE domains SET epoch=$3 WHERE tenant=$1 AND id=$2`, tx.p.Tenant, tx.p.Domain, tx.epoch)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.recompute(ctx); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	done := make(chan struct{})
	var result json.RawMessage
	var readErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(started)
		result, readErr = s.Execute(ctx, "worker-b", "assessment.explain", NewID(), encode(IDInput{ID: d}))
		close(done)
	}()
	<-started
	select {
	case <-done:
		t.Fatal("read escaped domain lock")
	default:
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if readErr != nil {
		t.Fatal(readErr)
	}
	var a Assessment
	_ = json.Unmarshal(result, &a)
	if a.Posture != "insufficient" || a.Health != "revalidation_required" {
		t.Fatalf("torn assessment: %s", result)
	}
	history := invoke(t, s, "worker-b", "assessment.history", IDInput{ID: d})["assessments"].([]any)
	found := false
	for _, v := range history {
		if v.(map[string]any)["posture"] == "supported" {
			found = true
		}
	}
	if !found {
		t.Fatal("historical support erased")
	}
}
func TestUnknownAndOutOfScopeCounterexamplesStayPending(t *testing.T) {
	s := fixture(t)
	c := claim(t, s, "counter scope "+NewID())
	support(t, s, c)
	ch := invoke(t, s, "worker-c", "challenge.submit", ChallengeInput{Claim: c, Kind: "counterexample", Reason: "no reproducer yet"})["id"].(string)
	reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "challenge", Target: ch, Verdict: "accepted", Reason: "without evidence", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, c, "supported")
	// Overlapping failing batches are not a controlled reproduction of the same
	// counterexample, even when an underspecified result digest happens to match.
	negativeA := evidence(t, s, "worker-c", "fail", NewID())
	raw, _ := json.Marshal(invoke(t, s, "worker-c", "evidence.get", IDInput{ID: negativeA})["manifest"])
	var negative EvidenceInput
	_ = json.Unmarshal(raw, &negative)
	negative.Cases = []string{"group-null"}
	negative.Root = NewID()
	negative.Artifact = "https://example.invalid/" + NewID()
	negative.Digest = "sha256:" + Digest(NewID())
	negativeB := invoke(t, s, "validator", "evidence.submit", negative)["id"].(string)
	batched := invoke(t, s, "worker-c", "challenge.submit", ChallengeInput{Claim: c, Kind: "counterexample", Evidence: []string{negativeA, negativeB}, Reason: "Different failing case batches"})["id"].(string)
	reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "challenge", Target: batched, Verdict: "accepted", Reason: "Not the same tested case set", CoverageChecked: true, ProvenanceChecked: true})
	c2 := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "all possible inputs", Scope: scope(), Kind: "generalization", Limitations: "extrapolation"})["id"].(string)
	e := evidence(t, s, "worker-a", "pass", NewID())
	r := evidence(t, s, "validator", "pass", NewID())
	j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c2, Evidence: []string{e, r}, Warrant: "not universal proof", Limitations: "finite"})["id"].(string)
	reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "finite tests", CoverageChecked: true, ProvenanceChecked: true})
	// Expiry remains checked at retrieval without a periodic worker.
	until := time.Now().Add(-time.Second)
	expired := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "expired", Scope: scope(), Kind: "finite_cases", Limitations: "finite", ReviewAfter: &until})["id"].(string)
	if assessment(t, s, expired)["health"] != "revalidation_required" {
		t.Fatal("expired scope served as current")
	}
}
