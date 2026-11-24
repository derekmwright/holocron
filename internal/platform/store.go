package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

//go:embed schema.sql
var Schema string

type Service struct {
	pool           *pgxpool.Pool
	MaxClaims      int
	MaxRecords     int
	latency        metric.Float64Histogram
	calls          metric.Int64Counter
	retrievalBytes metric.Int64Histogram
	propagation    metric.Int64Histogram
}

func Open(ctx context.Context, url string) (*Service, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = "holocron,public"
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "15000"
	cfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "20000"
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	var unsafe bool
	err = p.QueryRow(ctx, `SELECT r.rolsuper OR r.rolbypassrls OR EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON c.relnamespace=n.oid WHERE n.nspname='holocron' AND pg_has_role(r.oid,c.relowner,'MEMBER')) FROM pg_roles r WHERE r.rolname=current_user`).Scan(&unsafe)
	if err != nil || unsafe {
		p.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("runtime database role must be non-owner, non-superuser, without BYPASSRLS")
	}
	meter := otel.Meter("holocron")
	latency, _ := meter.Float64Histogram("holocron.operation.duration", metric.WithUnit("s"))
	calls, _ := meter.Int64Counter("holocron.operations")
	retrievalBytes, _ := meter.Int64Histogram("holocron.context.bytes", metric.WithUnit("By"))
	propagation, _ := meter.Int64Histogram("holocron.assessment.claims")
	return &Service{pool: p, MaxClaims: 1000, MaxRecords: 10000, latency: latency, calls: calls, retrievalBytes: retrievalBytes, propagation: propagation}, nil
}
func (s *Service) Close() { s.pool.Close() }
func (s *Service) Authenticate(ctx context.Context, token string) error {
	tx, err := s.begin(ctx, token)
	if err != nil {
		return err
	}
	return tx.Rollback(ctx)
}
func Digest(token string) string { h := sha256.Sum256([]byte(token)); return hex.EncodeToString(h[:]) }
func decode[T any](b []byte) (T, error) {
	var v T
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return v, problem("invalid", "invalid arguments: "+err.Error())
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return v, problem("invalid", "one JSON object required")
	}
	return v, nil
}

type transaction struct {
	pgx.Tx
	p       Principal
	epoch   int64
	service *Service
}

