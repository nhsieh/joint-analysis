package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// createTestTransaction creates a test transaction and returns the ID
func createTestTransaction(description string, amount float64, fileName string, assignedTo []string) (string, error) {
	// Convert amount to decimal
	amountDecimal := pgtype.Numeric{Int: nil, Exp: 0, NaN: false, Valid: true}
	amountDecimal.Scan(fmt.Sprintf("%.2f", amount))

	// Convert assigned_to to UUIDs array if provided
	var assignedUUIDs []pgtype.UUID
	if len(assignedTo) > 0 {
		for _, personID := range assignedTo {
			if personUUID, err := parseUUID(personID); err == nil {
				assignedUUIDs = append(assignedUUIDs, pgtype.UUID{Bytes: personUUID, Valid: true})
			}
		}
	}

	// Create transaction using raw SQL since we don't have a CreateTransaction query
	query := `
		INSERT INTO transactions (description, amount, assigned_to, file_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id pgtype.UUID
	err := testDB.QueryRow(context.Background(), query, description, amountDecimal, assignedUUIDs, fileName).Scan(&id)
	if err != nil {
		return "", err
	}

	return uuid.UUID(id.Bytes).String(), nil
}

// parseUUID helper function
func parseUUID(uuidStr string) ([16]byte, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return [16]byte{}, err
	}
	return parsed, nil
}

// TestGetTransactions tests the GET /api/transactions endpoint
func TestGetTransactions(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should return empty list when no transactions exist", func(t *testing.T) {
		resp := makeRequest("GET", "/api/transactions", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var transactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &transactions))

		if len(transactions) != 0 {
			t.Errorf("Expected empty list, got %d transactions", len(transactions))
		}
	})

	t.Run("should return list of transactions when they exist", func(t *testing.T) {
		// Create test transactions
		_, err := createTestTransaction("Grocery Shopping", 125.50, "test.csv", nil)
		assertNoError(t, err)

		_, err = createTestTransaction("Gas Station", 45.00, "test.csv", nil)
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/transactions", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var transactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &transactions))

		if len(transactions) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(transactions))
		}

		// Verify transaction data
		found := make(map[string]bool)
		for _, transaction := range transactions {
			found[transaction.Description] = true
			if transaction.Description == "Grocery Shopping" {
				if transaction.Amount != 125.50 {
					t.Errorf("Expected amount 125.50, got %f", transaction.Amount)
				}
			}
			if transaction.Description == "Gas Station" {
				if transaction.Amount != 45.00 {
					t.Errorf("Expected amount 45.00, got %f", transaction.Amount)
				}
			}
		}

		if !found["Grocery Shopping"] || !found["Gas Station"] {
			t.Error("Expected to find both transactions")
		}
	})
}

// TestClearAllTransactions tests the DELETE /api/transactions endpoint
func TestClearAllTransactions(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should clear all transactions successfully", func(t *testing.T) {
		// Create test transactions
		_, err := createTestTransaction("Test Transaction 1", 100.00, "test.csv", nil)
		assertNoError(t, err)

		_, err = createTestTransaction("Test Transaction 2", 200.00, "test.csv", nil)
		assertNoError(t, err)

		// Verify transactions exist
		resp := makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var transactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &transactions))

		if len(transactions) != 2 {
			t.Errorf("Expected 2 transactions before clearing, got %d", len(transactions))
		}

		// Clear all transactions
		resp = makeRequest("DELETE", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var response map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &response))

		if response["message"] == nil {
			t.Error("Expected success message in response")
		}

		// Verify transactions are cleared
		resp = makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		assertNoError(t, parseJSONResponse(resp, &transactions))

		if len(transactions) != 0 {
			t.Errorf("Expected 0 transactions after clearing, got %d", len(transactions))
		}
	})

	t.Run("should handle clearing when no transactions exist", func(t *testing.T) {
		// Ensure no transactions exist
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		resp := makeRequest("DELETE", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var response map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &response))

		if response["message"] == nil {
			t.Error("Expected success message in response")
		}
	})
}

// TestAssignTransaction tests the PUT /api/transactions/:id/assign endpoint
func TestAssignTransaction(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should assign people to transaction successfully", func(t *testing.T) {
		// Create test people
		person1ID, err := createTestPerson("John Doe", "john@example.com")
		assertNoError(t, err)

		person2ID, err := createTestPerson("Jane Smith", "jane@example.com")
		assertNoError(t, err)

		// Create test transaction
		transactionID, err := createTestTransaction("Dinner at Restaurant", 85.50, "test.csv", nil)
		assertNoError(t, err)

		// Assign people to transaction
		requestBody := map[string]interface{}{
			"assigned_to": []string{person1ID, person2ID},
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/assign", transactionID), bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusOK, resp.Code)

		var transaction Transaction
		assertNoError(t, parseJSONResponse(resp, &transaction))

		if len(transaction.AssignedTo) != 2 {
			t.Errorf("Expected 2 assigned people, got %d", len(transaction.AssignedTo))
		}

		// Verify the assigned people names (API returns names, not UUIDs)
		assignedMap := make(map[string]bool)
		for _, assignedName := range transaction.AssignedTo {
			assignedMap[assignedName] = true
		}

		if !assignedMap["John Doe"] || !assignedMap["Jane Smith"] {
			t.Error("Expected both people names to be assigned to transaction")
		}
	})

	t.Run("should handle empty assignment array", func(t *testing.T) {
		// Create test transaction
		transactionID, err := createTestTransaction("Solo Purchase", 25.00, "test.csv", nil)
		assertNoError(t, err)

		// Assign empty array
		requestBody := map[string]interface{}{
			"assigned_to": []string{},
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/assign", transactionID), bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusOK, resp.Code)

		var transaction Transaction
		assertNoError(t, parseJSONResponse(resp, &transaction))

		if len(transaction.AssignedTo) != 0 {
			t.Errorf("Expected 0 assigned people, got %d", len(transaction.AssignedTo))
		}
	})

	t.Run("should fail with invalid transaction ID", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"assigned_to": []string{},
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", "/api/transactions/invalid-uuid/assign", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should fail with non-existent transaction ID", func(t *testing.T) {
		fakeID := "550e8400-e29b-41d4-a716-446655440000"

		requestBody := map[string]interface{}{
			"assigned_to": []string{},
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/assign", fakeID), bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should fail with invalid JSON", func(t *testing.T) {
		transactionID, err := createTestTransaction("Test Transaction", 50.00, "test.csv", nil)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/assign", transactionID), bytes.NewBufferString("invalid json"))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}

// TestUpdateTransactionCategory tests the PUT /api/transactions/:id/category endpoint
func TestUpdateTransactionCategory(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should update transaction category successfully", func(t *testing.T) {
		// Create test category
		categoryID, err := createTestCategory("Food", "Restaurant and grocery expenses", "#FF5733")
		assertNoError(t, err)

		// Create test transaction
		transactionID, err := createTestTransaction("Restaurant Bill", 75.00, "test.csv", nil)
		assertNoError(t, err)

		// Update transaction category
		requestBody := map[string]interface{}{
			"category_id": categoryID,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/category", transactionID), bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusOK, resp.Code)

		var transaction Transaction
		assertNoError(t, parseJSONResponse(resp, &transaction))

		if transaction.CategoryID == nil || *transaction.CategoryID != categoryID {
			t.Errorf("Expected category ID %s, got %v", categoryID, transaction.CategoryID)
		}
	})

	t.Run("should clear transaction category with null", func(t *testing.T) {
		// Create test transaction
		transactionID, err := createTestTransaction("Uncategorized Purchase", 30.00, "test.csv", nil)
		assertNoError(t, err)

		// Clear category by setting to null
		requestBody := map[string]interface{}{
			"category_id": nil,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/category", transactionID), bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusOK, resp.Code)

		var transaction Transaction
		assertNoError(t, parseJSONResponse(resp, &transaction))

		if transaction.CategoryID != nil {
			t.Errorf("Expected nil category ID, got %v", transaction.CategoryID)
		}
	})

	t.Run("should fail with invalid transaction ID", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"category_id": nil,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", "/api/transactions/invalid-uuid/category", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should fail with non-existent transaction ID", func(t *testing.T) {
		fakeID := "550e8400-e29b-41d4-a716-446655440000"

		requestBody := map[string]interface{}{
			"category_id": nil,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/category", fakeID), bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should fail with invalid JSON", func(t *testing.T) {
		transactionID, err := createTestTransaction("Test Transaction", 50.00, "test.csv", nil)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/transactions/%s/category", transactionID), bytes.NewBufferString("invalid json"))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}
