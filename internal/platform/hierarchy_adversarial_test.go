package platform

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func setupHierarchy(t *testing.T, s *Service) (string, string, string, string, string) {
	t.Helper()
	org := node(t, s, "organization", "organization", "Engineering "+NewID())
	project := node(t, s, "organization", "project", "Wadjet")
	repo := invoke(t, s, "curator", "node.create", NodeInput{Category: "organization", Kind: "repository", Name: "Wadjet", Repository: scope().Repository})["id"].(string)
	component := node(t, s, "organization", "operator-family", "Hash aggregation")
	edge(t, s, "contains", org, project)
	edge(t, s, "contains", project, repo)
	edge(t, s, "contains", repo, component)
	concept := node(t, s, "concept", "mechanism", "Memory management")
	return org, project, repo, component, concept
}
func place(t *testing.T, s *Service, claimID, target string) string {
	id := invoke(t, s, "worker-a", "association.propose", AssociationInput{Kind: "placement", Claim: claimID, Target: target, Steward: "worker-a", Reason: "Accountable home"})["id"].(string)
	invoke(t, s, "curator", "association.review", AssociationReviewInput{ID: id, Verdict: "accepted", Reason: "Accepted responsibility in this domain"})
	return id
}
func TestHierarchyOrientationLateralTransferAndBudget(t *testing.T) {
	s := fixture(t)
	org, project, repo, component, memory := setupHierarchy(t, s)
	local := claim(t, s, "Aggregation memory scoped observation "+NewID())
	support(t, s, local)
	place(t, s, local, component)
	associate(t, s, "claim_concept", local, "", memory)
	broad := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "Memory pressure might share a mechanism", Kind: "generalization", Assumptions: []string{"mechanism may transfer"}, Limitations: "Not locally established", Applicability: &ApplicabilityEnvelope{Version: 2, State: "unknown", Description: "Possible organization-wide principle"}})["id"].(string)
	place(t, s, broad, org)
	associate(t, s, "claim_concept", broad, "", memory)
	irrelevant := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "Unrelated office decoration", Kind: "hypothesis", Limitations: "Unrelated", Applicability: &ApplicabilityEnvelope{Version: 2, State: "unknown", Description: "Unknown"}})["id"].(string)
	place(t, s, irrelevant, org)
	working := &WorkingContext{Organization: org, Project: project, Repository: repo, Components: []string{component}, Concepts: []string{memory}}
	bundle := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), Working: working, ByteBudget: 64000})
	found := map[string]map[string]any{}
	for _, value := range bundle["cards"].([]any) {
		card := value.(map[string]any)
		found[card["id"].(string)] = card
	}
	if found[local] == nil || found[broad] == nil || found[irrelevant] != nil {
		t.Fatal("bad contextual selection", found)
	}
	if found[broad]["local_applicability"].(map[string]any)["state"] != "unknown" {
		t.Fatal("broad placement became applicable")
	}
	// A concept-only request from a different repository still discovers the source,
	// but never turns a supported source finding into an established local finding.
	other := scope()
	other.Repository = "transit-fixture"
	lateral := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: other, Working: &WorkingContext{Concepts: []string{memory}}, ByteBudget: 64000})
	seen := false
	for _, value := range lateral["cards"].([]any) {
		card := value.(map[string]any)
		if card["id"] == local {
			seen = true
			if card["local_applicability"].(map[string]any)["state"] != "mismatched" || card["assessment"].(map[string]any)["posture"] != "supported" {
				t.Fatal(card)
			}
		}
	}
	if !seen {
		t.Fatal("lateral discovery lost")
	}
	for _, budget := range []int{512, 1200, 4000} {
		raw, err := s.Execute(context.Background(), "worker-b", "context.get", NewID(), encode(ContextInput{Scope: scope(), Working: working, ByteBudget: budget}))
		if err != nil {
			if budget == 512 {
				continue
			}
			t.Fatal(err)
		}
		if len(raw) > budget {
			t.Fatal("budget overflow")
		}
		var b Bundle
		_ = json.Unmarshal(raw, &b)
		for _, card := range b.Cards {
			if card.Limitations == "" || card.LocalApplicability.State == "" {
				t.Fatal("qualification stripped")
			}
		}
	}
	reject(t, s, "worker-b", "context.resolve", ResolveInput{Working: WorkingContext{Organization: component, Project: project}, Scope: func() *Scope { x := scope(); return &x }()})
}
func TestHierarchyProposalsAndMultiplePathsDoNotAmplify(t *testing.T) {
	s := fixture(t)
	a := node(t, s, "concept", "topic", "A")
	b := node(t, s, "concept", "topic", "B")
	root := node(t, s, "concept", "topic", "Root")
	edge(t, s, "broader", root, a)
	edge(t, s, "broader", root, b)
	edge(t, s, "related", a, b)
	c := claim(t, s, "Multi-path observation "+NewID())
	support(t, s, c)
	proposal := invoke(t, s, "worker-a", "association.propose", AssociationInput{Kind: "claim_concept", Claim: c, Target: a, Reason: "proposal only"})["id"].(string)
	get := func() map[string]any {
		return invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), Working: &WorkingContext{Concepts: []string{a, b}}, ByteBudget: 64000})
	}
	for _, v := range get()["cards"].([]any) {
		if v.(map[string]any)["id"] == c {
			t.Fatal("unreviewed association entered curated retrieval")
		}
	}
	invoke(t, s, "curator", "association.review", AssociationReviewInput{ID: proposal, Verdict: "accepted", Reason: "curated"})
	associate(t, s, "claim_concept", c, "", b)
	count := 0
	for _, v := range get()["cards"].([]any) {
		if v.(map[string]any)["id"] == c {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("duplicate discovery: %d", count)
	}
	reject(t, s, "curator", "edge.add", EdgeInput{Relation: "broader", From: a, To: root, Reason: "taxonomy cycle"})
	posture(t, s, c, "supported")
}
func TestConcurrentHierarchyCyclesAndMetadataMove(t *testing.T) {
	s := fixture(t)
	a := node(t, s, "organization", "project", "A")
	b := node(t, s, "organization", "project", "B")
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		wg.Add(1)
		go func(pair [2]string) {
			defer wg.Done()
			<-start
			_, err := s.Execute(context.Background(), "curator", "edge.add", NewID(), encode(EdgeInput{Relation: "contains", From: pair[0], To: pair[1], Reason: "concurrent cycle test"}))
			results <- err
		}(pair)
	}
	close(start)
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted %d cyclic edges", accepted)
	}
	org, project, repo, _, _ := setupHierarchy(t, s)
	newOrg := node(t, s, "organization", "business-unit", "New parent")
	hierarchyData := invoke(t, s, "curator", "hierarchy.get", HierarchyLookup{ID: project})
	oldEdge := ""
	for _, v := range hierarchyData["edges"].([]any) {
		e := v.(map[string]any)
		if e["from"] == org && e["to"] == project {
			oldEdge = e["id"].(string)
		}
	}
	if oldEdge == "" {
		t.Fatal("edge missing")
	}
	c := claim(t, s, "Stable finding "+NewID())
	support(t, s, c)
	placement := place(t, s, c, repo)
	before := assessment(t, s, c)
	invoke(t, s, "curator", "edge.add", EdgeInput{Relation: "contains", From: newOrg, To: project, Replace: oldEdge, Reason: "reviewed restructuring"})
	invoke(t, s, "curator", "placement.transfer", PlacementTransferInput{Previous: placement, Target: project, Steward: "worker-b", Reason: "new accountable steward"})
	invoke(t, s, "curator", "node.revise", NodeRevisionInput{ID: repo, ExpectedVersion: 1, Name: "Renamed repository", Repository: scope().Repository, Reason: "rename, no code change"})
	reject(t, s, "curator", "node.revise", NodeRevisionInput{ID: repo, ExpectedVersion: 1, Name: "stale rename", Reason: "stale"})
	after := assessment(t, s, c)
	if before["epoch"] != after["epoch"] || after["posture"] != "supported" {
		t.Fatal("restructuring altered evidence")
	}
	result := invoke(t, s, "worker-b", "context.resolve", ResolveInput{Working: WorkingContext{Organization: newOrg, Project: project, Repository: repo}})
	if result["metadata_epoch"] == nil {
		t.Fatal(result)
	}
	previous := invoke(t, s, "worker-b", "hierarchy.get", HierarchyLookup{ID: project})
	foundRetraction := false
	for _, v := range previous["edge_history"].([]any) {
		e := v.(map[string]any)
		if e["id"] == oldEdge && e["retracted"] == true {
			foundRetraction = true
		}
	}
	if !foundRetraction {
		t.Fatal("old containment history disappeared")
	}
	renamed := invoke(t, s, "worker-b", "hierarchy.get", HierarchyLookup{ID: repo})
	if renamed["initial"].(map[string]any)["name"] != "Wadjet" || len(renamed["history"].([]any)) != 1 {
		t.Fatal("rename history lost", renamed)
	}
}
func TestUnknownApplicabilityCannotCompleteAsLocalSupport(t *testing.T) {
	s := fixture(t)
	concept := node(t, s, "concept", "topic", "Transfer")
	c := claim(t, s, "Established only in the source context "+NewID())
	support(t, s, c)
	associate(t, s, "claim_concept", c, "", concept)
	other := scope()
	other.Repository = "target-repository"
	contract := invoke(t, s, "worker-b", "investigation.open", InvestigationInput{Objective: "transfer", Question: "is it applicable?", Scope: other, Criteria: "local support", BudgetTokens: 1000, Deadline: time.Now().Add(time.Hour), RequireSupported: true})["id"].(string)
	attempt := invoke(t, s, "worker-b", "attempt.start", AttemptInput{Investigation: contract, Model: "fixture", Harness: "test", Arm: "demo", TaskFamily: "transfer"})["id"].(string)
	receipt := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: other, Working: &WorkingContext{Concepts: []string{concept}}, ByteBudget: 64000})["receipt"].(string)
	finish := FinishInput{Attempt: attempt, Receipt: receipt, Outcome: "completed", Disposition: "only orientation", Applied: []string{c}}
	reject(t, s, "worker-b", "attempt.finish", finish)
	finish.Applied = nil
	finish.Considered = []string{c}
	invoke(t, s, "worker-b", "attempt.finish", finish)
}
func TestHierarchyAuthorizationAndRevokedCuration(t *testing.T) {
	s := fixture(t)
	c := node(t, s, "concept", "topic", "Private concept")
	reject(t, s, "outsider", "hierarchy.get", HierarchyLookup{ID: c})
	reject(t, s, "outsider", "context.resolve", ResolveInput{Working: WorkingContext{Concepts: []string{c}}})
	reject(t, s, "worker-a", "association.propose", map[string]any{"kind": "claim_concept", "claim": "absent", "target": c, "reason": "bad", "accepted": true})
	admin := os.Getenv("HOLOCRON_ADMIN_DATABASE_URL")
	if admin == "" {
		t.Skip("admin fixture needed")
	}
	id := NewID()
	ctx := context.Background()
	provision := ProvisionInput{Tenant: "demo", Domain: "wadjet", ID: id, Owner: id, Role: "worker", Token: id, Expires: time.Now().Add(time.Hour), Curator: true}
	if err := Provision(ctx, admin, provision); err != nil {
		t.Fatal(err)
	}
	key := NewID()
	body := encode(NodeInput{Category: "concept", Kind: "topic", Name: "Recorded before grant revocation"})
	raw, err := s.Execute(ctx, id, "node.create", key, body)
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]string
	_ = json.Unmarshal(raw, &record)
	if err = SetCuration(ctx, admin, "demo", "wadjet", id, false); err != nil {
		t.Fatal(err)
	}
	if err = Provision(ctx, admin, provision); err != nil {
		t.Fatal("exact provisioning retry failed", err)
	}
	altered := provision
	altered.Token = NewID()
	if err = Provision(ctx, admin, altered); err == nil {
		t.Fatal("provisioning retry replaced credentials")
	}
	if _, err = s.Execute(ctx, id, "node.create", key, body); err == nil {
		t.Fatal("revoked response replayed")
	}
	reject(t, s, id, "node.create", NodeInput{Category: "concept", Kind: "topic", Name: "new unauthorized mutation"})
	// Revoking curation does not erase the successful mutation or revoke domain read.
	invoke(t, s, id, "hierarchy.get", HierarchyLookup{ID: record["id"]})
}
func TestBoundedGeneralizationRevalidatesThroughPremises(t *testing.T) {
	s := fixture(t)
	source := claim(t, s, "Source finite finding "+NewID())
	support(t, s, source)
	c := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "Bounded synthesis over explicitly listed conditions", Kind: "generalization", Assumptions: []string{"same measured mechanism"}, Limitations: "Only enumerated contexts", Applicability: &ApplicabilityEnvelope{Version: 2, State: "enumerated", Contexts: []Scope{scope()}}})["id"].(string)
	a := evidence(t, s, "worker-a", "pass", NewID())
	b := evidence(t, s, "validator", "pass", NewID())
	j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: []string{a, b}, Premises: []string{source}, Warrant: "Bounded empirical synthesis", Limitations: "No universal conclusion", InferenceKind: "bounded_generalization"})["id"].(string)
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "All bounded conditions reviewed", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, c, "supported")
	concept := node(t, s, "concept", "principle", "Organization-wide orientation")
	associate(t, s, "claim_concept", c, "", concept)
	alternativeSource := claim(t, s, "Alternative scoped premise "+NewID())
	support(t, s, alternativeSource)
	alt := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: []string{a, b}, Premises: []string{alternativeSource}, Warrant: "Alternative bounded support", Limitations: "Same finite envelope", InferenceKind: "bounded_generalization"})["id"].(string)
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: alt, Verdict: "accepted", Reason: "Alternative reviewed", CoverageChecked: true, ProvenanceChecked: true})
	invoke(t, s, "worker-a", "record.retract", LifecycleInput{TargetKind: "claim", Target: source, Reason: "Lost source support"})
	posture(t, s, c, "supported")
	invoke(t, s, "worker-a", "record.retract", LifecycleInput{TargetKind: "claim", Target: alternativeSource, Reason: "Other prerequisite unavailable"})
	posture(t, s, c, "insufficient")
	if assessment(t, s, c)["health"] != "revalidation_required" {
		t.Fatal("generalization lost dependency lineage")
	}
	bad := &ApplicabilityEnvelope{Version: 2, State: "enumerated", Contexts: []Scope{scope(), scope()}}
	reject(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "duplicated contexts", Kind: "finite_cases", Limitations: "none", Applicability: bad})
	bad = &ApplicabilityEnvelope{Version: 2, State: "unconstrained", Description: "trust me"}
	reject(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "wildcard", Kind: "finite_cases", Limitations: "none", Applicability: bad})
	if !strings.Contains(Policy, "v1") {
		t.Fatal("legacy policy identifier changed")
	}
}

