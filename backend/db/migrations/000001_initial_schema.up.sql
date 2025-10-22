-- Initial schema for joint-analysis application
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create people table
CREATE TABLE people (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    color VARCHAR(7), -- hex color code
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    description VARCHAR(500) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL,
    assigned_to UUID[],
    date_uploaded TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    file_name VARCHAR(255),
    transaction_date DATE,
    posted_date DATE,
    card_number VARCHAR(20),
    category_id UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_people_name ON people(name);
CREATE INDEX idx_categories_name ON categories(name);
CREATE INDEX idx_transactions_assigned_to ON transactions USING GIN(assigned_to);
CREATE INDEX idx_transactions_date_uploaded ON transactions(date_uploaded);
CREATE INDEX idx_transactions_transaction_date ON transactions(transaction_date);
CREATE INDEX idx_transactions_file_name ON transactions(file_name);
CREATE INDEX idx_transactions_category_id ON transactions(category_id);

-- Add foreign key constraints
-- Note: PostgreSQL doesn't support foreign key constraints on arrays directly
-- We'll handle referential integrity in the application layer

ALTER TABLE transactions
ADD CONSTRAINT fk_transactions_category_id
FOREIGN KEY (category_id) REFERENCES categories(id)
ON UPDATE CASCADE ON DELETE SET NULL;

-- Insert default categories
INSERT INTO categories (name, description, color) VALUES
    ('Food & Dining', 'Restaurants, groceries, food delivery', '#FF7043'),
    ('Transportation', 'Gas, public transit, rideshare, parking', '#42A5F5'),
    ('Shopping', 'Retail purchases, online shopping', '#AB47BC'),
    ('Entertainment', 'Movies, concerts, streaming services', '#66BB6A'),
    ('Utilities', 'Electric, gas, water, internet, phone', '#FFA726'),
    ('Health & Fitness', 'Medical expenses, pharmacy, insurance, gym membership', '#EF5350'),
    ('Travel', 'Flights, hotels, vacation expenses', '#26C6DA'),
    ('Fees', 'Bank fees, interest charges, service fees', '#8D6E63'),
    ('Pets', 'Pet care, grooming, supplies, insurance', '#13AFAD'),
    ('Other', 'Miscellaneous expenses', '#78909C');