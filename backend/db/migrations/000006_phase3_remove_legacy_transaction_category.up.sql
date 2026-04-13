-- Phase 3: remove legacy transactions.category_id after split migration

-- Ensure every transaction has at least one split before dropping legacy category_id.
-- If category_id is null, default to top-level "Other" category.
INSERT INTO transaction_splits (transaction_id, amount, category_id, notes)
SELECT
    t.id,
    ABS(t.amount),
    COALESCE(
        t.category_id,
        (
            SELECT c.id
            FROM categories c
            WHERE c.name = 'Other' AND c.parent_id IS NULL
            LIMIT 1
        )
    ),
    'Backfilled from legacy transaction category'
FROM transactions t
WHERE NOT EXISTS (
    SELECT 1
    FROM transaction_splits ts
    WHERE ts.transaction_id = t.id
);

DROP VIEW IF EXISTS active_transactions;

ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_category_id;
DROP INDEX IF EXISTS idx_transactions_category_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS category_id;

CREATE VIEW active_transactions AS
SELECT * FROM transactions
WHERE archive_id IS NULL;