func TestSuccessorRequiresExplicitCuratedPlacement(t *testing.T) {
	s := fixture(t)
	home := node(t, s, "organization", "organization", "Organization home")
	topic := node(t, s, "concept", "topic", "Curated subject")
	old := claim(t, s, "Curated narrow claim "+NewID())
	place(t, s, old, home)
	associate(t, s, "claim_concept", old, "", topic)
	next := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "A newly revised assertion", Kind: "finite_cases", Scope: scope(), Limitations: "New assessment required", Previous: old})["id"].(string)
	card := invoke(t, s, "worker-b", "claim.get", IDInput{ID: next})
	if card["home"] != nil || card["classifications"] != nil || card["family_id"] != old {
		t.Fatal("revision acquired unreviewed placement or lost family identity", card)
	}
	posture(t, s, next, "unassessed")
}

func TestMatrixCasePartitionAndOutOfScopeCounterexample(t *testing.T) {
	s := fixture(t)
	x, y := scope(), scope()
	x.Cases = []string{"group-null"}
	y.Cases = []string{"group-decimal"}
	c := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "Two explicitly enumerated case contexts", Kind: "finite_cases", Limitations: "Finite matrix", Applicability: &ApplicabilityEnvelope{Version: 2, State: "enumerated", Contexts: []Scope{x, y}}})["id"].(string)
	evidenceIDs := []string{}
	for _, sc := range []Scope{x, y} {
		for _, actor := range []string{"worker-a", "validator"} {
			v := EvidenceInput{Scope: sc, Root: NewID(), Artifact: "https://example.invalid/" + NewID(), Digest: "sha256:" + Digest(NewID()), ResultDigest: "sha256:" + Digest("pass"), ExpectedDigest: "sha256:" + Digest("pass"), Method: "case-matrix/v1", Command: "bounded case probe", OracleSource: "oracle-v1", Outcome: "pass", ObservedAt: time.Now(), Cases: sc.Cases, TriggerCount: 1, Provenance: "known", Limitations: "finite", Shared: []string{"oracle"}}
			evidenceIDs = append(evidenceIDs, invoke(t, s, actor, "evidence.submit", v)["id"].(string))
		}
	}
	j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: evidenceIDs, Warrant: "Independent case contexts are covered separately", Limitations: "Finite"})["id"].(string)
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "Both case partitions reproduced", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, c, "supported")
	failures := []string{}
	for _, actor := range []string{"worker-c", "validator"} {
		sc := scope()
		sc.Cases = append(sc.Cases, "outside-envelope")
		v := EvidenceInput{Scope: sc, Root: NewID(), Artifact: "https://example.invalid/" + NewID(), Digest: "sha256:" + Digest(NewID()), ResultDigest: "sha256:" + Digest("mismatch"), ExpectedDigest: "sha256:" + Digest("pass"), Method: "case-matrix/v1", Command: "includes outside case", OracleSource: "oracle-v1", Outcome: "fail", ObservedAt: time.Now(), Cases: sc.Cases, TriggerCount: 1, Provenance: "known", Limitations: "Failure not isolated", Shared: []string{"oracle"}}
		failures = append(failures, invoke(t, s, actor, "evidence.submit", v)["id"].(string))
	}
	ch := invoke(t, s, "worker-c", "challenge.submit", ChallengeInput{Claim: c, Kind: "counterexample", Evidence: failures, Reason: "An aggregate failed but it included unclaimed cases"})["id"].(string)
	reject(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "challenge", Target: ch, Verdict: "accepted", Reason: "Cannot locate failure in scope", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, c, "supported")
}

