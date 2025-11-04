-- People queries
-- name: GetPeople :many
SELECT id, name, email, created_at, updated_at
FROM people
ORDER BY created_at;

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
       transaction_date, posted_date, card_number, category_id,
       created_at, updated_at
FROM transactions
ORDER BY date_uploaded DESC;

-- name: GetTransactionByID :one
SELECT id, description, amount, assigned_to, date_uploaded, file_name,
       transaction_date, posted_date, card_number, category_id,
       created_at, updated_at
FROM transactions
WHERE id = $1;

-- name: GetTransactionsByAssignedTo :many
SELECT id, description, amount, assigned_to, date_uploaded, file_name,
       transaction_date, posted_date, card_number, category_id,
       created_at, updated_at
FROM transactions
WHERE $1 = ANY(assigned_to)
ORDER BY date_uploaded DESC;

-- name: GetTransactionsByFileName :many
SELECT id, description, amount, assigned_to, date_uploaded, file_name,
       transaction_date, posted_date, card_number, category_id,
       created_at, updated_at
FROM transactions
WHERE file_name = $1
ORDER BY date_uploaded DESC;

-- name: CreateTransaction :one
INSERT INTO transactions (description, amount, file_name, transaction_date, posted_date, card_number, category_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category_id,
          created_at, updated_at;

-- name: FindDuplicateTransaction :one
SELECT COUNT(*)
FROM transactions
WHERE description = $1
  AND amount = $2
  AND transaction_date = $3
  AND posted_date = $4
  AND card_number = $5;

-- name: UpdateTransactionAssignment :one
UPDATE transactions
SET assigned_to = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category_id,
          created_at, updated_at;

-- name: AddPersonToTransaction :one
UPDATE transactions
SET assigned_to = array_append(COALESCE(assigned_to, '{}'), $2), updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category_id,
          created_at, updated_at;

-- name: RemovePersonFromTransaction :one
UPDATE transactions
SET assigned_to = array_remove(assigned_to, $2), updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category_id,
          created_at, updated_at;

-- name: UnassignTransactionsByPerson :exec
UPDATE transactions
SET assigned_to = array_remove(assigned_to, $1), updated_at = CURRENT_TIMESTAMP
WHERE $1 = ANY(assigned_to);

-- name: UpdateTransactionCategory :one
UPDATE transactions
SET category_id = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, description, amount, assigned_to, date_uploaded, file_name,
          transaction_date, posted_date, card_number, category_id,
          created_at, updated_at;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1;

-- name: GetTotalsByAssignedTo :many
SELECT p.name as assigned_to, SUM(t.amount / array_length(t.assigned_to, 1))::numeric as total
FROM transactions t
CROSS JOIN LATERAL unnest(t.assigned_to) AS person_id
JOIN people p ON p.id = person_id
WHERE t.assigned_to IS NOT NULL AND array_length(t.assigned_to, 1) > 0
GROUP BY p.id, p.name
ORDER BY p.name;

-- name: GetTotalsByCategory :many
SELECT c.name as category_name, SUM(t.amount)::numeric as total
FROM transactions t
JOIN categories c ON t.category_id = c.id
WHERE t.category_id IS NOT NULL
GROUP BY c.id, c.name
ORDER BY c.name;

-- name: DeleteAllTransactions :exec
DELETE FROM transactions;