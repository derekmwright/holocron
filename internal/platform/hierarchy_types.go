package platform

import (
	"encoding/hex"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
)

const MatrixPolicy = "software-context-matrix/v1"

// Source references preserve import provenance. They are declarations, not
// qualifying execution evidence and never participate in support aggregation.
type SourceReference struct {
	URI      string `json:"uri"`
	Revision string `json:"revision"`
	Fragment string `json:"fragment"`
	Digest   string `json:"digest"`
}

func validateSources(sources []SourceReference) error {
	if len(sources) > 16 {
		return problem("invalid", "at most 16 source references allowed")
	}
	for _, s := range sources {
		if err := required(s.URI, s.Revision, s.Fragment, s.Digest); err != nil {
			return err
		}
		u, err := url.Parse(s.URI)
		if err != nil || u.Scheme == "" || u.User != nil {
			return problem("invalid", "source URI must identify an artifact without embedded credentials")
		}
		decoded, err := hex.DecodeString(strings.TrimPrefix(s.Digest, "sha256:"))
		if err != nil || len(decoded) != 32 || !strings.HasPrefix(s.Digest, "sha256:") {
			return problem("invalid", "source reference requires a sha256 artifact digest")
		}
	}
	return nil
}

// Version 2 deliberately supports enumerated complete contexts or explicit unknown
// applicability. There are no implicit wildcards, executable predicates, or universal claims.
type ApplicabilityEnvelope struct {
	Version     int     `json:"version"`
	State       string  `json:"state"`
	Contexts    []Scope `json:"contexts,omitempty"`
	Description string  `json:"description,omitempty"`
}

func (c ClaimInput) contexts() []Scope {
	if c.Applicability == nil {
		return []Scope{c.Scope}
	}
	return c.Applicability.Contexts
}
func (c ClaimInput) policy() string {
	if c.Applicability != nil {
		return MatrixPolicy
	}
	return Policy
}
func (c ClaimInput) validateApplicability() error {
	if c.Applicability == nil {
		return c.Scope.validate()
	}
	a := c.Applicability
	if a.Version != 2 {
		return problem("invalid", "unsupported applicability version")
	}
	if !scopeEmpty(c.Scope) {
		return problem("invalid", "use either legacy scope or a version 2 envelope, not both")
	}
	if a.State == "unknown" {
		if len(a.Contexts) != 0 || c.Kind == "finite_cases" || strings.TrimSpace(a.Description) == "" {
			return problem("invalid", "unknown applicability requires a hypothesis/generalization description and no asserted contexts")
		}
		return nil
	}
	if a.State != "enumerated" || len(a.Contexts) == 0 || len(a.Contexts) > 16 {
		return problem("invalid", "enumerated applicability requires 1–16 complete contexts")
	}
	seen := map[string]bool{}
	for _, s := range a.Contexts {
		if err := s.validate(); err != nil {
			return err
		}
		copy := s
		copy.Cases = slices.Clone(s.Cases)
		slices.Sort(copy.Cases)
		key := string(encode(copy))
		if seen[key] {
			return problem("invalid", "duplicate applicability context")
		}
		seen[key] = true
	}
	if c.Kind == "generalization" && len(c.Assumptions) == 0 {
		return problem("invalid", "bounded generalization must declare its assumptions")
	}
	return nil
}
func scopeEmpty(s Scope) bool {
	return s.Repository == "" && s.Commit == "" && s.Environment == "" && s.Configuration == "" && s.Dataset == "" && s.Oracle == "" && len(s.Cases) == 0
}

type LocalApplicability struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

func (c ClaimInput) local(scope Scope, orientationOnly bool) LocalApplicability {
	if orientationOnly {
		return LocalApplicability{"unknown", "orientation-only request supplied no verified execution context"}
	}
	if c.Applicability != nil && c.Applicability.State == "unknown" {
		return LocalApplicability{"unknown", "applicability is explicitly unvalidated"}
	}
	for _, s := range c.contexts() {
		if s.covers(scope) {
			return LocalApplicability{"matched", "requested finite conditions are contained in an explicit envelope; placement contributes no support"}
		}
	}
	return LocalApplicability{"mismatched", "no explicit envelope covers the requested conditions; relevant only as qualified orientation or analogy"}
}

type NodeInput struct {
	Category    string `json:"category"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Repository  string `json:"repository,omitempty"`
}
type NodeRevisionInput struct {
	ID              string `json:"id"`
	ExpectedVersion int    `json:"expected_version"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	Repository      string `json:"repository,omitempty"`
	Reason          string `json:"reason"`
}
type EdgeInput struct {
	Relation string `json:"relation"`
	From     string `json:"from"`
	To       string `json:"to"`
	Reason   string `json:"reason"`
	Replace  string `json:"replace,omitempty"`
}
type RetractionInput struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}
type AssociationInput struct {
	Kind    string `json:"kind"`
	Claim   string `json:"claim,omitempty"`
	Entity  string `json:"entity,omitempty"`
	Target  string `json:"target"`
	Steward string `json:"steward,omitempty"`
	Reason  string `json:"reason"`
}
type AssociationReviewInput struct {
	ID      string `json:"id"`
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}
type PlacementTransferInput struct {
	Previous string `json:"previous"`
	Target   string `json:"target"`
	Steward  string `json:"steward"`
	Reason   string `json:"reason"`
}
type HierarchyLookup struct {
	ID    string `json:"id,omitempty"`
	Limit int    `json:"limit,omitempty"`
}
type WorkingContext struct {
	Organization string   `json:"organization,omitempty"`
	Project      string   `json:"project,omitempty"`
	Repository   string   `json:"repository,omitempty"`
	Components   []string `json:"components,omitempty"`
	Concepts     []string `json:"concepts,omitempty"`
}
type ResolveInput struct {
	Working WorkingContext `json:"working"`
	Scope   *Scope         `json:"scope,omitempty"`
}
type ResolvedContext struct {
	DirectEntities   []string `json:"direct_entities"`
	AncestorEntities []string `json:"ancestor_entities"`
	DirectConcepts   []string `json:"direct_concepts"`
	BroaderConcepts  []string `json:"broader_concepts"`
	RelatedConcepts  []string `json:"related_concepts"`
	MetadataEpoch    int64    `json:"metadata_epoch"`
}
type KnowledgeNode struct {
	NodeInput
	ID      string `json:"id"`
	Author  string `json:"author"`
	Version int    `json:"version"`
}
type KnowledgeEdge struct {
	EdgeInput
	ID     string `json:"id"`
	Author string `json:"author"`
}
type KnowledgeAssociation struct {
	AssociationInput
	ID           string `json:"id"`
	Author       string `json:"author"`
	Verdict      string `json:"verdict"`
	ReviewedBy   string `json:"reviewed_by,omitempty"`
	ReviewReason string `json:"review_reason,omitempty"`
}
type hierarchy struct {
	Nodes        map[string]KnowledgeNode
	Edges        map[string]KnowledgeEdge
	Associations map[string]KnowledgeAssociation
}

func unmarshal[T any](raw []byte) (T, error) { var v T; err := json.Unmarshal(raw, &v); return v, err }
