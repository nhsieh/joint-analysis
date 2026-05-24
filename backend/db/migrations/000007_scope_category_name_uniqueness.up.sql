-- Scope category name uniqueness by hierarchy level.
-- Allow same name between a top-level category and its subcategories,
-- while still preventing duplicates among top-level categories and sibling subcategories.

-- Remove the old global uniqueness constraint on categories.name.
ALTER TABLE categories
DROP CONSTRAINT IF EXISTS categories_name_key;

-- Top-level categories (parent_id IS NULL) must have unique names.
CREATE UNIQUE INDEX idx_categories_unique_top_level_name
ON categories (name)
WHERE parent_id IS NULL;

-- Subcategories must have unique names within the same parent category.
CREATE UNIQUE INDEX idx_categories_unique_parent_name
ON categories (parent_id, name)
WHERE parent_id IS NOT NULL;
