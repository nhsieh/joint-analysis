-- Add subcategory support: 2-level hierarchy (Category -> Subcategory)
ALTER TABLE categories
ADD COLUMN parent_id UUID REFERENCES categories(id) ON DELETE CASCADE;

-- Index for efficient subcategory lookups
CREATE INDEX idx_categories_parent_id ON categories(parent_id);

-- Seed default subcategories
INSERT INTO categories (name, description, color, parent_id)
SELECT 'Groceries', 'Grocery store purchases', c.color, c.id
FROM categories c WHERE c.name = 'Food & Dining' AND c.parent_id IS NULL;

INSERT INTO categories (name, description, color, parent_id)
SELECT 'Restaurants', 'Restaurant and food delivery', c.color, c.id
FROM categories c WHERE c.name = 'Food & Dining' AND c.parent_id IS NULL;

INSERT INTO categories (name, description, color, parent_id)
SELECT 'Flights', 'Airfare and airline fees', c.color, c.id
FROM categories c WHERE c.name = 'Travel' AND c.parent_id IS NULL;

INSERT INTO categories (name, description, color, parent_id)
SELECT 'Accommodations', 'Hotels, rentals, and lodging', c.color, c.id
FROM categories c WHERE c.name = 'Travel' AND c.parent_id IS NULL;

INSERT INTO categories (name, description, color, parent_id)
SELECT 'Gas & Charging', 'Gas stations and EV charging', c.color, c.id
FROM categories c WHERE c.name = 'Transportation' AND c.parent_id IS NULL;

INSERT INTO categories (name, description, color, parent_id)
SELECT 'Insurance', 'Insurance-related expenses', c.color, c.id
FROM categories c WHERE c.name = 'Transportation' AND c.parent_id IS NULL;

INSERT INTO categories (name, description, color, parent_id)
SELECT 'Parking', 'Parking-related expenses', c.color, c.id
FROM categories c WHERE c.name = 'Transportation' AND c.parent_id IS NULL;