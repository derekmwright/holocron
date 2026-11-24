#!/usr/bin/env bash
# Creates an isolated disposable PostgreSQL cluster; never touches an existing database.
set -euo pipefail
cd "$(dirname "$0")/.."
pgbin="${PG_BIN:-/usr/lib/postgresql/17/bin}"
test_root=$(mktemp -d /tmp/holocron-test.XXXXXX)
cleanup() {
  "$pgbin/pg_ctl" -D "$test_root/data" -m immediate stop >/dev/null 2>&1 || true
  rm -rf "$test_root"
}
trap cleanup EXIT
"$pgbin/initdb" -D "$test_root/data" -A trust --no-locale -E UTF8 --no-sync >"$test_root/init.log"
# Unix socket only: private directory, no listener exposed to the network.
"$pgbin/pg_ctl" -D "$test_root/data" -l "$test_root/postgres.log" -o "-k $test_root -h '' -F" start >/dev/null
export HOLOCRON_ADMIN_DATABASE_URL="postgresql:///postgres?host=$test_root&sslmode=disable"
export HOLOCRON_TEST_DATABASE_URL="postgresql://holocron_app@/postgres?host=$test_root&sslmode=disable"
go run ./cmd/holocron migrate
for actor in worker-a worker-b worker-c worker-d validator reviewer curator outsider; do
  tenant=demo; domain=wadjet; role=worker
  if [[ "$actor" == validator || "$actor" == reviewer ]]; then role=$actor; fi
  if [[ "$actor" == outsider ]]; then tenant=other; domain=private; fi
  curator=false
  if [[ "$actor" == curator ]]; then curator=true; fi
  # These credentials are test-only and only exist inside this disposable cluster.
  printf '{"Tenant":"%s","Domain":"%s","ID":"%s","Owner":"%s-owner","Role":"%s","Token":"%s","Expires":"2099-01-01T00:00:00Z","Curator":%s}\n' "$tenant" "$domain" "$actor" "$actor" "$role" "$actor" "$curator" | go run ./cmd/holocron provision
done
go test -race -p 1 -count=1 ./... "$@"
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts -p 'test_*.py'
