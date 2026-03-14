-- Remove subcategory support
-- Delete all seeded subcategories
DELETE FROM categories WHERE parent_id IS NOT NULL;

-- Drop the index and column
DROP INDEX IF EXISTS idx_categories_parent_id;
ALTER TABLE categories DROP COLUMN IF EXISTS parent_id;
