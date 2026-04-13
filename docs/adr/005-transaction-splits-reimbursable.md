# ADR-005: Split Transactions with Reimbursable Portion

## Status
Proposed

## Context

Today, a transaction has exactly one `category_id` (`backend/models.go` and `transactions` table), so a single purchase cannot be split into two categorized portions.

The requested workflow is:
1. Keep one normal selectable category for the personal/shared expense portion.
2. Assign other portions to any categories as needed, including `Reimbursable`.

Typical example:
- Total: $80 meal
- $50 is normal `Food & Dining`
- $30 is `Reimbursable`

## Decision

Introduce **transaction splits** as first-class data. A transaction can have any number of split rows:
1. splits use user-selectable categories (`category_id` required)
2. categories are treated uniformly; `Reimbursable` has no special backend or UI logic

The top-level `transactions.category_id` will be retained temporarily for backward compatibility and phased out after frontend + API migration.

## Data Model

Add table `transaction_splits`:

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | Auto-generated |
| `transaction_id` | UUID FK -> transactions(id) ON DELETE CASCADE | Parent transaction |
| `amount` | NUMERIC(10,2) NOT NULL | Positive split amount |
| `category_id` | UUID NOT NULL FK -> categories(id) | Required for all splits |
| `notes` | TEXT NULL | Optional memo |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

Constraints:
1. Sum of `transaction_splits.amount` for a transaction must equal `ABS(transactions.amount)`.
2. `category_id` must reference a valid category for every split.

## API Changes

### New payload on transaction update

`PUT /api/transactions/{id}/splits`

```json
{
  "splits": [
    {
      "amount": 50.00,
      "category_id": "<food-category-uuid>"
    },
    {
      "amount": 30.00,
      "category_id": "<reimbursable-category-uuid>",
      "notes": "Company reimbursement"
    }
  ]
}
```

Validation rules:
1. At least 1 split is required.
2. Split amounts must be positive and total to transaction absolute amount.
3. `category_id` is required for every split.

### Read model

Current API behavior:
1. `GET /api/transactions` returns the existing transaction shape (no embedded `splits` array yet).
2. `GET /api/transactions/{id}/splits` returns split rows for that transaction.

## UI/UX Changes

In transaction editing UI:
1. Add "Split transaction" toggle.
2. Split editor supports adding/removing any number of rows:
  - each row has category picker + amount + optional notes
  - new rows default category to `Reimbursable` only when user chooses that shortcut
3. Real-time validation that all split amounts sum to original transaction amount.

Display behavior:
1. Current implementation: users can edit splits from the transaction action menu, but the main transaction table does not yet render split row summaries.
2. Target behavior: transactions should show split rows and selected categories inline or in an expanded details view.
3. `Reimbursable` is displayed like any other category.
4. Any include/exclude behavior is done through generic category filters, not hardcoded category rules.

## Consequences

### Positive
1. Supports the exact user workflow and scales to multi-category splits without schema changes.
2. Maintains auditability without duplicating parent transactions.
3. Keeps reporting flexible through normal category-based filtering.

### Negative
1. Increases schema and API complexity.
2. Requires migration from single `category_id` model.
3. Adds validation edge cases during CSV imports and manual edits, especially with many split rows.

## Implementation Status

- [x] Database migration added for `transaction_splits` table.
- [x] Split endpoints implemented:
  - `GET /api/transactions/{id}/splits`
  - `PUT /api/transactions/{id}/splits`
- [x] Frontend split editor implemented (add/remove/edit/save split rows).
- [x] Backend tests added for split behavior and validation.
- [x] `GET /api/transactions` includes embedded `splits` array.
- [x] Main transaction list/dashboard renders split summaries.
- [x] Totals/category aggregation fully migrated to read from `transaction_splits`.

## Migration Plan

Phase 1:
1. Add `transaction_splits` table and backfill one `normal` split per existing transaction with `category_id` copied from `transactions.category_id`.
2. Keep existing category endpoint behavior unchanged.

Phase 2:
1. Add splits endpoints and frontend split editor.
2. Update totals/category endpoints to aggregate from `transaction_splits` with standard category-based filtering.
3. Update transaction list/dashboard UI to display split summaries (currently missing).

Phase 3:
1. Deprecate direct writes to `transactions.category_id`.
2. Remove legacy field once clients are migrated.

## Alternatives Considered

### Alternative 1: Keep single `transactions.category_id` only
- Rejected: cannot represent multi-category splits for one transaction.

### Alternative 2: Duplicate transaction rows
- Rejected: duplicates source transaction identity and complicates deduplication/import logic.

### Alternative 3: Single reimbursable amount column only
- Rejected: does not generalize to future multi-split needs and lacks category-level control.

### Alternative 4: Dedicated `is_reimbursable` flag on splits
- Rejected: unnecessary duplication because `Reimbursable` already exists as a category and `category_id` is sufficient.

## Answer to the Original Question

**Current state:** No, not today. The current data model supports only one category per transaction.

**After this ADR is implemented:** Yes. A transaction can be split into:
1. any number of category-based amounts, including
2. one or more amounts assigned to `Reimbursable` with no hardcoded special handling.

---
**Date**: April 13, 2026
**Supersedes**: None
**Superseded by**: None