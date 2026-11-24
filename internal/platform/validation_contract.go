package platform

import (
	"fmt"
	"slices"
	"strings"
)

const ScopeCasesMin = 1
const ScopeCasesMax = 100
const DigestPattern = `^sha256:[0-9a-fA-F]{64}$`
const URIWithSchemePattern = `^[A-Za-z][A-Za-z0-9+.-]*:`
const GitObjectPattern = `^([0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`

// Shared categorical vocabulary for server validation and advertised schemas.
// Callers receive a copy; transport code cannot change the accepted values.
var inputEnums = map[string][]string{
	"AttemptInput.arm":                  {"A", "B", "C", "demo"},
	"FinishInput.outcome":               {"completed", "failed", "partial", "timed_out", "cancelled", "abandoned"},
	"ClaimInput.kind":                   {"finite_cases", "hypothesis", "generalization"},
	"EvidenceInput.outcome":             {"pass", "fail", "inconclusive", "infrastructure_error", "exhausted"},
	"EvidenceInput.provenance":          {"known", "unknown"},
	"JustificationInput.inference_kind": {"reproduction", "deduction", "analogy", "bounded_generalization"},
	"ChallengeInput.kind":               {"methodology", "counterexample"},
	"ReviewInput.target_kind":           {"justification", "challenge"},
	"ReviewInput.verdict":               {"accepted", "rejected", "inconclusive"},
	"LifecycleInput.target_kind":        {"claim", "evidence"},
	"ContextInput.discovery":            {"legacy", "flat", "hierarchical"},
	"ApplicabilityEnvelope.state":       {"unknown", "enumerated"},
	"NodeInput.category":                {"organization", "concept"},
	"EdgeInput.relation":                {"contains", "broader", "related", "affiliated"},
	"AssociationInput.kind":             {"claim_concept", "entity_concept", "placement"},
	"AssociationReviewInput.verdict":    {"accepted", "rejected"},
}

func InputEnum(key string) []string { return slices.Clone(inputEnums[key]) }
func choice(key, value string) error {
	values := inputEnums[key]
	if slices.Contains(values, value) {
		return nil
	}
	_, field, _ := strings.Cut(key, ".")
	return problem("invalid", fmt.Sprintf("%s must be one of: %s", field, strings.Join(values, ", ")))
}
