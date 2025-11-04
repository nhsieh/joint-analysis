package main

import "time"

// Transaction represents a financial transaction
type Transaction struct {
	ID              string    `json:"id"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	AssignedTo      []string  `json:"assigned_to"`
	DateUploaded    time.Time `json:"date_uploaded"`
	FileName        *string   `json:"file_name"`
	TransactionDate *string   `json:"transaction_date"`
	PostedDate      *string   `json:"posted_date"`
	CardNumber      *string   `json:"card_number"`
	CategoryID      *string   `json:"category_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Person represents a person who can be assigned to transactions
type Person struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Category represents a transaction category
type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Color       *string   `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PersonTotal represents the total amount for a person
type PersonTotal struct {
	Name  string  `json:"name"`
	Total float64 `json:"total"`
}

// Total represents the total amount for a person (alternative format)
type Total struct {
	Person string  `json:"person"`
	Total  float64 `json:"total"`
}

// Archive represents an archived collection of transactions
type Archive struct {
	ID               string        `json:"id"`
	Description      *string       `json:"description"`
	ArchivedAt       time.Time     `json:"archived_at"`
	TransactionCount int           `json:"transaction_count"`
	TotalAmount      float64       `json:"total_amount"`
	PersonTotals     []PersonTotal `json:"person_totals,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// ArchiveRequest represents the request structure for creating an archive
type ArchiveRequest struct {
	Description string `json:"description"`
}
