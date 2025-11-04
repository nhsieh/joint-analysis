package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"jointanalysis/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// CategoryMapping maps CSV categories to our predefined categories
type CategoryMapping struct {
	categoriesByName map[string]generated.Category
}

// Validation functions

// validateName validates that a name is not empty or just whitespace
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

// validateHexColor validates that a color is in hex format (#RRGGBB)
func validateHexColor(color string) error {
	if color == "" {
		return nil // Empty color is allowed
	}

	hexColorRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	if !hexColorRegex.MatchString(color) {
		return fmt.Errorf("color must be in hex format (#RRGGBB)")
	}
	return nil
}

// handleDatabaseError converts database errors to appropriate HTTP responses
func handleDatabaseError(err error) (statusCode int, message string) {
	errorStr := err.Error()

	// Check for unique constraint violations
	if strings.Contains(errorStr, "duplicate key value violates unique constraint") {
		if strings.Contains(errorStr, "people_name_key") {
			return http.StatusConflict, "Person with this name already exists"
		}
		if strings.Contains(errorStr, "categories_name_key") {
			return http.StatusConflict, "Category with this name already exists"
		}
		return http.StatusConflict, "Resource already exists"
	}

	// Check for not found errors
	if strings.Contains(errorStr, "no rows in result set") {
		return http.StatusNotFound, "Resource not found"
	}

	// Default to internal server error
	return http.StatusInternalServerError, "Internal server error"
}

// Category mapping functions

// mapTransactionCategory determines the best category for a transaction
func (cm *CategoryMapping) mapTransactionCategory(csvCategory string) *generated.Category {
	// First try direct mapping from CSV category
	if category, exists := cm.categoriesByName[csvCategory]; exists {
		return &category
	}

	// Map common CSV categories to our categories
	csvCategoryMap := map[string]string{
		"Gas/Automotive":      "Transportation",
		"Insurance":           "Healthcare",
		"Dining":              "Food & Dining",
		"Other Travel":        "Travel",
		"Merchandise":         "Shopping",
		"Fee/Interest Charge": "Fees",
	}

	if mappedCategory, exists := csvCategoryMap[csvCategory]; exists {
		if category, exists := cm.categoriesByName[mappedCategory]; exists {
			return &category
		}
	}

	// Default to "Other" if no match found
	if category, exists := cm.categoriesByName["Other"]; exists {
		return &category
	}

	return nil
}

// initializeCategoryMapping loads categories and creates keyword mappings
func initializeCategoryMapping() (*CategoryMapping, error) {
	categories, err := queries.GetCategories(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}

	categoriesByName := make(map[string]generated.Category)
	for _, category := range categories {
		categoriesByName[category.Name] = category
	}

	return &CategoryMapping{
		categoriesByName: categoriesByName,
	}, nil
}

// UUID and conversion utility functions

// convertUUIDArrayToNames converts an array of UUIDs to person names
func convertUUIDArrayToNames(uuidArray []pgtype.UUID) ([]string, error) {
	if len(uuidArray) == 0 {
		return []string{}, nil
	}

	var names []string
	for _, uuidPg := range uuidArray {
		if uuidPg.Valid {
			person, err := queries.GetPersonByID(context.Background(), uuidPg)
			if err != nil {
				log.Printf("Error getting person by ID %v: %v", uuidPg, err)
				continue // Skip invalid UUIDs instead of failing completely
			}
			names = append(names, person.Name)
		}
	}
	return names, nil
}

// convertNamesToUUIDArray converts person names to UUID array
func convertNamesToUUIDArray(names []string) ([]pgtype.UUID, error) {
	if len(names) == 0 {
		return []pgtype.UUID{}, nil
	}

	var uuids []pgtype.UUID
	for _, name := range names {
		person, err := queries.GetPersonByName(context.Background(), name)
		if err != nil {
			log.Printf("Error getting person by name %s: %v", name, err)
			continue // Skip invalid names instead of failing completely
		}
		uuids = append(uuids, person.ID)
	}
	return uuids, nil
}

// convertUUIDStringsToArray converts string UUIDs to pgtype.UUID array
func convertUUIDStringsToArray(uuidStrings []string) ([]pgtype.UUID, error) {
	if len(uuidStrings) == 0 {
		return []pgtype.UUID{}, nil
	}

	var uuids []pgtype.UUID
	for _, uuidStr := range uuidStrings {
		parsedUUID, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID format: %s", uuidStr)
		}
		uuids = append(uuids, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	}
	return uuids, nil
}

// Transaction conversion utility functions

// convertTransaction converts a generated.Transaction to our Transaction struct
func convertTransaction(t generated.Transaction) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

// convertTransactionFromGetRow converts a generated.Transaction to our Transaction struct
func convertTransactionFromGetRow(t generated.Transaction) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

// convertTransactionFromUpdateRow converts a generated.Transaction to our Transaction struct
func convertTransactionFromUpdateRow(t generated.Transaction) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

// convertTransactionFromUpdateAssignmentRow converts from update assignment result
func convertTransactionFromUpdateAssignmentRow(t generated.UpdateTransactionAssignmentRow) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

// convertTransactionFromUpdateCategoryRow converts from update category result
func convertTransactionFromUpdateCategoryRow(t generated.UpdateTransactionCategoryRow) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

