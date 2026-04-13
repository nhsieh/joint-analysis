-- Roll back Phase 3: reintroduce legacy transactions.category_id

DROP VIEW IF EXISTS active_transactions;

ALTER TABLE transactions
ADD COLUMN category_id UUID;

CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);

ALTER TABLE transactions
ADD CONSTRAINT fk_transactions_category_id
FOREIGN KEY (category_id) REFERENCES categories(id)
ON UPDATE CASCADE ON DELETE SET NULL;

-- Best-effort restore of a legacy category from the earliest split row per transaction.
UPDATE transactions t
SET category_id = restored.category_id
FROM (
    SELECT DISTINCT ON (ts.transaction_id)
        ts.transaction_id,
        ts.category_id
    FROM transaction_splits ts
    ORDER BY ts.transaction_id, ts.created_at ASC
) AS restored
WHERE restored.transaction_id = t.id;

CREATE VIEW active_transactions AS
SELECT * FROM transactions
WHERE archive_id IS NULL;
