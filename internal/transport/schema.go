package transport

import (
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/derekmwright/holocron/internal/platform"
	"github.com/google/jsonschema-go/jsonschema"
)

// Infer object shape, then add the constraints which Go field types cannot
// express. The resulting schema is both advertised and validated by the MCP SDK.
func toolInputSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[ToolInput[T]](nil)
	if err != nil {
		panic(err)
	}
	annotateInput(s, reflect.TypeFor[ToolInput[T]]())
	s.Properties["key"].MinLength = jsonschema.Ptr(1)
	s.Properties["key"].MaxLength = jsonschema.Ptr(128)
	return s
}
func nonempty(s *jsonschema.Schema) { s.MinLength = jsonschema.Ptr(1); s.Pattern = `\S` }
func stringsIn(s *jsonschema.Schema, unique bool) {
	if s.Items != nil {
		nonempty(s.Items)
	}
	s.UniqueItems = unique
}
func arrayBounds(s *jsonschema.Schema, min, max int) {
	s.MinItems = jsonschema.Ptr(min)
	s.MaxItems = jsonschema.Ptr(max)
	if min > 0 {
		s.Type = "array"
		s.Types = nil
	}
}
func numberBounds(s *jsonschema.Schema, min, max float64) {
	s.Minimum = jsonschema.Ptr(min)
	s.Maximum = jsonschema.Ptr(max)
}
func constant(value any) *jsonschema.Schema { return &jsonschema.Schema{Const: &value} }
func props(values map[string]*jsonschema.Schema) *jsonschema.Schema {
	return &jsonschema.Schema{Properties: values}
}
func requires(names ...string) *jsonschema.Schema { return &jsonschema.Schema{Required: names} }
func when(field string, value any, then *jsonschema.Schema) *jsonschema.Schema {
	return &jsonschema.Schema{If: &jsonschema.Schema{Required: []string{field}, Properties: map[string]*jsonschema.Schema{field: constant(value)}}, Then: then}
}

