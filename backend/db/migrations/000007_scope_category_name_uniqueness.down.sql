-- Revert category name uniqueness back to a global unique constraint.

DROP INDEX IF EXISTS idx_categories_unique_parent_name;
DROP INDEX IF EXISTS idx_categories_unique_top_level_name;

ALTER TABLE categories
ADD CONSTRAINT categories_name_key UNIQUE (name);
