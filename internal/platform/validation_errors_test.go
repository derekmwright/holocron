package platform

import (
	"context"
	"strings"
	"testing"
)

func TestWorkerValidationErrorsExplainRecovery(t *testing.T) {
	s := fixture(t)
	for _, test := range []struct {
		name, op string
		input    any
		want     []string
	}{
		{"arm", "attempt.start", AttemptInput{Investigation: "missing", Model: "recorded-model", Harness: "Claude", TaskFamily: "source-audit", Arm: "holocron"}, []string{"arm", "A, B, C, demo"}},
		{"blocked outcome", "attempt.finish", FinishInput{Attempt: "missing", Receipt: "missing", Disposition: "Tooling blocked", Outcome: "blocked"}, []string{"outcome", "failed", "partial", "abandoned"}},
		{"usage", "attempt.finish", FinishInput{Attempt: "missing", Receipt: "missing", Disposition: "failed execution", Outcome: "failed", Tokens: -1}, []string{"tokens", "non-negative"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := s.Execute(context.Background(), "worker-a", test.op, NewID(), encode(test.input))
			if err == nil {
				t.Fatal("invalid request accepted")
			}
			for _, part := range test.want {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("not recoverable: %v", err)
				}
			}
		})
	}
	exact := scope()
	exact.Cases = nil
	if err := exact.validate(); err == nil || !strings.Contains(err.Error(), "applicability") {
		t.Fatal("source-only recovery path omitted", err)
	}
	source := SourceReference{URI: "docs/adr/example.md", Revision: "pinned", Fragment: "section", Digest: "sha256:" + Digest("bytes")}
	err := validateSources([]SourceReference{source})
	if err == nil || !strings.Contains(err.Error(), "sources[0].uri") || !strings.Contains(err.Error(), "scheme") || strings.Contains(err.Error(), "credentials") {
		t.Fatal("wrong source error", err)
	}
	source.URI = "https://example.invalid/doc"
	source.Digest = Digest("bytes")
	err = validateSources([]SourceReference{source})
	if err == nil || !strings.Contains(err.Error(), "sha256:") || !strings.Contains(err.Error(), "64") {
		t.Fatal("digest recovery hidden", err)
	}
	source.URI = "https://user:never-echo-this@example.invalid/doc"
	source.Digest = "sha256:" + Digest("bytes")
	err = validateSources([]SourceReference{source})
	if err == nil || strings.Contains(err.Error(), "never-echo-this") {
		t.Fatal("credential diagnostic unsafe", err)
	}
}
