package platform

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const Policy = "software-case-reproduction/v1"

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string           { return e.Code + ": " + e.Message }
func problem(code, message string) error { return &Error{code, message} }
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

type Scope struct {
	Repository    string   `json:"repository"`
	Commit        string   `json:"commit"`
	Environment   string   `json:"environment"`
	Configuration string   `json:"configuration"`
	Dataset       string   `json:"dataset"`
	Oracle        string   `json:"oracle"`
	Cases         []string `json:"cases"`
}

func (s Scope) validate() error {
	for _, v := range []string{s.Repository, s.Commit, s.Environment, s.Configuration, s.Dataset, s.Oracle} {
		if strings.TrimSpace(v) == "" || v == "*" || v == "unknown" {
			return problem("invalid", "exact named scope dimensions are required")
		}
	}
	if raw, err := hex.DecodeString(s.Commit); err != nil || (len(raw) != 20 && len(raw) != 32) {
		return problem("invalid", "commit must be a full immutable Git object ID")
	}
	if len(s.Cases) == 0 || len(s.Cases) > 100 {
		return problem("invalid", "scope requires 1–100 named finite cases")
	}
	return unique(s.Cases)
}
func (s Scope) sameEnvironment(t Scope) bool {
	return s.Repository == t.Repository && s.Commit == t.Commit && s.Environment == t.Environment && s.Configuration == t.Configuration && s.Dataset == t.Dataset && s.Oracle == t.Oracle
}
func (s Scope) covers(t Scope) bool {
	if !s.sameEnvironment(t) {
		return false
	}
	for _, c := range t.Cases {
		if !slices.Contains(s.Cases, c) {
			return false
		}
	}
	return true
}
func unique(ids []string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || seen[id] {
			return problem("invalid", "empty or duplicate identifiers")
		}
		seen[id] = true
	}
	return nil
}