func TestMetadataMutationFencesCompletionAndBoundsRollback(t *testing.T) {
	s := fixture(t)
	contract := invoke(t, s, "worker-b", "investigation.open", InvestigationInput{Objective: "metadata receipt", Question: "bounded", Scope: scope(), Criteria: "Fresh context", BudgetTokens: 100, Deadline: time.Now().Add(time.Hour)})["id"].(string)
	attempt := invoke(t, s, "worker-b", "attempt.start", AttemptInput{Investigation: contract, Model: "fixture", Harness: "test", Arm: "demo", TaskFamily: "metadata"})["id"].(string)
	receipt := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), ByteBudget: 64000})["receipt"].(string)
	node(t, s, "concept", "topic", "New curated metadata")
	finish := FinishInput{Attempt: attempt, Receipt: receipt, Outcome: "completed", Disposition: "No discoveries"}
	reject(t, s, "worker-b", "attempt.finish", finish)
	finish.Receipt = invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), ByteBudget: 64000})["receipt"].(string)
	invoke(t, s, "worker-b", "attempt.finish", finish)
	ctx := context.Background()
	tx, err := s.begin(ctx, "curator")
	if err != nil {
		t.Fatal(err)
	}
	h, err := tx.hierarchy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	before := tx.metadataEpoch
	_ = tx.Rollback(ctx)
	s.MaxHierarchyRecords = len(h.Nodes) + len(h.Edges) + len(h.Associations)
	reject(t, s, "curator", "node.create", NodeInput{Category: "concept", Kind: "topic", Name: "over-bound mutation"})
	tx, err = s.begin(ctx, "curator")
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if tx.metadataEpoch != before {
		t.Fatal("partial metadata mutation committed")
	}
}

