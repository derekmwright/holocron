package platform

import (
	"context"
	"encoding/json"
	"slices"
	"time"
)

func (t *transaction) work(ctx context.Context, op string, b json.RawMessage, id string) (json.RawMessage, bool, error) {
	result := encode(map[string]string{"id": id})
	switch op {
	case "investigation.open":
		v, err := decode[InvestigationInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = v.Scope.validate(); err != nil {
			return nil, false, err
		}
		if err = required(v.Objective, v.Question, v.Criteria); err != nil {
			return nil, false, err
		}
		if v.BudgetTokens <= 0 || !v.Deadline.After(time.Now()) {
			return nil, false, problem("invalid", "positive budget and future deadline required")
		}
		return result, false, t.record(ctx, "investigations", id, v, "")
	case "attempt.start":
		v, err := decode[AttemptInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.Investigation, v.Model, v.Harness, v.TaskFamily); err != nil {
			return nil, false, err
		}
		if !slices.Contains([]string{"A", "B", "C", "demo"}, v.Arm) {
			return nil, false, problem("invalid", "arm must be A, B, C or demo")
		}
		raw, _, err := t.get(ctx, "investigations", v.Investigation)
		if err != nil {
			return nil, false, err
		}
		in, _ := decode[InvestigationInput](raw)
		if !in.Deadline.After(time.Now()) {
			return nil, false, problem("conflict", "investigation deadline passed")
		}
		return result, false, t.record(ctx, "attempts", id, v, ",investigation", v.Investigation)
	case "attempt.finish":
		v, err := decode[FinishInput](b)
		if err != nil {
			return nil, false, err
		}
		if err = required(v.Attempt, v.Receipt, v.Disposition); err != nil {
			return nil, false, err
		}
		if !slices.Contains([]string{"completed", "failed", "partial", "timed_out", "cancelled", "abandoned"}, v.Outcome) || v.Tokens < 0 || v.ElapsedMS < 0 || v.CostMicros < 0 {
			return nil, false, problem("invalid", "invalid outcome or cost")
		}
		if err = unique(v.Applied); err != nil {
			return nil, false, err
		}
		if err = unique(v.Findings); err != nil {
			return nil, false, err
		}
		raw, author, err := t.get(ctx, "attempts", v.Attempt)
		if err != nil {
			return nil, false, err
		}
		if author != t.p.ID {
			return nil, false, problem("forbidden", "only attempt owner may report outcome")
		}
		attempt, _ := decode[AttemptInput](raw)
		raw, _, err = t.get(ctx, "investigations", attempt.Investigation)
		if err != nil {
			return nil, false, err
		}
		in, _ := decode[InvestigationInput](raw)
		raw, author, err = t.get(ctx, "receipts", v.Receipt)
		if err != nil {
			return nil, false, err
		}
		if author != t.p.ID {
			return nil, false, problem("forbidden", "receipt belongs to another participant")
		}
		var receipt struct {
			Scope  Scope    `json:"scope"`
			Claims []string `json:"claims"`
			Bytes  int      `json:"bytes"`
		}
		if err = json.Unmarshal(raw, &receipt); err != nil {
			return nil, false, err
		}
		var epoch int64
		if err = t.QueryRow(ctx, `SELECT epoch FROM receipts WHERE tenant=$1 AND domain=$2 AND id=$3`, t.p.Tenant, t.p.Domain, v.Receipt).Scan(&epoch); err != nil {
			return nil, false, err
		}
		// Negative and interrupted attempts remain reportable after expiry/staleness.
		if v.Outcome == "completed" && (epoch != t.epoch || !in.Deadline.After(time.Now()) || v.Tokens > in.BudgetTokens) {
			return nil, false, problem("conflict", "completion requires current context and unexceeded contract budget/deadline")
		}
		if !receipt.Scope.covers(in.Scope) {
			return nil, false, problem("invalid", "context receipt does not cover the work contract")
		}
		g, err := t.graph(ctx)
		if err != nil {
			return nil, false, err
		}
		for _, claim := range v.Applied {
			if !slices.Contains(receipt.Claims, claim) {
				return nil, false, problem("invalid", "applied claim was not delivered in this receipt")
			}
			a, err := g.assess(claim)
			if err != nil {
				return nil, false, err
			}
			if v.Outcome == "completed" && ((in.RequireSupported && (a.Posture != "supported" || a.Health != "current" || a.Lifecycle != "active")) || (in.RequireNoPending && a.Pending)) {
				return nil, false, problem("conflict", "applied claim does not meet completion policy")
			}
		}
		for _, claim := range v.Findings {
			if _, ok := g.claims[claim]; !ok {
				return nil, false, problem("not_found", "finding not found")
			}
		}
		if err = t.record(ctx, "outcomes", id, v, ",attempt,receipt", v.Attempt, v.Receipt); err != nil {
			return nil, false, err
		}
		for _, claim := range v.Applied {
			if err = t.link(ctx, "outcome_claims", "outcome", "claim", id, claim); err != nil {
				return nil, false, err
			}
		}
		return result, false, nil
	case "work.list", "experiment.export":
		if _, err := decode[struct{}](b); err != nil {
			return nil, false, err
		}
		if op == "experiment.export" {
			if err := t.reviewer(); err != nil {
				return nil, false, err
			}
		}
		output := map[string]any{}
		for _, table := range []string{"investigations", "attempts", "outcomes"} {
			rows, err := t.Query(ctx, "SELECT id,author,body,created_at FROM "+table+" WHERE tenant=$1 AND domain=$2 ORDER BY created_at,id LIMIT 1000", t.p.Tenant, t.p.Domain)
			if err != nil {
				return nil, false, err
			}
			items := []any{}
			for rows.Next() {
				var id, author string
				var b json.RawMessage
				var at time.Time
				if err = rows.Scan(&id, &author, &b, &at); err != nil {
					rows.Close()
					return nil, false, err
				}
				items = append(items, map[string]any{"id": id, "author": author, "body": b, "created_at": at})
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, false, err
			}
			output[table] = items
		}
		output["policy"] = Policy
		output["measurement_warning"] = "Agent-reported outcomes and cost require external correctness scoring; not a causal experiment result. Each collection is limited to 1000 rows."
		return encode(output), false, nil
	}
	return nil, false, problem("invalid", "unknown work operation")
}
func (t *transaction) events(ctx context.Context, op string, b json.RawMessage) (json.RawMessage, bool, error) {
	if err := t.reviewer(); err != nil {
		return nil, false, err
	}
	v, err := decode[EventInput](b)
	if err != nil {
		return nil, false, err
	}
	if err = required(v.Consumer); err != nil {
		return nil, false, err
	}
	if len(v.Consumer) > 128 {
		return nil, false, problem("invalid", "consumer name too long")
	}
	if op == "events.ack" {
		if err = required(v.Event); err != nil {
			return nil, false, err
		}
		_, err = t.Exec(ctx, `INSERT INTO deliveries(tenant,domain,consumer,event) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, t.p.Tenant, t.p.Domain, v.Consumer, v.Event)
		return encode(map[string]bool{"acknowledged": err == nil}), false, err
	}
	rows, err := t.Query(ctx, `SELECT o.id,o.epoch,o.operation FROM outbox o WHERE o.tenant=$1 AND o.domain=$2 AND NOT EXISTS(SELECT FROM deliveries d WHERE (d.tenant,d.domain,d.event)=(o.tenant,o.domain,o.id) AND d.consumer=$3) ORDER BY epoch,id LIMIT 100`, t.p.Tenant, t.p.Domain, v.Consumer)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	events := []any{}
	for rows.Next() {
		var id, op string
		var epoch int64
		if err = rows.Scan(&id, &epoch, &op); err != nil {
			return nil, false, err
		}
		events = append(events, map[string]any{"id": id, "epoch": epoch, "operation": op})
	}
	return encode(map[string]any{"events": events}), false, rows.Err()
}