type ClaimInput struct {
	Statement   string     `json:"statement"`
	Scope       Scope      `json:"scope"`
	Kind        string     `json:"kind"`
	Limitations string     `json:"limitations"`
	Assumptions []string   `json:"assumptions,omitempty"`
	Concepts    []string   `json:"concepts,omitempty"`
	Previous    string     `json:"previous,omitempty"`
	ReviewAfter *time.Time `json:"review_after,omitempty"`
}
type EvidenceInput struct {
	Scope          Scope     `json:"scope"`
	Root           string    `json:"root"`
	Artifact       string    `json:"artifact"`
	Digest         string    `json:"digest"`
	ResultDigest   string    `json:"result_digest"`
	ExpectedDigest string    `json:"expected_digest"`
	DerivedFrom    []string  `json:"derived_from,omitempty"`
	Method         string    `json:"method"`
	Command        string    `json:"command"`
	OracleSource   string    `json:"oracle_source"`
	Outcome        string    `json:"outcome"`
	ObservedAt     time.Time `json:"observed_at"`
	Cases          []string  `json:"cases"`
	TriggerCount   int64     `json:"trigger_count"`
	Provenance     string    `json:"provenance"`
	Limitations    string    `json:"limitations"`
	Shared         []string  `json:"shared"`
}
type JustificationInput struct {
	Claim       string   `json:"claim"`
	Evidence    []string `json:"evidence"`
	Premises    []string `json:"premises,omitempty"`
	Warrant     string   `json:"warrant"`
	Limitations string   `json:"limitations"`
}
type ChallengeInput struct {
	Claim         string   `json:"claim"`
	Kind          string   `json:"kind"`
	Justification string   `json:"justification,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Reason        string   `json:"reason"`
}
type ReviewInput struct {
	TargetKind        string `json:"target_kind"`
	Target            string `json:"target"`
	Verdict           string `json:"verdict"`
	Reason            string `json:"reason"`
	CoverageChecked   bool   `json:"coverage_checked"`
	ProvenanceChecked bool   `json:"provenance_checked"`
}
type LifecycleInput struct {
	TargetKind string `json:"target_kind"`
	Target     string `json:"target"`
	Reason     string `json:"reason"`
}
type IDInput struct {
	ID string `json:"id"`
}
type ContextInput struct {
	Scope      Scope  `json:"scope"`
	Query      string `json:"query,omitempty"`
	Objective  string `json:"objective,omitempty"`
	ByteBudget int    `json:"byte_budget"`
}
type InvestigationInput struct {
	Objective        string    `json:"objective"`
	Question         string    `json:"question"`
	Scope            Scope     `json:"scope"`
	Criteria         string    `json:"criteria"`
	BudgetTokens     int64     `json:"budget_tokens"`
	Deadline         time.Time `json:"deadline"`
	RequireSupported bool      `json:"require_supported"`
	RequireNoPending bool      `json:"require_no_pending"`
}
type AttemptInput struct {
	Investigation string `json:"investigation"`
	Model         string `json:"model"`
	Harness       string `json:"harness"`
	Arm           string `json:"arm"`
	TaskFamily    string `json:"task_family"`
}
type FinishInput struct {
	Attempt         string   `json:"attempt"`
	Receipt         string   `json:"receipt"`
	Outcome         string   `json:"outcome"`
	Disposition     string   `json:"disposition"`
	Applied         []string `json:"applied"`
	Findings        []string `json:"findings"`
	NegativeResults []string `json:"negative_results"`
	Questions       []string `json:"questions"`
	Tokens          int64    `json:"tokens"`
	ElapsedMS       int64    `json:"elapsed_ms"`
	CostMicros      int64    `json:"cost_micros"`
}
type EventInput struct {
	Consumer string `json:"consumer"`
	Event    string `json:"event,omitempty"`
}

type Principal struct{ Tenant, Domain, ID, Owner, Role string }
type Participant struct {
	Owner, Role string
	Revoked     bool
}
type claimRecord struct {
	ClaimInput
	ID, Author string
}
type evidenceRecord struct {
	EvidenceInput
	ID, Author string
}
type justificationRecord struct {
	JustificationInput
	ID, Author string
}
type challengeRecord struct {
	ChallengeInput
	ID, Author string
}
type reviewRecord struct {
	ReviewInput
	Author string
}
type Explanation struct {
	Justification string   `json:"justification"`
	Evidence      []string `json:"evidence"`
	Premises      []string `json:"premises"`
	Eligible      bool     `json:"eligible"`
	Reasons       []string `json:"reasons"`
}
type ChallengeExplanation struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Verdict  string   `json:"verdict"`
	Reason   string   `json:"reason"`
	Evidence []string `json:"evidence"`
}
type Assessment struct {
	Claim          string                 `json:"claim"`
	Policy         string                 `json:"policy"`
	Epoch          int64                  `json:"epoch"`
	Posture        string                 `json:"posture"`
	Health         string                 `json:"health"`
	Lifecycle      string                 `json:"lifecycle"`
	Pending        bool                   `json:"pending_challenge"`
	Justifications []Explanation          `json:"justifications"`
	Challenges     []ChallengeExplanation `json:"challenges"`
	Reasons        []string               `json:"reasons"`
}
type EvidenceSummary struct {
	ID          string   `json:"id"`
	Root        string   `json:"root"`
	Author      string   `json:"author"`
	Artifact    string   `json:"artifact"`
	Outcome     string   `json:"outcome"`
	Shared      []string `json:"shared"`
	Limitations string   `json:"limitations"`
}
type Card struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Previous    string            `json:"previous,omitempty"`
	Successor   string            `json:"successor,omitempty"`
	Concepts    []string          `json:"concepts"`
	Statement   string            `json:"statement"`
	Scope       Scope             `json:"scope"`
	Limitations string            `json:"limitations"`
	Assumptions []string          `json:"assumptions"`
	Assessment  Assessment        `json:"assessment"`
	Evidence    []EvidenceSummary `json:"evidence"`
}
type Bundle struct {
	Receipt        string            `json:"receipt"`
	Epoch          int64             `json:"epoch"`
	Trust          string            `json:"trust"`
	Policy         string            `json:"policy"`
	Cards          []Card            `json:"cards"`
	Investigations []json.RawMessage `json:"investigations"`
	PriorOutcomes  []json.RawMessage `json:"prior_outcomes"`
	Truncated      bool              `json:"truncated"`
	Bytes          int               `json:"bytes"`
}

func encode(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
func required(v ...string) error {
	for _, s := range v {
		if strings.TrimSpace(s) == "" {
			return problem("invalid", "required field is empty")
		}
	}
	return nil
}
func asError(err error) error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return fmt.Errorf("storage operation: %w", err)
}
