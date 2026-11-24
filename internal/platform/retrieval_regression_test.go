package platform

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

const naturalGateQuery = "forced-drain aggregate gate: what it proves, reference taken disarmed, duplicate output rows"

func TestPackingDoesNotInventAnOrientationCap(t *testing.T) {
	bundle := Bundle{Cards: []Card{}, Diagnostics: RetrievalDiagnostics{Discovery: "flat", Candidates: 1}}
	card := Card{ID: "large-unmatched", Statement: strings.Repeat("x", 6500), Limitations: "Unvalidated guidance", LocalApplicability: LocalApplicability{State: "unknown"}}
	packContextCards(&bundle, []Card{card}, ContextInput{ByteBudget: 6000}, 100)
	if bundle.Diagnostics.BudgetOmitted != 1 || bundle.Diagnostics.OrientationOmitted != 0 {
		t.Fatal("total byte limit misreported as an unrequested orientation cap", bundle.Diagnostics)
	}
	if len(bundle.Cards) != 0 || len(bundle.Deferred) != 1 || !bundle.Deferred[0].RequiresDetail {
		t.Fatal("oversized assertion was stripped or lost")
	}
}

func TestNaturalQueryFindsQualifiedADRAt6000Bytes(t *testing.T) {
	s := fixture(t)
	rule := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "Condition-triggered defects use test-only forcing knobs, with the reference taken DISARMED. A forcing knob drains a chosen batch. Arming both comparison sides can hide the defect.", Kind: "hypothesis", Limitations: "Documented guidance only; not independently reproduced", Applicability: &ApplicabilityEnvelope{Version: 2, State: "unknown", Description: "Applicability to the current execution is unvalidated"}, Sources: []SourceReference{{URI: "https://example.invalid/repository/blob/pinned/docs/spill-gates.md", Revision: "pinned-source", Fragment: "Disarmed reference rule", Digest: "sha256:" + Digest("source fixture")}}})["id"].(string)
	long := invoke(t, s, "worker-a", "claim.publish", ClaimInput{Statement: "Spill/drain/merge/clone gates that catch duplicate output rows: " + strings.Repeat("aggregate gate details; ", 125), Kind: "finite_cases", Scope: scope(), Limitations: strings.Repeat("These are source-level observations, not executed correctness results. ", 18)})["id"].(string)
	raw, err := s.Execute(context.Background(), "worker-b", "context.get", NewID(), encode(ContextInput{Scope: scope(), Query: naturalGateQuery, ByteBudget: 6000}))
	if err != nil {
		t.Fatal(err)
	}
	var b Bundle
	if err = json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if len(raw) > 6000 || b.Bytes != len(raw) || b.Diagnostics.Discovery != "flat" {
		t.Fatal("budget/default mismatch", len(raw), b.Diagnostics)
	}
	found := false
	for _, card := range b.Cards {
		if card.ID == rule {
			found = true
			if card.LocalApplicability.State != "unknown" || card.Assessment.Posture != "unassessed" || len(card.Sources) != 1 || card.Limitations == "" {
				t.Fatal("qualification lost", card)
			}
		}
	}
	if !found {
		t.Fatalf("ADR starved at unchanged budget: %s", raw)
	}
	for _, ref := range b.Deferred {
		if ref.ID == long && (!ref.RequiresDetail || ref.Fetch != "claim.get") {
			t.Fatal("unsafe navigation fragment")
		}
	}
	// A caller's explicit cap remains a cap; no silent scope widening or override.
	zero := 0
	out := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), Query: naturalGateQuery, ByteBudget: 6000, OrientationPercent: &zero})
	for _, v := range out["cards"].([]any) {
		if v.(map[string]any)["id"] == rule {
			t.Fatal("explicit orientation exclusion ignored")
		}
	}
	legacy := invoke(t, s, "worker-b", "context.get", ContextInput{Scope: scope(), Query: naturalGateQuery, Discovery: "legacy", ByteBudget: 6000})
	if legacy["diagnostics"].(map[string]any)["scope_excluded"].(float64) == 0 {
		t.Fatal("scope filtering was not explained")
	}
	for _, v := range legacy["cards"].([]any) {
		if v.(map[string]any)["id"] == rule {
			t.Fatal("explicit legacy mode changed")
		}
	}
}