func annotateInput(s *jsonschema.Schema, t reflect.Type) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeFor[time.Time]() {
		s.Description = "RFC3339 timestamp; provide the actual observation time or contract deadline."
		return
	}
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		if s.Items != nil {
			annotateInput(s.Items, t.Elem())
		}
		return
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name, options, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" {
			name = field.Name
		}
		p := s.Properties[name]
		if p == nil {
			continue
		}
		annotateInput(p, field.Type)
		values := platform.InputEnum(t.Name() + "." + name)
		if len(values) > 0 {
			for _, v := range values {
				p.Enum = append(p.Enum, v)
			}
			if strings.Contains(options, "omitempty") && field.Type.Kind() == reflect.String {
				p.Enum = append(p.Enum, "")
			}
		}
	}
	p := s.Properties
	requiredStrings := func(names ...string) {
		for _, name := range names {
			nonempty(p[name])
		}
	}
	switch t {
	case reflect.TypeFor[platform.Scope]():
		requiredStrings("repository", "commit", "environment", "configuration", "dataset", "oracle")
		for _, name := range []string{"repository", "environment", "configuration", "dataset", "oracle"} {
			p[name].Not = &jsonschema.Schema{Enum: []any{"*", "unknown"}}
			p[name].Description = "Exact observed identifier; empty, '*' and 'unknown' are not execution scope. Use an unknown applicability envelope for source-only claims."
		}
		p["commit"].Pattern = platform.GitObjectPattern
		p["commit"].Description = "Full immutable Git object ID: 40 or 64 hexadecimal characters, not a branch name."
		arrayBounds(p["cases"], platform.ScopeCasesMin, platform.ScopeCasesMax)
		stringsIn(p["cases"], true)
		p["cases"].Description = "1–100 unique non-empty named cases actually in scope. Do not invent execution cases for ADR imports; omit claim.scope and use applicability version 2, state unknown."
	case reflect.TypeFor[platform.SourceReference]():
		requiredStrings("uri", "revision", "fragment", "digest")
		p["uri"].Pattern = platform.URIWithSchemePattern
		p["uri"].Format = "uri"
		p["uri"].Description = "Absolute artifact URI with a scheme, e.g. https://host/repo/blob/REV/path or file:///absolute/path. Repository-relative paths and embedded credentials are invalid."
		p["digest"].Pattern = platform.DigestPattern
		p["digest"].Description = "Digest of the actual inspected bytes: sha256: followed by exactly 64 hexadecimal characters."
		p["revision"].Description = "Actual immutable source revision, distinct from applicability to the current execution."
	case reflect.TypeFor[platform.ApplicabilityEnvelope]():
		p["version"].Enum = []any{float64(2)}
		arrayBounds(p["contexts"], 0, 16)
		p["state"].Description = "unknown: proposed/unvalidated applicability; enumerated: explicitly listed complete contexts, each requiring its own evidence."
		nonblank := &jsonschema.Schema{Type: "string", MinLength: jsonschema.Ptr(1), Pattern: `\S`}
		s.AllOf = []*jsonschema.Schema{
			when("state", "unknown", &jsonschema.Schema{Required: []string{"description"}, Properties: map[string]*jsonschema.Schema{"description": nonblank, "contexts": {MaxItems: jsonschema.Ptr(0)}}}),
			when("state", "enumerated", &jsonschema.Schema{Required: []string{"contexts"}, Properties: map[string]*jsonschema.Schema{"contexts": {Type: "array", MinItems: jsonschema.Ptr(1), MaxItems: jsonschema.Ptr(16), UniqueItems: true}}}),
		}
	case reflect.TypeFor[platform.ClaimInput]():
		requiredStrings("statement", "limitations")
		p["statement"].MaxLength = jsonschema.Ptr(8192)
		p["scope"].Description = "Legacy exact execution scope. Mutually exclusive with a non-null applicability envelope. Omit entirely for source-only ADR imports."
		p["applicability"].Description = "Alternative to scope. For an unvalidated source-derived hypothesis/generalization use {version:2,state:unknown,description:...}; omit scope. Repository/commit provenance stays in sources."
		p["sources"].MaxItems = jsonschema.Ptr(16)
		p["assumptions"].MaxItems = jsonschema.Ptr(64)
		stringsIn(p["assumptions"], true)
		p["concepts"].MaxItems = jsonschema.Ptr(20)
		s.OneOf = []*jsonschema.Schema{
			{Required: []string{"scope"}, Properties: map[string]*jsonschema.Schema{"applicability": {Type: "null"}}},
			{Required: []string{"applicability"}, Not: requires("scope"), Properties: map[string]*jsonschema.Schema{"applicability": {Type: "object"}}},
		}
		s.AllOf = append(s.AllOf, when("kind", "finite_cases", props(map[string]*jsonschema.Schema{"applicability": props(map[string]*jsonschema.Schema{"state": constant("enumerated")})})))
	case reflect.TypeFor[platform.AttemptInput]():
		requiredStrings("investigation", "model", "harness", "task_family")
		p["arm"].Description = "A=baseline without Holocron; B=document retrieval; C=structured Holocron; demo=exploratory/non-comparative run. 'holocron' is not an accepted arm."
	case reflect.TypeFor[platform.FinishInput]():
		requiredStrings("attempt", "receipt", "disposition")
		p["outcome"].Description = "Allowed terminal attempt outcomes only. Use failed, not failure. For unfinished blocked work, use partial and describe the blocker in disposition/questions; blocked is not an outcome."
		for _, name := range []string{"tokens", "elapsed_ms", "cost_micros"} {
			p[name].Minimum = jsonschema.Ptr(float64(0))
			p[name].Description = "Non-negative integer. Keep measurement limitations explicit; do not invent usage."
		}
		for _, name := range []string{"applied", "considered", "findings"} {
			stringsIn(p[name], true)
		}
		p["applied"].Description = "Only locally matched claim IDs delivered in this receipt; qualification/support checks still apply."
		p["considered"].Description = "Delivered orientation/analogy IDs used as qualified guidance; not locally established premises."
	case reflect.TypeFor[platform.InvestigationInput]():
		requiredStrings("objective", "question", "criteria")
		p["budget_tokens"].Minimum = jsonschema.Ptr(float64(1))
		p["deadline"].Description = "Future RFC3339 deadline; time-dependent validation is performed by the server."
	case reflect.TypeFor[platform.EvidenceInput]():
		requiredStrings("root", "artifact", "method", "command", "limitations")
		p["artifact"].Pattern = platform.URIWithSchemePattern
		p["artifact"].Description = "URI of the actual capture, e.g. file:///absolute/path/to/probe.json. Bind those bytes with digest. Do not substitute a GitHub source URL for different scratch output. The server never fetches or executes the artifact."
		s.Required = slices.DeleteFunc(s.Required, func(name string) bool {
			return name == "result_digest" || name == "expected_digest" || name == "oracle_source"
		})
		p["observed_at"].Description = "Actual non-zero RFC3339 observation timestamp, not the deadline; at most five minutes ahead of the server clock. Do not guess timestamps."
		p["provenance"].Description = "known means the origin and actual artifact digest are recorded; it does not mean an oracle or independent reproduction exists."
		p["outcome"].Description = "Producer-reported method outcome, not platform verification. Use inconclusive when the run supplies no comparison supporting a pass/fail conclusion."
		p["trigger_count"].Minimum = jsonschema.Ptr(float64(0))
		stringsIn(p["cases"], true)
		p["derived_from"].MaxItems = jsonschema.Ptr(20)
		stringsIn(p["derived_from"], true)
		known := map[string]*jsonschema.Schema{"digest": {Pattern: platform.DigestPattern}}
		p["digest"].Description = "For known provenance: SHA-256 of the actual referenced artifact, with sha256: prefix and 64 hex characters."
		for _, name := range []string{"result_digest", "expected_digest"} {
			known[name] = &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Pattern: platform.DigestPattern}, {MaxLength: jsonschema.Ptr(0)}}}
		}
		p["result_digest"].Description = "Optional normalized observed-result digest (sha256: plus 64 hex characters). Required for comparison-based establishment, not basic capture."
		p["expected_digest"].Description = "Optional digest of a genuinely defined expected result/property. Leave absent/empty when no oracle is defined; never copy the actual digest just to make a pass. Missing comparison keeps the claim unestablished."
		p["oracle_source"].Description = "Optional reference to the actual comparison oracle/method. A metamorphic property can be an oracle; absence prevents establishment but does not make a known artifact origin unknown."
		s.AllOf = []*jsonschema.Schema{when("provenance", "known", props(known))}
	case reflect.TypeFor[platform.JustificationInput]():
		requiredStrings("claim", "warrant", "limitations")
		arrayBounds(p["evidence"], 1, 20)
		stringsIn(p["evidence"], true)
		p["premises"].MaxItems = jsonschema.Ptr(20)
		stringsIn(p["premises"], true)
		p["inference_kind"].Description = "Omit for legacy reproduction. Bounded generalizations require bounded_generalization, premises, assumptions and coverage for every context. Analogy alone cannot establish support."
	case reflect.TypeFor[platform.ChallengeInput]():
		requiredStrings("claim", "reason")
		p["evidence"].MaxItems = jsonschema.Ptr(20)
		stringsIn(p["evidence"], true)
		s.AllOf = []*jsonschema.Schema{when("kind", "methodology", &jsonschema.Schema{Required: []string{"justification"}, Properties: map[string]*jsonschema.Schema{"justification": {MinLength: jsonschema.Ptr(1)}}})}
	case reflect.TypeFor[platform.ReviewInput]():
		requiredStrings("target", "reason")
		s.AllOf = []*jsonschema.Schema{when("verdict", "accepted", props(map[string]*jsonschema.Schema{"coverage_checked": constant(true), "provenance_checked": constant(true)}))}
	case reflect.TypeFor[platform.LifecycleInput]():
		requiredStrings("target", "reason")
	case reflect.TypeFor[platform.IDInput]():
		requiredStrings("id")
	case reflect.TypeFor[platform.ContextInput]():
		numberBounds(p["byte_budget"], 512, 262144)
		numberBounds(p["orientation_percent"], 0, 100)
		p["claims"].MaxItems = jsonschema.Ptr(32)
		stringsIn(p["claims"], true)
		p["scope"].Description = "Complete target execution scope. Omit when orientation_only is true."
		p["orientation_only"].Description = "Retrieve qualified orientation without asserting an execution scope; its receipt cannot complete execution-scoped work."
		p["orientation_percent"].Description = "Explicit hard maximum percentage of the raw bundle for full orientation cards (0–100). Omit to use the server's advertised adaptive allocation."
		p["discovery"].Description = "legacy restricts candidates to matching scope; flat uses task text; hierarchical also uses curated relationships. Omitted query-only requests use flat; working/orientation requests use hierarchical; scope-only requests retain legacy."
		s.AllOf = []*jsonschema.Schema{{If: &jsonschema.Schema{Required: []string{"orientation_only"}, Properties: map[string]*jsonschema.Schema{"orientation_only": constant(true)}}, Then: &jsonschema.Schema{Not: requires("scope")}, Else: requires("scope")}}
	case reflect.TypeFor[platform.WorkingContext]():
		for _, name := range []string{"components", "concepts"} {
			p[name].MaxItems = jsonschema.Ptr(32)
			stringsIn(p[name], true)
		}
	case reflect.TypeFor[platform.NodeInput]():
		requiredStrings("kind", "name")
		p["kind"].MaxLength = jsonschema.Ptr(64)
		p["name"].MaxLength = jsonschema.Ptr(256)
		p["description"].MaxLength = jsonschema.Ptr(8192)
	case reflect.TypeFor[platform.NodeRevisionInput]():
		requiredStrings("id", "name", "reason")
		p["expected_version"].Minimum = jsonschema.Ptr(float64(1))
		p["name"].MaxLength = jsonschema.Ptr(256)
		p["description"].MaxLength = jsonschema.Ptr(8192)
	case reflect.TypeFor[platform.EdgeInput]():
		requiredStrings("from", "to", "reason")
	case reflect.TypeFor[platform.RetractionInput]():
		requiredStrings("id", "reason")
	case reflect.TypeFor[platform.AssociationInput]():
		requiredStrings("target", "reason")
		p["claim"].Description = "Required for claim_concept and placement; omit for entity_concept."
		p["entity"].Description = "Required only for entity_concept; must identify an organizational entity."
		p["steward"].Description = "Required only for placement; must identify an active principal."
		s.AllOf = []*jsonschema.Schema{when("kind", "entity_concept", requires("entity")), when("kind", "claim_concept", requires("claim")), when("kind", "placement", requires("claim", "steward"))}
	case reflect.TypeFor[platform.AssociationReviewInput]():
		requiredStrings("id", "reason")
	case reflect.TypeFor[platform.PlacementTransferInput]():
		requiredStrings("previous", "target", "steward", "reason")
	case reflect.TypeFor[platform.HierarchyLookup]():
		numberBounds(p["limit"], 0, 200)
		p["limit"].Description = "1–200, or omit/0 for the default of 100."
	case reflect.TypeFor[platform.EventInput]():
		requiredStrings("consumer")
		p["consumer"].MaxLength = jsonschema.Ptr(128)
	}
}