func TestCredentialExpiryDoesNotEraseHistoricalEvidence(t *testing.T) {
	s := fixture(t)
	adminURL := os.Getenv("HOLOCRON_ADMIN_DATABASE_URL")
	if adminURL == "" {
		t.Skip("admin fixture required")
	}
	ctx := context.Background()
	actor := NewID()
	if err := Provision(ctx, adminURL, ProvisionInput{Tenant: "demo", Domain: "wadjet", ID: actor, Owner: actor, Role: "validator", Token: actor, Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	c := claim(t, s, "Historical evidence outlives credential "+NewID())
	a := evidence(t, s, "worker-a", "pass", NewID())
	b := evidence(t, s, actor, "pass", NewID())
	j := invoke(t, s, "worker-a", "justification.propose", JustificationInput{Claim: c, Evidence: []string{a, b}, Warrant: "Finite cases", Limitations: "Finite"})["id"].(string)
	invoke(t, s, "reviewer", "review.record", ReviewInput{TargetKind: "justification", Target: j, Verdict: "accepted", Reason: "Validated while identity active", CoverageChecked: true, ProvenanceChecked: true})
	posture(t, s, c, "supported")
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	if _, err = admin.Exec(ctx, `UPDATE holocron.principals SET expires_at=clock_timestamp()-interval '1 second' WHERE tenant='demo' AND domain='wadjet' AND id=$1`, actor); err != nil {
		t.Fatal(err)
	}
	reject(t, s, actor, "claim.get", IDInput{ID: c})
	posture(t, s, c, "supported")
	if err = Revoke(ctx, adminURL, "demo", "wadjet", actor); err != nil {
		t.Fatal(err)
	}
	posture(t, s, c, "insufficient")
}

func TestImportedSourcesRemainUnassessedAndHiddenMetadataIsInvisible(t *testing.T) {
	s := fixture(t)
	concept := node(t, s, "concept", "topic", "Imported spill guidance")
	claimID := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "A source reports a spill coverage requirement", Kind: "hypothesis", Limitations: "Source report only; execution not reproduced", Applicability: &ApplicabilityEnvelope{Version: 2, State: "unknown", Description: "Current implementation applicability unknown"}, Sources: []SourceReference{{URI: "https://example.invalid/repository/blob/fixed/adr.md", Revision: "fixed-source-version", Fragment: "Decision 5", Digest: "sha256:" + Digest("source fixture")}}})["id"].(string)
	associate(t, s, "claim_concept", claimID, "", concept)
	get := func() map[string]any {
		return invoke(t, s, "worker-b", "context.get", ContextInput{OrientationOnly: true, Working: &WorkingContext{Concepts: []string{concept}}, ByteBudget: 16000})
	}
	before := get()
	cards := before["cards"].([]any)
	if len(cards) != 1 {
		t.Fatal(cards)
	}
	card := cards[0].(map[string]any)
	if card["assessment"].(map[string]any)["posture"] != "unassessed" || card["sources"] == nil {
		t.Fatal("source import became evidence or lost provenance")
	}
	admin := os.Getenv("HOLOCRON_ADMIN_DATABASE_URL")
	if admin == "" {
		t.Skip("admin fixture needed")
	}
	actor := NewID()
	ctx := context.Background()
	if err := Provision(ctx, admin, ProvisionInput{Tenant: "other", Domain: "private", ID: actor, Owner: actor, Role: "worker", Token: actor, Expires: time.Now().Add(time.Hour), Curator: true}); err != nil {
		t.Fatal(err)
	}
	hidden := invoke(t, s, actor, "node.create", NodeInput{Category: "concept", Kind: "secret-topic", Name: "Hidden association must not influence public output"})["id"].(string)
	reject(t, s, "worker-a", "association.propose", AssociationInput{Kind: "claim_concept", Claim: claimID, Target: hidden, Reason: "No cross-boundary classification"})
	after := get()
	// Normalize only receipt/time/accounting fields, which are intentionally fresh
	// for every response. The complete authorized payload must otherwise agree.
	for _, v := range []map[string]any{before, after} {
		delete(v, "receipt")
		delete(v, "snapshot_at")
		delete(v, "bytes")
	}
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatal("hidden metadata influenced visible result")
	}
}
