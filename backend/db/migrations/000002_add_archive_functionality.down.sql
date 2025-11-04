-- Remove archive functionality
DROP TRIGGER IF EXISTS trigger_insert_archive_totals ON transactions;
DROP TRIGGER IF EXISTS trigger_update_archive_totals ON transactions;
DROP FUNCTION IF EXISTS update_archive_totals();
DROP VIEW IF EXISTS active_transactions;
DROP INDEX IF EXISTS idx_archive_person_totals_person_id;
DROP INDEX IF EXISTS idx_archive_person_totals_archive_id;
DROP INDEX IF EXISTS idx_transactions_archive_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS archive_id;
DROP TABLE IF EXISTS archive_person_totals;
DROP TABLE IF EXISTS archives;