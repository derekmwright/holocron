package platform

import "fmt"

func (g *graph) supportReasons(j justificationRecord, c claimRecord) []string {
	if j.InferenceKind == "analogy" {
		return []string{"analogy alone cannot establish applicability"}
	}
	if c.Applicability == nil {
		if c.Kind != "finite_cases" {
			return []string{"legacy policy covers finite cases only"}
		}
		return g.evidencePair(j.Evidence, c.Scope, "pass")
	}
	if c.Applicability.State != "enumerated" {
		return []string{"applicability is unvalidated; no positive context coverage is asserted"}
	}
	if c.Kind != "finite_cases" && c.Kind != "generalization" {
		return []string{"hypothesis must be revised into a bounded claim before establishment"}
	}
	if c.Kind == "generalization" && (j.InferenceKind != "bounded_generalization" || len(j.Premises) == 0 || len(c.Assumptions) == 0) {
		return []string{"bounded generalization requires explicit premises, assumptions and inference kind"}
	}
	if j.InferenceKind == "analogy" {
		return []string{"analogy alone cannot establish applicability"}
	}
	reasons := []string{}
	used := map[string]bool{}
	for index, scope := range c.contexts() {
		ids := []string{}
		for _, id := range j.Evidence {
			if e, ok := g.evidence[id]; ok {
				observed := e.Scope
				observed.Cases = e.Cases
				if !observed.covers(scope) {
					continue
				}
				ids = append(ids, id)
				used[id] = true
			}
		}
		for _, reason := range g.evidencePair(ids, scope, "pass") {
			reasons = append(reasons, fmt.Sprintf("context %d: %s", index+1, reason))
		}
	}
	for _, id := range j.Evidence {
		if !used[id] {
			reasons = append(reasons, "evidence outside the declared context matrix: "+id)
		}
	}
	return reasons
}
func (g *graph) counterReasons(ids []string, c claimRecord) []string {
	if c.Applicability == nil {
		return g.evidencePair(ids, c.Scope, "fail")
	}
	if c.Applicability.State != "enumerated" {
		return []string{"cannot establish in-scope falsification for unknown applicability"}
	}
	for _, scope := range c.contexts() {
		if len(g.evidencePair(ids, scope, "fail")) == 0 {
			return nil
		}
	}
	return []string{"counterexample does not reproduce within one declared context"}
}
