package platform

import (
	"context"
	"slices"
	"sort"
)

func (g *graph) familyOf(id string) string {
	for i := 0; i < len(g.claims); i++ {
		c, ok := g.claims[id]
		if !ok || c.Previous == "" {
			return id
		}
		id = c.Previous
	}
	return id
}
func (h *hierarchy) decorate(card *Card, g *graph) {
	for _, id := range sortedKeys(h.Associations) {
		a := h.Associations[id]
		if a.Verdict != "accepted" {
			continue
		}
		if a.Kind == "claim_concept" && a.Claim == card.ID {
			card.Classifications = append(card.Classifications, a)
		}
		if a.Kind == "placement" && a.Claim == card.ID {
			copy := a
			card.Home = &copy
		}
	}
}

type contextCandidate struct {
	card     Card
	priority int
	lexical  float64
}

func (t *transaction) selectCards(ctx context.Context, v ContextInput, g *graph, h *hierarchy, diagnostics *RetrievalDiagnostics) ([]Card, *ResolvedContext, error) {
	mode := v.Discovery
	if mode == "" {
		mode = "legacy"
		if v.Working != nil || v.OrientationOnly {
			mode = "hierarchical"
		} else if v.Query != "" || v.Objective != "" || len(v.Claims) > 0 {
			mode = "flat"
		}
	}
	if err := choice("ContextInput.discovery", mode); err != nil {
		return nil, nil, err
	}
	diagnostics.Discovery = mode
	if len(v.Claims) > 32 {
		return nil, nil, problem("invalid", "too many explicit claim references")
	}
	if err := unique(v.Claims); err != nil {
		return nil, nil, err
	}
	for _, id := range v.Claims {
		if _, ok := g.claims[id]; !ok {
			return nil, nil, problem("not_found", "claim not found")
		}
	}
	var resolved *ResolvedContext
	if v.Working != nil {
		var scope *Scope
		if !v.OrientationOnly {
			scope = &v.Scope
		}
		r, err := h.resolve(*v.Working, scope, t.metadataEpoch)
		if err != nil {
			return nil, nil, err
		}
		resolved = &r
	}
	query := v.Query
	if query == "" {
		query = v.Objective
	}
	if v.OrientationOnly && query == "" && resolved == nil && len(v.Claims) == 0 {
		return nil, nil, problem("invalid", "orientation requires a query or explicit context")
	}
	rows, err := t.Query(ctx, `WITH q AS (
 SELECT websearch_to_tsquery('english',$3) exact,
 (SELECT string_agg(quote_literal(term),' | ')::tsquery FROM unnest(tsvector_to_array(to_tsvector('english',$3))) term) overlap
) SELECT id,ts_rank_cd(search,q.exact,2),coalesce(ts_rank_cd(search,q.overlap,2),0)
 FROM claims,q WHERE tenant=$1 AND domain=$2`, t.p.Tenant, t.p.Domain, query)
	if err != nil {
		return nil, nil, err
	}
	ranks := map[string]float64{}
	exactMatches := map[string]bool{}
	for rows.Next() {
		var id string
		var exact, overlap float64
		if err = rows.Scan(&id, &exact, &overlap); err != nil {
			rows.Close()
			return nil, nil, err
		}
		exactMatches[id] = exact > 0
		ranks[id] = exact
		if mode != "legacy" {
			ranks[id] += overlap
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	candidates := []contextCandidate{}
	for _, id := range sortedKeys(g.claims) {
		c := g.claims[id]
		local := c.local(v.Scope, v.OrientationOnly)
		matched := local.State == "matched"
		lexical := ranks[id] > 0
		explicit := slices.Contains(v.Claims, id)
		a, err := g.assess(id)
		if err != nil {
			return nil, nil, err
		}
		card := g.card(id, a)
		card.LocalApplicability = local
		card.MetadataEpoch = t.metadataEpoch
		h.decorate(&card, g)
		reasons := map[string]bool{}
		directPlacement, directConcept, broadConcept, related, ancestor := false, false, false, false, false
		if resolved != nil && mode == "hierarchical" {
			if card.Home != nil {
				directPlacement = slices.Contains(resolved.DirectEntities, card.Home.Target)
				ancestor = slices.Contains(resolved.AncestorEntities, card.Home.Target)
			}
			for _, association := range card.Classifications {
				if slices.Contains(resolved.DirectConcepts, association.Target) {
					directConcept = true
					reasons["classified under requested concept "+association.Target] = true
				}
				if slices.Contains(resolved.BroaderConcepts, association.Target) {
					broadConcept = true
				}
				if slices.Contains(resolved.RelatedConcepts, association.Target) {
					related = true
				}
			}
		}
		relevant := false
		switch mode {
		case "legacy":
			relevant = matched
		case "flat":
			relevant = explicit || lexical || (query == "" && matched)
		case "hierarchical":
			relevant = explicit || lexical || directPlacement || directConcept || (resolved == nil && query == "" && matched)
		}
		if !relevant {
			if mode == "legacy" && !matched {
				diagnostics.ScopeExcluded++
			} else {
				diagnostics.RelevanceExcluded++
			}
			continue
		}
		if explicit {
			reasons["explicit claim reference"] = true
		}
		if lexical {
			if exactMatches[id] {
				reasons["task text match"] = true
			} else {
				reasons["partial task text match (not all query terms)"] = true
			}
		}
		if matched {
			reasons["explicit finite applicability match"] = true
		}
		if directPlacement {
			reasons["maintained at a requested organizational entity"] = true
		}
		if ancestor && (lexical || directConcept || explicit) {
			reasons["relevant ancestor placement"] = true
		}
		if broadConcept && lexical {
			reasons["broader concept with task text corroboration"] = true
		}
		if related && lexical {
			reasons["related concept with task text corroboration"] = true
		}
		card.RetrievalReasons = sortedKeys(reasons)
		priority := 5
		if directConcept || directPlacement {
			priority = 4
		}
		if matched {
			priority = 2
		}
		if (a.Pending || a.Posture == "contested" || a.Health != "current") && (lexical || directConcept || explicit) {
			priority = 1
		}
		if explicit {
			priority = 0
		}
		candidates = append(candidates, contextCandidate{card, priority, ranks[id]})
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.priority != b.priority {
			return a.priority < b.priority
		}
		if a.lexical != b.lexical {
			return a.lexical > b.lexical
		}
		return a.card.ID < b.card.ID
	})
	result := make([]Card, 0, len(candidates))
	for _, c := range candidates {
		result = append(result, c.card)
	}
	diagnostics.Candidates = len(result)
	return result, resolved, nil
}
