# ADR-004: Categorization Rules Engine

## Status
Accepted

## Context

Transaction auto-categorization is currently handled by a hardcoded mapping in `backend/utils.go` (`mapTransactionCategory`). This mapping covers 6 static CSV-category-to-category translations (e.g. `"Dining"` → `"Food & Dining"`) and falls back to the `"Other"` category.

This approach has two problems:
1. **Not user-configurable**: Adding or changing a rule requires a code change and redeploy.
2. **Limited matching**: Rules only match on the CSV category column, not on the transaction description — which is often more descriptive and useful.

Users need to define their own rules, e.g. "if the description contains 'Trader Joe' → Groceries" or "if the CSV category is 'Dining' → Food & Dining".

## Decision

Introduce a persistent **Categorization Rules** system stored in PostgreSQL. Each rule has a `match_value` (a case-insensitive substring) that is tested against **both** the transaction description and the CSV category field during import. The first matching rule (by priority) wins.

The existing 6 hardcoded mappings are **migrated into the database as seeded rules** and removed from the code. A new Settings section ("Rules") provides CRUD management of rules through the UI.

### Rule Schema

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | Auto-generated |
| `match_value` | VARCHAR(255) NOT NULL | Case-insensitive substring to match |
| `category_id` | UUID FK → categories(id) ON DELETE CASCADE | Target category |
| `priority` | INT DEFAULT 0 | Lower number = higher priority; first match wins |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### Matching Logic (CSV Import)

For each imported transaction:
  1. Load all rules ORDER BY priority ASC, created_at ASC
  2. For each rule:
     a. If LOWER(description) CONTAINS LOWER(match_value) → assign category, stop
     b. If LOWER(csv_category) CONTAINS LOWER(match_value) → assign category, stop
  3. If no rule matched → fall back to "Other" category (existing behavior)

### API

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/rules` | List all rules (with resolved category name) |
| POST | `/api/rules` | Create a new rule |
| PUT | `/api/rules/:id` | Update a rule |
| DELETE | `/api/rules/:id` | Delete a rule |

### Settings UI

A new "Rules" section is added to `Settings.tsx` below the existing "Categories" section. It displays a table of rules (match value, target category, priority) with Add/Edit/Delete actions.

## Consequences

### Pros

1. **User-configurable**: Rules can be created, edited, and deleted without code changes
2. **Richer matching**: Rules match against transaction descriptions — usually much more specific than the import category
3. **Ordered by priority**: Users can control which rule wins when multiple match
4. **Safe migration**: Existing hardcoded rules are seeded into the DB so categorization behavior is unchanged immediately after migration

### Cons

1. **Extra DB query on import**: Each CSV upload loads all rules before processing rows (acceptable at current scale)
2. **No retroactive application**: Rules only affect future imports; existing transactions must be manually re-categorized
3. **Substring-only matching**: No regex or exact-match support in v1; harder edge cases require workarounds

### Files Changed

| File | Change |
|---|---|
| `docs/adr/004-rules.md` | This file |
| `backend/db/migrations/000004_add_categorization_rules.up.sql` | New — `categorization_rules` table + seed 6 default rules |
| `backend/db/migrations/000004_add_categorization_rules.down.sql` | New — drop table |
| `backend/db/query.sql` | Add `GetRules`, `GetRuleByID`, `CreateRule`, `UpdateRule`, `DeleteRule` |
| `backend/db/generated/` | Regenerated via `sqlc generate` |
| `backend/models.go` | Add `Rule` struct |
| `backend/rules.go` | New — CRUD handlers with Swagger annotations |
| `backend/rules_test.go` | New — handler tests |
| `backend/main.go` | Register `/api/rules` routes |
| `backend/utils.go` | Replace hardcoded `csvCategoryMap` with DB rule lookup |
| `backend/docs/` | Regenerated via `make generate-docs` |
| `frontend/src/types.ts` | Add `Rule` interface |
| `frontend/src/Settings.tsx` | Add Rules section below Categories |

## Alternatives Considered

### Alternative 1: Keep hardcoded mappings, add rules on top
- Hardcoded rules would silently take priority over user-defined ones, causing confusion.
- **Rejected**: Seeding them as DB rules gives identical behavior with full user control.

### Alternative 2: Regex matching
- More powerful but significantly harder to author for non-technical users.
- **Rejected for v1**: Case-insensitive substring covers the vast majority of use cases. Regex can be added as a `match_type` column in a future migration.

### Alternative 3: Retroactive rule application
- Would automatically re-categorize existing transactions when a rule is saved.
- **Rejected**: Side effects are hard to undo and may overwrite intentional manual categorizations. An explicit "Apply rules to existing transactions" action could be added in the future.

## Out of Scope

- Regex or wildcard match types (future `match_type` column)
- Retroactive application of rules to existing transactions
- Match field selection (description vs. CSV category vs. either) — always matches both in v1
- Rule import/export

---
**Date**: March 14, 2026
**Supersedes**: None
**Superseded by**: None