// convertTransactionFromFields converts transaction fields to our Transaction struct
func convertTransactionFromFields(
	id pgtype.UUID,
	description string,
	amount pgtype.Numeric,
	assignedTo []pgtype.UUID,
	dateUploaded pgtype.Timestamp,
	fileName pgtype.Text,
	transactionDate pgtype.Date,
	postedDate pgtype.Date,
	cardNumber pgtype.Text,
	categoryID pgtype.UUID,
	createdAt pgtype.Timestamp,
	updatedAt pgtype.Timestamp,
) Transaction {
	result := Transaction{
		ID:           uuid.UUID(id.Bytes).String(), // Convert UUID to string
		Description:  description,
		AssignedTo:   []string{}, // Initialize as empty array
		DateUploaded: dateUploaded.Time,
		FileName:     nil,
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}

	// Convert numeric amount
	if amount.Valid {
		amountFloat, _ := amount.Float64Value()
		result.Amount = amountFloat.Float64
	}

	// Convert UUID array to person names
	if len(assignedTo) > 0 {
		names, err := convertUUIDArrayToNames(assignedTo)
		if err != nil {
			log.Printf("Error converting UUIDs to names: %v", err)
		} else {
			result.AssignedTo = names
		}
	}

	// Handle nullable fields
	if fileName.Valid {
		result.FileName = &fileName.String
	}
	if transactionDate.Valid {
		dateStr := transactionDate.Time.Format("2006-01-02")
		result.TransactionDate = &dateStr
	}
	if postedDate.Valid {
		dateStr := postedDate.Time.Format("2006-01-02")
		result.PostedDate = &dateStr
	}
	if cardNumber.Valid {
		result.CardNumber = &cardNumber.String
	}
	if categoryID.Valid {
		categoryStr := uuid.UUID(categoryID.Bytes).String()
		result.CategoryID = &categoryStr
	}

	return result
}

// convertTransactionFromActiveRow converts active transaction row to Transaction struct
func convertTransactionFromActiveRow(t generated.GetActiveTransactionsRow) Transaction {
	transaction := Transaction{
		ID:          uuid.UUID(t.ID.Bytes).String(),
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
		UpdatedAt:   t.UpdatedAt.Time,
	}

	// Convert amount
	if amountValue, err := t.Amount.Float64Value(); err == nil {
		transaction.Amount = amountValue.Float64
	}

	// Convert assigned_to array from UUIDs to names
	if len(t.AssignedTo) > 0 {
		names, err := convertUUIDArrayToNames(t.AssignedTo)
		if err != nil {
			log.Printf("Error converting UUIDs to names: %v", err)
			transaction.AssignedTo = []string{} // Initialize as empty array
		} else {
			transaction.AssignedTo = names
		}
	} else {
		transaction.AssignedTo = []string{} // Initialize as empty array
	}

	// Convert optional fields
	if t.DateUploaded.Valid {
		transaction.DateUploaded = t.DateUploaded.Time
	}
	if t.FileName.Valid {
		transaction.FileName = &t.FileName.String
	}
	if t.TransactionDate.Valid {
		dateStr := t.TransactionDate.Time.Format("2006-01-02")
		transaction.TransactionDate = &dateStr
	}
	if t.PostedDate.Valid {
		dateStr := t.PostedDate.Time.Format("2006-01-02")
		transaction.PostedDate = &dateStr
	}
	if t.CardNumber.Valid {
		transaction.CardNumber = &t.CardNumber.String
	}
	if t.CategoryID.Valid {
		categoryID := uuid.UUID(t.CategoryID.Bytes).String()
		transaction.CategoryID = &categoryID
	}

	return transaction
}

// convertTransactionFromArchivedRow converts archived transaction row to Transaction struct
func convertTransactionFromArchivedRow(t generated.GetArchivedTransactionsRow) Transaction {
	transaction := Transaction{
		ID:          uuid.UUID(t.ID.Bytes).String(),
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
		UpdatedAt:   t.UpdatedAt.Time,
	}

	// Convert amount
	if amountValue, err := t.Amount.Float64Value(); err == nil {
		transaction.Amount = amountValue.Float64
	}

	// Convert assigned_to array from UUIDs to names
	if len(t.AssignedTo) > 0 {
		names, err := convertUUIDArrayToNames(t.AssignedTo)
		if err != nil {
			log.Printf("Error converting UUIDs to names: %v", err)
		} else {
			transaction.AssignedTo = names
		}
	} else {
		transaction.AssignedTo = []string{}
	}

	// Convert optional fields
	if t.DateUploaded.Valid {
		transaction.DateUploaded = t.DateUploaded.Time
	}
	if t.FileName.Valid {
		transaction.FileName = &t.FileName.String
	}
	if t.TransactionDate.Valid {
		dateStr := t.TransactionDate.Time.Format("2006-01-02")
		transaction.TransactionDate = &dateStr
	}
	if t.PostedDate.Valid {
		dateStr := t.PostedDate.Time.Format("2006-01-02")
		transaction.PostedDate = &dateStr
	}
	if t.CardNumber.Valid {
		transaction.CardNumber = &t.CardNumber.String
	}
	if t.CategoryID.Valid {
		categoryID := uuid.UUID(t.CategoryID.Bytes).String()
		transaction.CategoryID = &categoryID
	}

	return transaction
}
