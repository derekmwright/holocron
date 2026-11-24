package platform

import "slices"

func packContextCards(b *Bundle, cards []Card, v ContextInput, orientationPercent int) {
	included := map[string]bool{}
	reasons := map[string]string{}
	orientationBytes := 0
	try := func(card Card) bool {
		if included[card.ID] {
			return true
		}
		orientation := card.LocalApplicability.State != "matched"
		size := len(encode(card))
		if orientation && orientationBytes+size > v.ByteBudget*orientationPercent/100 {
			reasons[card.ID] = "orientation_limit"
			return false
		}
		b.Cards = append(b.Cards, card)
		if len(bundleBytes(b)) > v.ByteBudget {
			b.Cards = b.Cards[:len(b.Cards)-1]
			reasons[card.ID] = "byte_budget"
			return false
		}
		included[card.ID] = true
		delete(reasons, card.ID)
		if orientation {
			orientationBytes += size
		}
		return true
	}
	// Explicitly requested revisions take precedence over automatic diversification.
	for _, card := range cards {
		if slices.Contains(v.Claims, card.ID) {
			try(card)
		}
	}
	// Preserve a complete relevant orientation card before a large matched summary
	// consumes the bundle. When an explicit cap is supplied, it remains a hard cap.
	for _, card := range cards {
		if card.LocalApplicability.State != "matched" && !included[card.ID] {
			if try(card) {
				break
			}
		}
	}
	for _, card := range cards {
		if card.LocalApplicability.State == "matched" {
			try(card)
		}
	}
	// Oversized matched assertions remain discoverable without stripping their
	// qualifiers. References are not receipt-delivered claims and cannot be applied.
	addReference := func(card Card) {
		if included[card.ID] || len(b.Deferred) >= 4 {
			return
		}
		reason := reasons[card.ID]
		if reason == "" {
			reason = "byte_budget"
		}
		ref := DeferredClaim{ID: card.ID, Applicability: card.LocalApplicability.State, Posture: card.Assessment.Posture, Reason: reason, Fetch: "claim.get", RequiresDetail: true}
		b.Deferred = append(b.Deferred, ref)
		if len(bundleBytes(b)) > v.ByteBudget {
			b.Deferred = b.Deferred[:len(b.Deferred)-1]
		}
	}
	for _, card := range cards {
		if card.LocalApplicability.State == "matched" {
			addReference(card)
		}
	}
	// With no explicit cap, unused capacity is available to qualified orientation.
	// This does not change its applicability or assessment.
	for _, card := range cards {
		if card.LocalApplicability.State != "matched" {
			try(card)
		}
	}
	for _, card := range cards {
		if !included[card.ID] {
			b.Truncated = true
			if reasons[card.ID] == "orientation_limit" {
				b.Diagnostics.OrientationOmitted++
			} else {
				b.Diagnostics.BudgetOmitted++
			}
			if card.LocalApplicability.State != "matched" {
				addReference(card)
			}
		}
	}
}