func (s *Service) begin(ctx context.Context, token string) (*transaction, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	fail := func(e error) (*transaction, error) { _ = tx.Rollback(ctx); return nil, e }
	var p Principal
	if token == "" {
		return fail(problem("unauthorized", "authentication required"))
	}
	err = tx.QueryRow(ctx, `SELECT tenant,domain,id,owner,role FROM holocron.authenticate($1)`, Digest(token)).Scan(&p.Tenant, &p.Domain, &p.ID, &p.Owner, &p.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return fail(problem("unauthorized", "invalid or expired credential"))
	}
	if err != nil {
		return fail(err)
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('holocron.tenant',$1,true),set_config('holocron.domain',$2,true)`, p.Tenant, p.Domain); err != nil {
		return fail(err)
	}
	var epoch int64
	if err = tx.QueryRow(ctx, `SELECT epoch FROM domains WHERE tenant=$1 AND id=$2 FOR UPDATE`, p.Tenant, p.Domain).Scan(&epoch); err != nil {
		return fail(err)
	}
	// Credential revocation and domain writes use the same domain lock in admin operations.
	var id string
	if err = tx.QueryRow(ctx, `SELECT id FROM holocron.authenticate($1)`, Digest(token)).Scan(&id); err != nil {
		return fail(problem("unauthorized", "credential revoked"))
	}
	return &transaction{Tx: tx, p: p, epoch: epoch, service: s}, nil
}
func (s *Service) Execute(ctx context.Context, token, op, key string, body json.RawMessage) (out json.RawMessage, err error) {
	start := time.Now()
	ctx, span := otel.Tracer("holocron").Start(ctx, "operation")
	defer span.End()
	// Do not record arbitrary operation names or payloads in telemetry.
	if !knownOperation(op) {
		return nil, problem("invalid", "unknown operation")
	}
	span.SetAttributes(attribute.String("operation", op))
	defer func() {
		status := "ok"
		if err != nil {
			status = "error"
		}
		attrs := metric.WithAttributes(attribute.String("operation", op), attribute.String("status", status))
		s.latency.Record(ctx, time.Since(start).Seconds(), attrs)
		s.calls.Add(ctx, 1, attrs)
	}()
	if len(body) > 256*1024 {
		return nil, problem("invalid", "arguments exceed 256 KiB")
	}
	if len(key) == 0 || len(key) > 128 {
		return nil, problem("invalid", "idempotency key of 1–128 bytes required")
	}
	var canonical any
	if json.Unmarshal(body, &canonical) != nil {
		return nil, problem("invalid", "invalid JSON")
	}
	if _, ok := canonical.(map[string]any); !ok {
		return nil, problem("invalid", "arguments must be a JSON object")
	}
	hash := Digest(string(encode(canonical)))
	tx, err := s.begin(ctx, token)
	if err != nil {
		return nil, asError(err)
	}
	defer tx.Rollback(context.Background())
	var oldHash string
	var old json.RawMessage
	err = tx.QueryRow(ctx, `SELECT digest,response FROM commands WHERE tenant=$1 AND domain=$2 AND author=$3 AND operation=$4 AND key=$5`, tx.p.Tenant, tx.p.Domain, tx.p.ID, op, key).Scan(&oldHash, &old)
	if err == nil {
		if oldHash != hash {
			return nil, problem("conflict", "idempotency key already used with different arguments")
		}
		return old, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	// A savepoint lets denied commands be audited without retaining partial mutations.
	initialEpoch := tx.epoch
	if _, err = tx.Exec(ctx, "SAVEPOINT command"); err != nil {
		return nil, err
	}
	out, changed, err := tx.dispatch(ctx, op, body)
	if err == nil && changed {
		tx.epoch++
		_, err = tx.Exec(ctx, `UPDATE domains SET epoch=$3 WHERE tenant=$1 AND id=$2`, tx.p.Tenant, tx.p.Domain, tx.epoch)
		if err == nil {
			err = tx.recompute(ctx)
		}
	}
	if err != nil {
		_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT command")
		tx.epoch = initialEpoch
		_, auditErr := tx.Exec(ctx, `INSERT INTO audit(tenant,domain,id,author,operation,outcome,epoch) VALUES($1,$2,$3,$4,$5,'denied',$6)`, tx.p.Tenant, tx.p.Domain, NewID(), tx.p.ID, op, tx.epoch)
		if auditErr == nil {
			_ = tx.Commit(ctx)
		}
		var pe *pgconn.PgError
		if errors.As(err, &pe) {
			return nil, problem("invalid", "invalid reference or invariant violation")
		}
		return nil, asError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO commands VALUES($1,$2,$3,$4,$5,$6,$7)`, tx.p.Tenant, tx.p.Domain, tx.p.ID, op, key, hash, out); err != nil {
		return nil, err
	}
	event := NewID()
	if _, err = tx.Exec(ctx, `INSERT INTO audit(tenant,domain,id,author,operation,outcome,epoch) VALUES($1,$2,$3,$4,$5,'accepted',$6)`, tx.p.Tenant, tx.p.Domain, event, tx.p.ID, op, tx.epoch); err != nil {
		return nil, err
	}
	if changed {
		if _, err = tx.Exec(ctx, `INSERT INTO outbox(tenant,domain,id,epoch,operation) VALUES($1,$2,$3,$4,$5)`, tx.p.Tenant, tx.p.Domain, event, tx.epoch, op); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (t *transaction) reviewer() error {
	if t.p.Role != "reviewer" && t.p.Role != "admin" {
		return problem("forbidden", "reviewer authority required")
	}
	return nil
}
func (t *transaction) record(ctx context.Context, table, id string, body any, extraCols string, extraArgs ...any) error {
	// All identifiers and column expressions are compile-time constants in callers.
	args := []any{t.p.Tenant, t.p.Domain, id, t.p.ID, encode(body)}
	args = append(args, extraArgs...)
	placeholders := "$1,$2,$3,$4,$5"
	for i := range extraArgs {
		placeholders += fmt.Sprintf(",$%d", i+6)
	}
	_, err := t.Exec(ctx, "INSERT INTO "+table+"(tenant,domain,id,author,body"+extraCols+") VALUES("+placeholders+")", args...)
	return err
}
func (t *transaction) link(ctx context.Context, table, left, right, a, b string) error {
	_, err := t.Exec(ctx, "INSERT INTO "+table+"(tenant,domain,"+left+","+right+") VALUES($1,$2,$3,$4)", t.p.Tenant, t.p.Domain, a, b)
	return err
}
func (t *transaction) get(ctx context.Context, table, id string) (json.RawMessage, string, error) {
	var b json.RawMessage
	var author string
	err := t.QueryRow(ctx, "SELECT body,author FROM "+table+" WHERE tenant=$1 AND domain=$2 AND id=$3", t.p.Tenant, t.p.Domain, id).Scan(&b, &author)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", problem("not_found", "record not found")
	}
	return b, author, err
}
