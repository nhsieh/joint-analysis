-- People queries
-- name: GetPeople :many
SELECT id, name, email, created_at, updated_at
FROM people
ORDER BY name;

-- name: GetPersonByID :one
SELECT id, name, email, created_at, updated_at
FROM people
WHERE id = $1;

-- name: GetPersonByName :one
SELECT id, name, email, created_at, updated_at
FROM people
WHERE name = $1;

-- name: CreatePerson :one
INSERT INTO people (name, email)
VALUES ($1, $2)
RETURNING id, name, email, created_at, updated_at;

-- name: UpdatePerson :one
UPDATE people
SET name = $2, email = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, name, email, created_at, updated_at;

-- name: DeletePerson :exec
DELETE FROM people
WHERE id = $1;

-- Categories queries
-- name: GetCategories :many
SELECT id, name, description, color, created_at, updated_at
FROM categories
ORDER BY name;

-- name: GetCategoryByID :one
SELECT id, name, description, color, created_at, updated_at
FROM categories
WHERE id = $1;

-- name: GetCategoryByName :one
SELECT id, name, description, color, created_at, updated_at
FROM categories
WHERE name = $1;

-- name: CreateCategory :one
INSERT INTO categories (name, description, color)
VALUES ($1, $2, $3)
RETURNING id, name, description, color, created_at, updated_at;

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, description = $3, color = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, name, description, color, created_at, updated_at;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1;

-- Transactions queries
-- name: GetTransactions :many
SELECT id, description, amount, assigned_to, date_uploaded, file_name,
       transaction_date, posted_date, card_number, category, category_id,
       created_at, updated_at
FROM transactions
ORDER BY date_uploaded DESC;

-- name: GetTransactionByID :one
SELECT id, description, amount, assigned_to, date_uploaded, file_name,
       transaction_date, posted_date, card_number, category, category_id,
       created_at, updated_at
FROM transactions
WHERE id = $1;

-- name: GetTransactionsByAssignedTo :many
SELECT id, description, amount, assigned_to, date_uploaded, file_name,
       transaction_date, posted_date, card_number, category, category_id,
       created_at, updated_at
FROM transactions
WHERE assigned_to = $1
ORDER BY date_uploaded DESC;

-- name: GetTransactionsByFileName :many
SELECT id, description, amount, assigned_to, date_uploaded, file_name,
       transaction_date, posted_date, card_number, category, category_id,
       created_at, updated_at
FROM transactions
WHERE file_name = $1
ORDER BY date_uploaded DESC;

-- name: CreateTransaction :one
INSERT INTO transactions (description, amount, file_name, transaction_date, posted_date, card_number, category)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category, category_id,
          created_at, updated_at;

-- name: UpdateTransactionAssignment :one
UPDATE transactions
SET assigned_to = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category, category_id,
          created_at, updated_at;

-- name: UpdateTransactionCategory :one
UPDATE transactions
SET category_id = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category, category_id,
          created_at, updated_at;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1;

-- name: GetTotalsByAssignedTo :many
SELECT assigned_to, SUM(amount) as total
FROM transactions
WHERE assigned_to IS NOT NULL AND assigned_to != ''
GROUP BY assigned_to
ORDER BY assigned_to;

-- name: GetTotalsByCategory :many
SELECT c.name as category_name, SUM(t.amount) as total
FROM transactions t
JOIN categories c ON t.category_id = c.id
WHERE t.category_id IS NOT NULL
GROUP BY c.id, c.name
ORDER BY c.name;