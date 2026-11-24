package platform

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestKnownProvenanceCaptureDoesNotRequireGoldenOutput(t *testing.T) {
	s := fixture(t)
	c := claim(t, s, "An exploratory comparator observation "+NewID())
	v := EvidenceInput{Scope: scope(), Root: NewID(), Artifact: "file:///retained-captures/exploratory-probe.json", Digest: "sha256:" + Digest("actual capture fixture"), Method: "Exploratory probe; expected relation not defined", Command: "probe comparator", Outcome: "inconclusive", ObservedAt: time.Now(), Cases: scope().Cases, TriggerCount: 1, Provenance: "known", Limitations: "Origin recorded; no comparison basis or independent reproduction"}
	evidenceID := invoke(t, s, "worker-a", "evidence.submit", v)["id"].(string)
	j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: []string{evidenceID}, Warrant: "Record exploratory evidence without establishment", Limitations: "Comparison and replication missing"})["id"].(string)
	a := assessment(t, s, c)
	if a["posture"] != "insufficient" {
		t.Fatal("capture was promoted", a)
	}
	raw, _ := json.Marshal(a)
	if strings.Contains(string(raw), "unavailable evidence") || !strings.Contains(string(raw), "comparison oracle not specified") {
		t.Fatalf("capture mislabeled: %s", raw)
	}
	profile := a["evidence_profile"].([]any)
	if len(profile) != 1 {
		t.Fatal("recorded execution indistinguishable from no evidence", a)
	}
	p := profile[0].(map[string]any)
	if p["declared_provenance"] != "known" || p["reported_outcome"] != "inconclusive" || p["expected_digest_recorded"] != false {
		t.Fatal(p)
	}
	reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "Cannot waive missing comparison", CoverageChecked: true, ProvenanceChecked: true})
	v.Outcome = "pass"
	v.OracleSource = "Declared property without expected-result capture"
	v.ResultDigest = "sha256:" + Digest("observed")
	other := invoke(t, s, "validator", "evidence.submit", v)["id"].(string)
	next := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: []string{other}, Warrant: "Comparison incomplete", Limitations: "No expected digest"})["id"].(string)
	reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: next, Verdict: "accepted", Reason: "No expected basis", CoverageChecked: true, ProvenanceChecked: true})
}

func TestObservationErrorsNameTheFailingField(t *testing.T) {
	s := fixture(t)
	v := EvidenceInput{Scope: scope(), Root: NewID(), Artifact: "file:///retained/probe.json", Digest: "sha256:" + Digest("capture"), Method: "fixture", Command: "probe", Outcome: "inconclusive", ObservedAt: time.Now().Add(6 * time.Minute), Cases: scope().Cases, TriggerCount: 1, Provenance: "known", Limitations: "No verification"}
	_, err := s.Execute(context.Background(), "worker-a", "evidence.submit", NewID(), encode(v))
	if err == nil || !strings.Contains(err.Error(), "observed_at") || !strings.Contains(err.Error(), "server_time=") || strings.Contains(err.Error(), "trigger_count") {
		t.Fatal("ambiguous future timestamp error", err)
	}
	v.ObservedAt = time.Now()
	v.TriggerCount = -1
	_, err = s.Execute(context.Background(), "worker-a", "evidence.submit", NewID(), encode(v))
	if err == nil || !strings.Contains(err.Error(), "trigger_count") {
		t.Fatal(err)
	}
	v.TriggerCount = 1
	v.Artifact = "/retained/probe.json"
	_, err = s.Execute(context.Background(), "worker-a", "evidence.submit", NewID(), encode(v))
	if err == nil || !strings.Contains(err.Error(), "file:///") {
		t.Fatal("local capture guidance absent", err)
	}
}
