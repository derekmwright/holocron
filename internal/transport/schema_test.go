package transport

import (
	"slices"
	"testing"

	"github.com/derekmwright/holocron/internal/platform"
)

func TestEvidenceSchemaSeparatesOriginFromComparison(t *testing.T) {
	s := toolInputSchema[platform.EvidenceInput]().Properties["arguments"]
	for _, field := range []string{"expected_digest", "result_digest", "oracle_source"} {
		if slices.Contains(s.Required, field) {
			t.Fatalf("capture still requires %s", field)
		}
	}
	schema := toolInputSchema[platform.EvidenceInput]()
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	scope := map[string]any{"repository": "example/repo", "commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "environment": "fixture", "configuration": "fixture", "dataset": "fixture", "oracle": "exploratory-method", "cases": []any{"case"}}
	args := map[string]any{"scope": scope, "root": "observation", "artifact": "file:///retained/probe.json", "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "method": "exploratory", "command": "probe", "outcome": "inconclusive", "observed_at": "2026-11-24T21:00:00Z", "cases": []any{"case"}, "trigger_count": float64(1), "provenance": "known", "limitations": "No comparison", "shared": []any{}}
	if err = resolved.Validate(map[string]any{"key": "capture", "arguments": args}); err != nil {
		t.Fatal("known capture rejected without a golden output", err)
	}
}

func TestToolSchemasExposeLiveWorkerConstraints(t *testing.T) {
	start := toolInputSchema[platform.AttemptInput]().Properties["arguments"]
	if len(start.Properties["arm"].Enum) != 4 {
		t.Fatal("experiment arms hidden from agents")
	}
	finish := toolInputSchema[platform.FinishInput]().Properties["arguments"]
	if len(finish.Properties["outcome"].Enum) != 6 || finish.Properties["tokens"].Minimum == nil {
		t.Fatal("outcomes or usage bounds hidden")
	}
	claim := toolInputSchema[platform.ClaimInput]().Properties["arguments"]
	cases := claim.Properties["scope"].Properties["cases"]
	if cases.MinItems == nil || *cases.MinItems != 1 || cases.MaxItems == nil || *cases.MaxItems != 100 || !cases.UniqueItems {
		t.Fatal("scope case constraints hidden")
	}
	source := claim.Properties["sources"].Items
	if source.Properties["uri"].Pattern == "" || source.Properties["digest"].Pattern == "" {
		t.Fatal("source identity constraints hidden")
	}
}

func TestClaimSchemaDescribesExclusiveApplicabilityPaths(t *testing.T) {
	schema := toolInputSchema[platform.ClaimInput]()
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	scope := map[string]any{"repository": "example/repo", "commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "environment": "fixture", "configuration": "fixture", "dataset": "fixture", "oracle": "fixture", "cases": []any{"named-case"}}
	envelope := map[string]any{"version": float64(2), "state": "unknown", "description": "Runtime applicability unvalidated"}
	for _, test := range []struct {
		name  string
		args  map[string]any
		valid bool
	}{
		{"source-only hypothesis", map[string]any{"statement": "The ADR reports a rule", "kind": "hypothesis", "limitations": "Source only", "applicability": envelope}, true},
		{"legacy exact scope", map[string]any{"statement": "Finite assertion", "kind": "finite_cases", "limitations": "Finite", "scope": scope}, true},
		{"both", map[string]any{"statement": "x", "kind": "hypothesis", "limitations": "x", "scope": scope, "applicability": envelope}, false},
		{"neither", map[string]any{"statement": "x", "kind": "hypothesis", "limitations": "x"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := resolved.Validate(map[string]any{"key": "test", "arguments": test.args})
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t, err=%v", test.valid, err)
			}
		})
	}
}
