-- Add archive functionality
-- Create archives table to store archive metadata
CREATE TABLE archives (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    description TEXT,
    archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    transaction_count INTEGER NOT NULL DEFAULT 0,
    total_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create archive_person_totals table to store individual person totals for each archive
CREATE TABLE archive_person_totals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    archive_id UUID NOT NULL REFERENCES archives(id) ON UPDATE CASCADE ON DELETE CASCADE,
    person_id UUID NOT NULL REFERENCES people(id) ON UPDATE CASCADE ON DELETE CASCADE,
    total_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(archive_id, person_id)
);

-- Add archive_id column to transactions table
ALTER TABLE transactions
ADD COLUMN archive_id UUID REFERENCES archives(id) ON UPDATE CASCADE ON DELETE SET NULL;

-- Create indexes for better query performance
CREATE INDEX idx_transactions_archive_id ON transactions(archive_id);
CREATE INDEX idx_archive_person_totals_archive_id ON archive_person_totals(archive_id);
CREATE INDEX idx_archive_person_totals_person_id ON archive_person_totals(person_id);

-- Create view for active (non-archived) transactions
CREATE VIEW active_transactions AS
SELECT * FROM transactions
WHERE archive_id IS NULL;

-- Create function to update archive totals when transactions are archived
CREATE OR REPLACE FUNCTION update_archive_totals()
RETURNS TRIGGER AS $$
BEGIN
    -- Update the archive's transaction count and total amount
    UPDATE archives
    SET
        transaction_count = (
            SELECT COUNT(*)
            FROM transactions
            WHERE archive_id = NEW.archive_id
        ),
        total_amount = (
            SELECT COALESCE(SUM(t.amount / array_length(t.assigned_to, 1) * array_length(t.assigned_to, 1)), 0)
            FROM transactions t
            WHERE t.archive_id = NEW.archive_id
              AND t.assigned_to IS NOT NULL
              AND array_length(t.assigned_to, 1) > 0
        ),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.archive_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to update archive totals when transactions are archived
CREATE TRIGGER trigger_update_archive_totals
    AFTER UPDATE OF archive_id ON transactions
    FOR EACH ROW
    WHEN (OLD.archive_id IS DISTINCT FROM NEW.archive_id AND NEW.archive_id IS NOT NULL)
    EXECUTE FUNCTION update_archive_totals();

-- Create trigger to update archive totals when archived transactions are inserted
CREATE TRIGGER trigger_insert_archive_totals
    AFTER INSERT ON transactions
    FOR EACH ROW
    WHEN (NEW.archive_id IS NOT NULL)
    EXECUTE FUNCTION update_archive_totals();