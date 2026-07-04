# Production Readiness Plan

Status of the review performed on 2026-07-03 (branch `prod_ready`). Verdict: **not
production ready yet** — the read-only check path is fine, but the `-fix*` paths have
correctness bugs that can misbehave against real replication topologies. This file is
the punch list; check items off as they land.

## P0 — Correctness & safety (must fix before any production use)

- [x] **Session-scoped statements run on a connection pool.**
  `SET GTID_NEXT`, `SET sql_log_bin=0`, `BEGIN`, `COMMIT` are executed via `db.Exec()`
  on `*sql.DB`, which is a pool — consecutive statements can land on *different*
  connections. The empty transaction may commit under the wrong GTID, or with binary
  logging still enabled. Fix: pin a single `*sql.Conn` for the whole fix sequence.
- [x] **No cleanup on mid-fix failure.** If applying entries fails partway, the session
  is left with `GTID_NEXT` set, `sql_log_bin=0`, and replication stopped. Fix: `defer`
  restoration of `GTID_NEXT='AUTOMATIC'` and `sql_log_bin=1`, and always attempt to
  restart replication.
- [x] **MySQL 8.4+ / 9.x incompatibility.** `SHOW MASTER STATUS` and `STOP/START SLAVE`
  were removed in 8.4. Version detection only recognized `8.0.x`; on 8.4/9.x the tool
  would issue removed statements and fail (or worse, fail halfway through a fix). Fix:
  proper semver comparison (>= 8.0.22 → `REPLICA` commands) and
  `SHOW BINARY LOG STATUS` with fallback to `SHOW MASTER STATUS`.
- [x] **SQL built by string concatenation.** GTID sets were interpolated directly into
  `gtid_subtract('...','...')` and `SET GTID_NEXT='...'`. Values come from the server,
  but a malformed/hostile GTID string still breaks the statement. Fix: placeholders for
  queries; strict GTID-entry validation before use in `SET GTID_NEXT` (which cannot be
  parameterized).
- [x] **Dead/misleading code in the fix path.** A regex tried to match
  `"Errant Transactions: <gtid>"` against `Executed_Gtid_Set` output — it can never
  match (the comment even said "even if it seems incorrect"). Removed.
- [x] **Multi-target loop is a lie.** `strings.Split(target, ",")` iterated "targets"
  but only one target connection ever exists, so every iteration re-queried the same
  server. Removed the loop; the CLI takes exactly one target.
- [x] **`-fix-missing-replica` silently discards data.** Injecting empty transactions
  for *missing* GTIDs marks them executed on the replica — the source will never send
  that data again. Legitimate (pt-slave-restart-style) but must warn loudly. Fix:
  prominent warning printed before applying.
- [x] **Duplicated fix logic.** `applyMissingGtidFixes` duplicated ~100 lines of the
  `-fix-replica` orchestration in `CheckGtidSetSubset`. Consolidated into one
  `applyGtidsToReplica` path so future fixes land in one place.

## P1 — Robustness & UX

- [x] **Connection/read/write timeouts.** DSN now sets `timeout`, `readTimeout`,
  `writeTimeout` so a hung server can't hang the tool forever.
- [x] **`-version` flag.** `.goreleaser.yml` injects `main.version/commit/date` ldflags
  but `main.go` had no such variables, so release binaries had no version info.
- [x] **`flag.Parse()` in `init()`.** Moved to `main()` (parsing in `init` breaks
  `go test` flag handling and is a known Go antipattern).
- [x] **Debug output in normal operation.** `Debug:` prints removed from the
  replication-status check; user-facing status report kept.
- [x] **Credentials only from `~/.my.cnf`.** Now honors `MYSQL_USER` / `MYSQL_PASSWORD`
  env vars as an override, uses `os.UserHomeDir()` (works on Windows builds we ship),
  and tolerates `user = x` spacing.
- [x] **`-dry-run` flag** that prints the exact statements a `-fix*` run would execute
  (including the STOP/START REPLICA orchestration for replica-side fixes).
- [x] **Confirmation prompt** (`-yes` to skip) before destructive fix operations.
  Non-interactive runs (cron, closed stdin) hit EOF and abort safely; piping `yes`
  or passing `-yes` opts in explicitly. Note: scripted `-fix` callers must now add
  `-yes` — this is an intentional breaking change.
- [x] **Structured exit codes** so the tool is scriptable in monitoring:
  0 = in sync (or fix applied), 1 = error, 2 = errant/missing transactions remain
  (found in check mode, shown in dry-run, or fix declined at the prompt).
  Caveat: Go's flag package also exits 2 on CLI misuse — distinguishable by the
  usage text on stderr.
- [x] **Context plumbing** (`context.Context` through all DB calls) for cancellation
  via Ctrl-C/SIGTERM. Cleanup paths (GTID_NEXT reset, re-enable binlog, restart
  replication) use `context.WithoutCancel` so an interrupt mid-fix still restores state.
- [x] **Retry heuristic** now uses `errors.Is`/`errors.As` against driver error types
  (bad/invalid conn, `net.Error`, lock-wait-timeout 1205, deadlock 1213) instead of
  error-string substring matching. Non-retryable errors fail fast.

## P2 — Testing & CI

- [x] **CI never ran tests or vet.** `build.yml` only cross-compiled. Added
  `go vet` + `go test -short` before builds.
- [x] **Makefile/test mismatch.** `test-integration` passed `-tags=integration` but no
  test file has that build tag (they gate on `testing.Short()`), so `make test` was
  silently running integration tests too. Fixed: `test` uses `-short`,
  `test-integration` runs the full suite.
- [x] **No unit coverage of the fix paths.** Added sqlmock tests
  (`gtid_mock_test.go`) covering: the exact statement sequence of a fix, rejection
  of invalid/injection GTID entries, GTID_NEXT reset after mid-fix failure, the
  `SHOW BINARY LOG STATUS` -> `SHOW MASTER STATUS` fallback (both directions),
  replica column-name handling, retry fail-fast, and context cancellation.
- [x] **golangci-lint** in CI (validated clean locally with v1.64 first).
- [x] **Integration test in CI** using the existing docker-compose setup
  (`test-integration` job in build.yml).
- [x] **Test the version-detection matrix** — `TestReplicationCommandsForVersion`
  covers 5.7, 8.0.21, 8.0.22, 8.4, 9.x, MariaDB, and garbage strings.

## P3 — Docs & release hygiene

- [x] **`.gitignore` misses `bin/` and `dist/`** (a built binary was sitting untracked
  in `bin/`).
- [x] **README rewrite.** Now: what the tool does, install, flags reference, exit
  codes, recommended fix workflow, credential setup, dev/testing guide, releases.
- [x] ~~`CHANGED.md` → `CHANGELOG.md`~~ CHANGED.md turned out to be an *attribution*
  file (Orchestrator code provenance), not a changelog — left in place and linked
  from the README credits section. GoReleaser generates release changelogs.
- [x] **Document `-fix-missing-replica` data-loss semantics** in README (warning
  section) and in the flag's help text.
- [x] **Dependabot** added for gomod + github-actions (weekly).
- [ ] **Pin GitHub Actions by SHA** (supply-chain hygiene) — needs SHA lookup per
  action; dependabot will keep them fresh once pinned.
