-- Create categorization_rules table
CREATE TABLE categorization_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_value VARCHAR(255) NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_categorization_rules_priority ON categorization_rules(priority ASC, created_at ASC);

-- Seed the 6 hardcoded mappings as rules with their resolved category IDs
INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'Gas', c.id, 0 FROM categories c WHERE c.name = 'Gas & Charging';

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'RALPHS', c.id, 0 FROM categories c WHERE c.name = 'Groceries';

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT '99 RANCH', c.id, 0 FROM categories c WHERE c.name = 'Groceries';

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'SUPERMARKET', c.id, 0 FROM categories c WHERE c.name = 'Groceries';

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'Spotify', c.id, 1 FROM categories c WHERE c.name = 'Entertainment' AND c.parent_id IS NULL;

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'Insurance', c.id, 1 FROM categories c WHERE c.name = 'Other' AND c.parent_id IS NULL;

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'Dining', c.id, 2 FROM categories c WHERE c.name = 'Food & Dining' AND c.parent_id IS NULL;

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'Other Travel', c.id, 3 FROM categories c WHERE c.name = 'Travel' AND c.parent_id IS NULL;

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'Merchandise', c.id, 4 FROM categories c WHERE c.name = 'Shopping' AND c.parent_id IS NULL;

INSERT INTO categorization_rules (match_value, category_id, priority)
SELECT 'Fee/Interest Charge', c.id, 5 FROM categories c WHERE c.name = 'Fees' AND c.parent_id IS NULL;
