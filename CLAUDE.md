# yomauro

Shared tooling, conventions, and patterns for Claude Code projects.

## Makefile Convention

**All entrypoints go through the root `Makefile`.** Every operation — running the server, building, migrating, generating code, installing dependencies — must have a `make` target. No bare `go run`, `npm run`, or `migrate` commands in documentation or workflows. If it's a thing you do, it's a `make` target.

## Go Conventions

- All packages live under `internal/` (unexported to external modules)
- `repository/` uses sqlc-generated types; raw queries in `internal/repository/queries/`
- Background jobs use River (PostgreSQL-backed queue); job definitions in `internal/job/`
- Settings loaded once at startup from environment variables via `internal/config`

## Configuration

Settings via `config/.env` (primary), `.env` (fallback). Go `internal/config` package reads environment variables. See `.env.example` for all variables.

## Database Migration Philosophy

**Migrations are irreversible.** Never write a migration that assumes a clean rollback. Prefer additive changes (add a column, add a table). When a column or table needs to be restructured, use expand-and-contract: add the new column/table, migrate data, remove the old one in a later separate migration. Never rename or change a column type destructively in a single step.

## Stack Layers

Define explicit layer boundaries for automated issue classification and scope assessment. An issue touches a single layer if all its changes fall within exactly one boundary.

Example layer table (customize per project):

| Layer | Scope | Paths |
|-------|-------|-------|
| **Database** | Schema migrations only | `migrations/` |
| **Data Access** | Query definitions and generated types | `internal/repository/` |
| **Business Logic** | Domain computation and classification | `internal/analytics/`, `internal/classify/` |
| **API** | HTTP handlers and OpenAPI spec | `internal/handler/`, `api/openapi.yaml` |
| **Frontend** | UI components, pages, lib | `web/src/` |

Identify packages that are inherently cross-cutting and never qualify as single-layer (e.g., ETL/seed, background jobs, external API clients).

**Additive vs. destructive** — Additive means new files, new functions, new columns (nullable or with default), new tables, new routes. Destructive means modifying existing SQL queries, renaming or removing API fields, changing column types, or altering existing migration files. Additive changes can be factory-automated. Destructive changes require human review regardless of scope.

## Implementation Conventions

**Verify contracts against the actual codebase before implementing.** When working from an implementation issue, treat the Data Contracts section as a starting hypothesis, not ground truth. Before writing code, read the relevant files and confirm that field names, column names, type signatures, and query names match what the issue claims. If they don't, work from the codebase and note the discrepancy. Contracts in issues are written before the code they describe exists and may have drifted.

**Annotate the meaning of values in contracts, not just their type.** A type mismatch fails at compile time; a meaning mismatch fails silently. When writing or reading Data Contracts, always include what a value represents alongside its type — e.g. `cut_signal_threshold float64 // CoversPoint × 0.85`, not just `cut_signal_threshold float64`. Two fields can be structurally identical but semantically incompatible, and no tool catches that except the spec.

## Go API Boundary Convention

Every Go package under `internal/` has a **boundary file** named `<package>.go` (e.g., `analytics/analytics.go`). This is the package's public API surface.

- **Boundary file contains:** package doc comment, all exported type definitions (structs, interfaces, type aliases, constants, exported vars/errors), and constructor functions (`New*`).
- **Boundary file does NOT contain:** method implementations, unexported types, or business logic.
- **Naming rule:** the boundary file matches the package directory name — `handler/handler.go`, `seed/seed.go`, etc.
- **Sub-packages** get their own boundary file (e.g., `integrations/opentable/opentable.go`).
- **New packages** must create their boundary file before any implementation files.
- **Exemption:** Code-generated packages (e.g., sqlc output) use their generator's config as the contract, not a boundary file.
