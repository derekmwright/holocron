package platform

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
)

func (t *transaction) curator(ctx context.Context) error {
	var allowed bool
	if err := t.QueryRow(ctx, `SELECT can_curate($1)`, t.p.ID).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return problem("forbidden", "explicit domain curation grant required")
	}
	return nil
}
func (t *transaction) hierarchy(ctx context.Context) (*hierarchy, error) {
	h := &hierarchy{Nodes: map[string]KnowledgeNode{}, Edges: map[string]KnowledgeEdge{}, Associations: map[string]KnowledgeAssociation{}}
	count := 0
	for _, table := range []string{"knowledge_nodes", "knowledge_edges", "knowledge_associations"} {
		query := "SELECT id,author,body FROM " + table + " WHERE tenant=$1 AND domain=$2"
		if table == "knowledge_edges" {
			query += ` AND NOT EXISTS(SELECT FROM edge_retractions r WHERE (r.tenant,r.domain,r.edge)=(knowledge_edges.tenant,knowledge_edges.domain,knowledge_edges.id))`
		}
		rows, err := t.Query(ctx, query+" LIMIT $3", t.p.Tenant, t.p.Domain, t.service.MaxHierarchyRecords+1)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			count++
			if count > t.service.MaxHierarchyRecords {
				rows.Close()
				return nil, problem("limit", "hierarchy traversal bound exceeded")
			}
			var id string
			var author string
			var raw []byte
			if err = rows.Scan(&id, &author, &raw); err != nil {
				rows.Close()
				return nil, err
			}
			switch table {
			case "knowledge_nodes":
				v, e := unmarshal[NodeInput](raw)
				err = e
				h.Nodes[id] = KnowledgeNode{NodeInput: v, ID: id, Author: author, Version: 1}
			case "knowledge_edges":
				v, e := unmarshal[EdgeInput](raw)
				err = e
				h.Edges[id] = KnowledgeEdge{EdgeInput: v, ID: id, Author: author}
			case "knowledge_associations":
				v, e := unmarshal[AssociationInput](raw)
				err = e
				h.Associations[id] = KnowledgeAssociation{AssociationInput: v, ID: id, Author: author, Verdict: "proposed"}
			}
			if err != nil {
				rows.Close()
				return nil, err
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	rows, err := t.Query(ctx, `SELECT DISTINCT ON(node) node,version,body FROM node_revisions WHERE tenant=$1 AND domain=$2 ORDER BY node,version DESC`, t.p.Tenant, t.p.Domain)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		var version int
		var raw []byte
		if err = rows.Scan(&id, &version, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		v, e := unmarshal[NodeRevisionInput](raw)
		if e != nil {
			rows.Close()
			return nil, e
		}
		n := h.Nodes[id]
		n.Name = v.Name
		n.Description = v.Description
		n.Repository = v.Repository
		n.Version = version
		h.Nodes[id] = n
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = t.Query(ctx, `SELECT DISTINCT ON(association) association,body,author FROM association_reviews WHERE tenant=$1 AND domain=$2 ORDER BY association,seq DESC`, t.p.Tenant, t.p.Domain)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		var raw []byte
		var author string
		if err = rows.Scan(&id, &raw, &author); err != nil {
			rows.Close()
			return nil, err
		}
		v, e := unmarshal[AssociationReviewInput](raw)
		if e != nil {
			rows.Close()
			return nil, e
		}
		a, ok := h.Associations[id]
		if ok {
			a.Verdict = v.Verdict
			a.ReviewedBy = author
			a.ReviewReason = v.Reason
			h.Associations[id] = a
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	return h, h.validate()
}
func (h *hierarchy) validate() error {
	for _, relation := range []string{"contains", "broader"} {
		next := map[string][]string{}
		for _, edge := range h.Edges {
			if edge.Relation == relation {
				next[edge.From] = append(next[edge.From], edge.To)
			}
		}
		state := map[string]int{}
		var visit func(string) error
		visit = func(id string) error {
			if state[id] == 1 {
				return problem("invalid", "cycle in "+relation+" relationship")
			}
			if state[id] == 2 {
				return nil
			}
			state[id] = 1
			for _, child := range next[id] {
				if err := visit(child); err != nil {
					return err
				}
			}
			state[id] = 2
			return nil
		}
		for id := range h.Nodes {
			if err := visit(id); err != nil {
				return err
			}
		}
	}
	return nil
}
func (h *hierarchy) node(id, category string) error {
	n, ok := h.Nodes[id]
	if !ok {
		return problem("not_found", "node not found")
	}
	if n.Category != category {
		return problem("invalid", "node category mismatch")
	}
	return nil
}
func (h *hierarchy) ancestors(relation string, seeds []string) []string {
	reverse := map[string][]string{}
	for _, e := range h.Edges {
		if e.Relation == relation {
			reverse[e.To] = append(reverse[e.To], e.From)
		}
	}
	seen := map[string]bool{}
	queue := slices.Clone(seeds)
	for _, id := range seeds {
		seen[id] = true
	}
	result := map[string]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, parent := range reverse[id] {
			if !seen[parent] {
				seen[parent] = true
				result[parent] = true
				queue = append(queue, parent)
			}
		}
	}
	return sortedKeys(result)
}
func (h *hierarchy) within(child, parent string) bool {
	return child == parent || slices.Contains(h.ancestors("contains", []string{child}), parent)
}
func (h *hierarchy) resolve(w WorkingContext, s *Scope, epoch int64) (ResolvedContext, error) {
	r := ResolvedContext{DirectEntities: []string{}, AncestorEntities: []string{}, DirectConcepts: []string{}, BroaderConcepts: []string{}, RelatedConcepts: []string{}, MetadataEpoch: epoch}
	if len(w.Components) > 32 || len(w.Concepts) > 32 {
		return r, problem("invalid", "too many working context identifiers")
	}
	if err := unique(w.Components); err != nil {
		return r, err
	}
	if err := unique(w.Concepts); err != nil {
		return r, err
	}
	direct := map[string]bool{}
	for _, id := range append([]string{w.Organization, w.Project, w.Repository}, w.Components...) {
		if id == "" {
			continue
		}
		if err := h.node(id, "organization"); err != nil {
			return r, err
		}
		direct[id] = true
	}
	if w.Project != "" && w.Organization != "" && !h.within(w.Project, w.Organization) {
		return r, problem("invalid", "project is not contained by the supplied organization")
	}
	if w.Repository != "" {
		if w.Project != "" && !h.within(w.Repository, w.Project) {
			return r, problem("invalid", "repository is not contained by the supplied project")
		}
		if w.Organization != "" && !h.within(w.Repository, w.Organization) {
			return r, problem("invalid", "repository is not contained by the supplied organization")
		}
		if s != nil && (h.Nodes[w.Repository].Repository == "" || h.Nodes[w.Repository].Repository != s.Repository) {
			return r, problem("invalid", "repository identity does not match execution scope")
		}
	}
	for _, id := range w.Components {
		if w.Repository == "" || !h.within(id, w.Repository) {
			return r, problem("invalid", "components require their containing repository")
		}
	}
	leaves := slices.Clone(w.Components)
	if len(leaves) == 0 {
		for _, id := range []string{w.Repository, w.Project, w.Organization} {
			if id != "" {
				leaves = []string{id}
				break
			}
		}
	}
	r.DirectEntities = leaves
	slices.Sort(r.DirectEntities)
	r.AncestorEntities = h.ancestors("contains", r.DirectEntities)
	concepts := map[string]bool{}
	for _, id := range w.Concepts {
		if err := h.node(id, "concept"); err != nil {
			return r, err
		}
		concepts[id] = true
	}
	classificationSources := map[string]bool{}
	for _, id := range leaves {
		classificationSources[id] = true
	}
	if w.Repository != "" {
		classificationSources[w.Repository] = true
	}
	for _, a := range h.Associations {
		if a.Verdict == "accepted" && a.Kind == "entity_concept" && classificationSources[a.Entity] {
			concepts[a.Target] = true
		}
	}
	r.DirectConcepts = sortedKeys(concepts)
	r.BroaderConcepts = h.ancestors("broader", r.DirectConcepts)
	related := map[string]bool{}
	for _, e := range h.Edges {
		if e.Relation == "related" {
			if concepts[e.From] && !concepts[e.To] {
				related[e.To] = true
			}
			if concepts[e.To] && !concepts[e.From] {
				related[e.From] = true
			}
		}
	}
	r.RelatedConcepts = sortedKeys(related)
	return r, nil
}
func validNode(v NodeInput) error {
	if err := required(v.Category, v.Kind, v.Name); err != nil {
		return err
	}
	if !slices.Contains([]string{"organization", "concept"}, v.Category) || len(v.Kind) > 64 || len(v.Name) > 256 || len(v.Description) > 8192 {
		return problem("invalid", "invalid node category or size")
	}
	if v.Category == "concept" && v.Repository != "" {
		return problem("invalid", "repository identity belongs to an organizational resource")
	}
	return nil
}
func (t *transaction) validateAssociation(ctx context.Context, h *hierarchy, v *AssociationInput) error {
	if err := required(v.Kind, v.Target, v.Reason); err != nil {
		return err
	}
	if !slices.Contains([]string{"placement", "claim_concept", "entity_concept"}, v.Kind) {
		return problem("invalid", "unknown association kind")
	}
	if v.Kind == "entity_concept" {
		if v.Claim != "" || v.Steward != "" {
			return problem("invalid", "entity classification cannot name a claim or steward")
		}
		if err := h.node(v.Entity, "organization"); err != nil {
			return err
		}
		return h.node(v.Target, "concept")
	}
	if v.Entity != "" {
		return problem("invalid", "claim association cannot name an entity source")
	}
	if _, _, err := t.get(ctx, "claims", v.Claim); err != nil {
		return err
	}
	if v.Kind == "claim_concept" {
		if v.Steward != "" {
			return problem("invalid", "classification confers no stewardship")
		}
		return h.node(v.Target, "concept")
	}
	if err := h.node(v.Target, "organization"); err != nil {
		return err
	}
	var exists bool
	if err := t.QueryRow(ctx, `SELECT EXISTS(SELECT FROM participants() WHERE id=$1 AND NOT revoked)`, v.Steward).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return problem("not_found", "active steward not found")
	}
	return nil
}
func (t *transaction) reviewAssociation(ctx context.Context, h *hierarchy, v AssociationReviewInput) error {
	if err := required(v.ID, v.Reason); err != nil {
		return err
	}
	if v.Verdict != "accepted" && v.Verdict != "rejected" {
		return problem("invalid", "association disposition must be accepted or rejected")
	}
	a, ok := h.Associations[v.ID]
	if !ok {
		return problem("not_found", "association not found")
	}
	if v.Verdict == "accepted" {
		candidate := a.AssociationInput
		if err := t.validateAssociation(ctx, h, &candidate); err != nil {
			return err
		}
		for id, other := range h.Associations {
			if id == v.ID || other.Verdict != "accepted" || other.Kind != a.Kind {
				continue
			}
			if a.Kind == "placement" && other.Claim == a.Claim {
				return problem("conflict", "claim revision already has a home; use placement.transfer")
			}
			if other.Claim == a.Claim && other.Entity == a.Entity && other.Target == a.Target {
				return problem("conflict", "association already accepted")
			}
		}
	}
	return t.record(ctx, "association_reviews", NewID(), v, ",association", v.ID)
}

func (t *transaction) hierarchyCommand(ctx context.Context, op string, b json.RawMessage, id string) (json.RawMessage, bool, error) {
	result := encode(map[string]string{"id": id})
	isMutation := op != "hierarchy.get" && op != "context.resolve"
	if isMutation && op != "association.propose" {
		if err := t.curator(ctx); err != nil {
			return nil, false, err
		}
	}
	h, err := t.hierarchy(ctx)
	if err != nil {
		return nil, false, err
	}
	switch op {
	case "node.create":
		v, err := decode[NodeInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = validNode(v); err != nil {
			return nil, false, err
		}
		if _, err = t.Exec(ctx, `INSERT INTO node_kinds VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, t.p.Tenant, t.p.Domain, v.Category, v.Kind); err != nil {
			return nil, false, err
		}
		err = t.record(ctx, "knowledge_nodes", id, v, ",category,kind", v.Category, v.Kind)
		if err != nil {
			return nil, false, err
		}
	case "node.revise":
		v, err := decode[NodeRevisionInput](b)
		if err != nil {
			return nil, false, err
		}
		n, ok := h.Nodes[v.ID]
		if !ok {
			return nil, false, problem("not_found", "node not found")
		}
		if n.Version != v.ExpectedVersion {
			return nil, false, problem("conflict", "node version changed")
		}
		updated := n.NodeInput
		updated.Name = v.Name
		updated.Description = v.Description
		updated.Repository = v.Repository
		if err = validNode(updated); err != nil {
			return nil, false, err
		}
		if err = required(v.Reason); err != nil {
			return nil, false, err
		}
		if err = t.record(ctx, "node_revisions", id, v, ",node,version", v.ID, v.ExpectedVersion+1); err != nil {
			return nil, false, err
		}
		result = encode(map[string]any{"id": v.ID, "version": v.ExpectedVersion + 1, "revision": id})
	case "edge.add":
		v, err := decode[EdgeInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.Relation, v.From, v.To, v.Reason); err != nil {
			return nil, false, err
		}
		category := "organization"
		if v.Relation == "broader" || v.Relation == "related" {
			category = "concept"
		}
		if !slices.Contains([]string{"contains", "broader", "related", "affiliated"}, v.Relation) || v.From == v.To {
			return nil, false, problem("invalid", "invalid edge relation or self-edge")
		}
		if err = h.node(v.From, category); err != nil {
			return nil, false, err
		}
		if err = h.node(v.To, category); err != nil {
			return nil, false, err
		}
		if v.Relation == "related" || v.Relation == "affiliated" {
			if strings.Compare(v.From, v.To) > 0 {
				v.From, v.To = v.To, v.From
			}
		}
		for _, e := range h.Edges {
			if e.Relation == v.Relation && e.From == v.From && e.To == v.To {
				return nil, false, problem("conflict", "edge already exists")
			}
		}
		if v.Replace != "" {
			old, ok := h.Edges[v.Replace]
			if !ok {
				return nil, false, problem("not_found", "active edge not found")
			}
			if old.Relation != v.Relation || old.To != v.To {
				return nil, false, problem("invalid", "replacement must preserve relation and child")
			}
			if err = t.record(ctx, "edge_retractions", NewID(), RetractionInput{v.Replace, v.Reason}, ",edge", v.Replace); err != nil {
				return nil, false, err
			}
		}
		if err = t.record(ctx, "knowledge_edges", id, v, ",relation,source,target", v.Relation, v.From, v.To); err != nil {
			return nil, false, err
		}
	case "edge.retract":
		v, err := decode[RetractionInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.ID, v.Reason); err != nil {
			return nil, false, err
		}
		if _, ok := h.Edges[v.ID]; !ok {
			return nil, false, problem("not_found", "active edge not found")
		}
		if err = t.record(ctx, "edge_retractions", id, v, ",edge", v.ID); err != nil {
			return nil, false, err
		}
	case "association.propose":
		v, err := decode[AssociationInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = t.validateAssociation(ctx, h, &v); err != nil {
			return nil, false, err
		}
		var claim, entity, steward any
		if v.Claim != "" {
			claim = v.Claim
		}
		if v.Entity != "" {
			entity = v.Entity
		}
		if v.Steward != "" {
			steward = v.Steward
		}
		if err = t.record(ctx, "knowledge_associations", id, v, ",kind,claim,entity,target,steward", v.Kind, claim, entity, v.Target, steward); err != nil {
			return nil, false, err
		}
	case "association.review":
		v, err := decode[AssociationReviewInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = t.reviewAssociation(ctx, h, v); err != nil {
			return nil, false, err
		}
		result = encode(map[string]string{"id": v.ID, "verdict": v.Verdict})
	case "placement.transfer":
		v, err := decode[PlacementTransferInput](b)
		if err != nil {
			return nil, false, err
		}
		old, ok := h.Associations[v.Previous]
		if !ok || old.Kind != "placement" || old.Verdict != "accepted" {
			return nil, false, problem("conflict", "accepted previous placement required")
		}
		next := AssociationInput{Kind: "placement", Claim: old.Claim, Target: v.Target, Steward: v.Steward, Reason: v.Reason}
		if err = t.validateAssociation(ctx, h, &next); err != nil {
			return nil, false, err
		}
		if err = t.record(ctx, "knowledge_associations", id, next, ",kind,claim,target,steward", next.Kind, next.Claim, next.Target, next.Steward); err != nil {
			return nil, false, err
		}
		if err = t.reviewAssociation(ctx, h, AssociationReviewInput{ID: old.ID, Verdict: "rejected", Reason: v.Reason}); err != nil {
			return nil, false, err
		}
		if err = t.record(ctx, "association_reviews", NewID(), AssociationReviewInput{ID: id, Verdict: "accepted", Reason: v.Reason}, ",association", id); err != nil {
			return nil, false, err
		}
	case "context.resolve":
		v, err := decode[ResolveInput](b)
		if err != nil {
			return nil, false, err
		}
		if v.Scope != nil {
			if err = v.Scope.validate(); err != nil {
				return nil, false, err
			}
		}
		resolved, err := h.resolve(v.Working, v.Scope, t.metadataEpoch)
		return encode(resolved), false, err
	case "hierarchy.get":
		v, err := decode[HierarchyLookup](b)
		if err != nil {
			return nil, false, err
		}
		limit := v.Limit
		if limit == 0 {
			limit = 100
		}
		if limit < 1 || limit > 200 {
			return nil, false, problem("invalid", "limit must be 1–200")
		}
		nodes := []KnowledgeNode{}
		edges := []KnowledgeEdge{}
		associations := []KnowledgeAssociation{}
		history := []json.RawMessage{}
		var initial json.RawMessage
		edgeHistory := []json.RawMessage{}
		truncated := false
		if v.ID != "" {
			n, ok := h.Nodes[v.ID]
			if !ok {
				return nil, false, problem("not_found", "node not found")
			}
			nodes = append(nodes, n)
			initial, _, err = t.get(ctx, "knowledge_nodes", v.ID)
			if err != nil {
				return nil, false, err
			}
			rows, err := t.Query(ctx, `SELECT body FROM node_revisions WHERE tenant=$1 AND domain=$2 AND node=$3 ORDER BY version DESC LIMIT $4`, t.p.Tenant, t.p.Domain, v.ID, limit+1)
			if err != nil {
				return nil, false, err
			}
			for rows.Next() {
				var raw json.RawMessage
				if err = rows.Scan(&raw); err != nil {
					rows.Close()
					return nil, false, err
				}
				if len(history) == limit {
					truncated = true
				} else {
					history = append(history, raw)
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, false, err
			}
			rows, err = t.Query(ctx, `SELECT jsonb_build_object('id',e.id,'author',e.author,'body',e.body,'retracted',EXISTS(SELECT FROM edge_retractions r WHERE (r.tenant,r.domain,r.edge)=(e.tenant,e.domain,e.id))) FROM knowledge_edges e WHERE e.tenant=$1 AND e.domain=$2 AND (e.source=$3 OR e.target=$3) ORDER BY e.created_at,e.id LIMIT $4`, t.p.Tenant, t.p.Domain, v.ID, limit+1)
			if err != nil {
				return nil, false, err
			}
			for rows.Next() {
				var raw json.RawMessage
				if err = rows.Scan(&raw); err != nil {
					rows.Close()
					return nil, false, err
				}
				if len(edgeHistory) == limit {
					truncated = true
				} else {
					edgeHistory = append(edgeHistory, raw)
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, false, err
			}

		} else {
			for _, id := range sortedKeys(h.Nodes) {
				if len(nodes) >= limit {
					truncated = true
					break
				}
				nodes = append(nodes, h.Nodes[id])
			}
		}
		visibleNodes := map[string]bool{}
		for _, n := range nodes {
			visibleNodes[n.ID] = true
		}
		for _, id := range sortedKeys(h.Edges) {
			e := h.Edges[id]
			if (v.ID != "" && (e.From == v.ID || e.To == v.ID)) || (v.ID == "" && visibleNodes[e.From] && visibleNodes[e.To]) {
				if len(edges) >= limit {
					truncated = true
					continue
				}
				edges = append(edges, e)
			}
		}
		for _, id := range sortedKeys(h.Associations) {
			a := h.Associations[id]
			if v.ID == "" || a.Target == v.ID || a.Entity == v.ID {
				if len(associations) >= limit {
					truncated = true
					continue
				}
				associations = append(associations, a)
			}
		}
		return encode(map[string]any{"nodes": nodes, "edges": edges, "associations": associations, "history": history, "initial": initial, "edge_history": edgeHistory, "truncated": truncated, "metadata_epoch": t.metadataEpoch}), false, nil
	default:
		return nil, false, problem("invalid", "unknown hierarchy operation")
	}
	t.metadataChanged = true
	return result, false, nil
}