// Optional private fixture replay. The fixture is captured through a read-only
// production transaction and is never committed; writes go only to this suite's
// disposable PostgreSQL cluster and a newly provisioned compartment.
func TestReportedRetrievalSnapshot(t *testing.T) {
	path := os.Getenv("HOLOCRON_RETRIEVAL_FIXTURE")
	if path == "" {
		t.Skip("private live-round snapshot not configured")
	}
	s := fixture(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Claims []struct {
			ID   string     `json:"id"`
			Body ClaimInput `json:"body"`
		} `json:"claims"`
		Scope          Scope  `json:"scope"`
		Query          string `json:"query"`
		Rule           string `json:"rule"`
		Summary        string `json:"summary"`
		Investigations []struct {
			ID   string             `json:"id"`
			Body InvestigationInput `json:"body"`
		} `json:"investigations"`
		Attempts []struct {
			ID   string       `json:"id"`
			Body AttemptInput `json:"body"`
		} `json:"attempts"`
		Outcomes []struct {
			ID   string      `json:"id"`
			Body FinishInput `json:"body"`
		} `json:"outcomes"`
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	actor := NewID()
	admin := os.Getenv("HOLOCRON_ADMIN_DATABASE_URL")
	if admin == "" {
		t.Fatal("disposable admin fixture required")
	}
	if err = Provision(ctx, admin, ProvisionInput{Tenant: actor, Domain: actor, ID: actor, Owner: actor, Role: "worker", Token: actor, Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	tx, err := s.begin(ctx, actor)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range data.Claims {
		var previous any
		if c.Body.Previous != "" {
			previous = c.Body.Previous
		}
		if err = tx.record(ctx, "claims", c.ID, c.Body, ",statement,previous", c.Body.Statement, previous); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
	}
	for _, item := range data.Investigations {
		if err = tx.record(ctx, "investigations", item.ID, item.Body, ""); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
	}
	for _, item := range data.Attempts {
		if err = tx.record(ctx, "attempts", item.ID, item.Body, ",investigation", item.Body.Investigation); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
	}
	receipts := map[string]bool{}
	for _, item := range data.Outcomes {
		if !receipts[item.Body.Receipt] {
			if err = tx.record(ctx, "receipts", item.Body.Receipt, map[string]any{"scope": data.Scope, "claims": []string{}}, ",epoch", int64(0)); err != nil {
				_ = tx.Rollback(ctx)
				t.Fatal(err)
			}
			receipts[item.Body.Receipt] = true
		}
		if err = tx.record(ctx, "outcomes", item.ID, item.Body, ",attempt,receipt", item.Body.Attempt, item.Body.Receipt); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	legacy := invoke(t, s, actor, "context.get", ContextInput{Scope: data.Scope, Query: data.Query, ByteBudget: 6000, Discovery: "legacy"})
	legacyIDs := []string{}
	for _, c := range legacy["cards"].([]any) {
		id := c.(map[string]any)["id"].(string)
		legacyIDs = append(legacyIDs, id)
		if id == data.Rule {
			t.Fatal("old scope-only filter unexpectedly found the rule")
		}
	}
	fixed, err := s.Execute(ctx, actor, "context.get", NewID(), encode(ContextInput{Scope: data.Scope, Query: data.Query, ByteBudget: 6000}))
	if err != nil {
		t.Fatal(err)
	}
	deferredSummary := false
	var bundle Bundle
	_ = json.Unmarshal(fixed, &bundle)
	found := false
	for _, card := range bundle.Cards {
		if card.ID == data.Summary {
			deferredSummary = true
		}
		if card.ID == data.Rule {
			found = true
			if card.LocalApplicability.State != "unknown" || card.Assessment.Posture != "unassessed" || len(card.Sources) == 0 {
				t.Fatal("source promoted or provenance lost")
			}
		}
	}
	if !found || len(fixed) > 6000 {
		t.Fatalf("exact reported query failed at same budget: %s", fixed)
	}
	for _, ref := range bundle.Deferred {
		if ref.ID == data.Summary && ref.RequiresDetail && ref.Fetch == "claim.get" {
			deferredSummary = true
		}
	}
	if !deferredSummary {
		t.Fatal("matched summary was neither returned nor available for progressive detail")
	}
	ids := []string{}
	for _, card := range bundle.Cards {
		ids = append(ids, card.ID)
	}
	t.Logf("legacy cards=%v; fixed bytes=%d cards=%v deferred=%d diagnostics=%+v", legacyIDs, len(fixed), ids, len(bundle.Deferred), bundle.Diagnostics)
}
