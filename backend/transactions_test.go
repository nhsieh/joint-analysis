package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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
		// Create test category with unique name
		categoryID, err := createTestCategory("Custom Food", "Restaurant and grocery expenses", "#FF5733")
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

// createCSVFile creates a multipart form with a CSV file
func createCSVFile(t *testing.T, filename, content string) (*bytes.Buffer, string) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	_, err = part.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write to form file: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	return &body, writer.FormDataContentType()
}

// makeRequestWithCustomRequest helper to use existing router with custom request
func makeRequestWithCustomRequest(req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	testRouter.ServeHTTP(recorder, req)
	return recorder
}

// containsIgnoreCase helper to check string contains case-insensitively
func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

func TestUploadCSV(t *testing.T) {
	// Clean data before all CSV tests
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	// Valid CSV content with the expected 7-column format
	validCSV := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2023-01-01,2023-01-01,****1234,Coffee,Food,5.50,
2023-01-02,2023-01-02,****1234,Lunch,Food,12.00,
2023-01-03,2023-01-03,****1234,Gas,Transport,,30.00`

	t.Run("should upload valid CSV file", func(t *testing.T) {
		body, contentType := createCSVFile(t, "test.csv", validCSV)

		req, err := http.NewRequest("POST", "/api/upload-csv", body)
		assertNoError(t, err)

		req.Header.Set("Content-Type", contentType)

		resp := makeRequestWithCustomRequest(req)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &result)
		assertNoError(t, err)

		// Check that we got the expected number of transactions
		transactions, ok := result["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		if len(transactions) != 3 {
			t.Errorf("Expected 3 transactions, got %d", len(transactions))
		}

		// Check that skipped_rows is present and 0 for valid CSV
		skippedRows, ok := result["skipped_rows"].(float64)
		if !ok {
			t.Fatal("Expected skipped_rows field in response")
		}
		if skippedRows != 0 {
			t.Errorf("Expected 0 skipped rows, got %v", skippedRows)
		}
	})

	t.Run("should accept simple text file as CSV", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		body, contentType := createCSVFile(t, "test.txt", "not a csv")

		req, err := http.NewRequest("POST", "/api/upload-csv", body)
		assertNoError(t, err)

		req.Header.Set("Content-Type", contentType)

		resp := makeRequestWithCustomRequest(req)

		// The CSV parser accepts simple text as a CSV with 0 valid records
		assertStatusCode(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &result)
		assertNoError(t, err)

		transactions, ok := result["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		if len(transactions) != 0 {
			t.Errorf("Expected 0 transactions for non-CSV content, got %d", len(transactions))
		}
	})

	t.Run("should reject request with no file", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		req, err := http.NewRequest("POST", "/api/upload-csv", bytes.NewBuffer([]byte{}))
		assertNoError(t, err)

		resp := makeRequestWithCustomRequest(req)

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should handle invalid CSV content", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		invalidCSV := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2023-01-01,2023-01-01,****1234,Coffee,Food,invalid-amount,
2023-01-02,2023-01-02,****1234,Lunch,Food,12.00,`

		body, contentType := createCSVFile(t, "test.csv", invalidCSV)

		req, err := http.NewRequest("POST", "/api/upload-csv", body)
		assertNoError(t, err)

		req.Header.Set("Content-Type", contentType)

		resp := makeRequestWithCustomRequest(req)

		// Should still succeed but skip invalid rows
		assertStatusCode(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &result)
		assertNoError(t, err)

		transactions, ok := result["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		// Should only have 1 valid transaction (the lunch one)
		if len(transactions) != 1 {
			t.Errorf("Expected 1 valid transaction, got %d", len(transactions))
		}

		// Check that 1 row was skipped due to invalid amount
		skippedRows, ok := result["skipped_rows"].(float64)
		if !ok {
			t.Fatal("Expected skipped_rows field in response")
		}
		if skippedRows != 1 {
			t.Errorf("Expected 1 skipped row, got %v", skippedRows)
		}
	})

	t.Run("should handle empty CSV file", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		body, contentType := createCSVFile(t, "empty.csv", "")

		req, err := http.NewRequest("POST", "/api/upload-csv", body)
		assertNoError(t, err)

		req.Header.Set("Content-Type", contentType)

		resp := makeRequestWithCustomRequest(req)

		// Empty CSV is accepted and returns an empty transactions array
		assertStatusCode(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &result)
		assertNoError(t, err)

		transactions, ok := result["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		if len(transactions) != 0 {
			t.Errorf("Expected 0 transactions for empty CSV, got %d", len(transactions))
		}
	})

	t.Run("should handle CSV with only headers", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		headerOnlyCSV := "Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit"

		body, contentType := createCSVFile(t, "headers.csv", headerOnlyCSV)

		req, err := http.NewRequest("POST", "/api/upload-csv", body)
		assertNoError(t, err)

		req.Header.Set("Content-Type", contentType)

		resp := makeRequestWithCustomRequest(req)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &result)
		assertNoError(t, err)

		transactions, ok := result["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		if len(transactions) != 0 {
			t.Errorf("Expected 0 transactions for headers-only CSV, got %d", len(transactions))
		}
	})

	t.Run("should skip duplicate transactions", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		duplicateCSV := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2023-01-01,2023-01-01,****1234,Coffee,Food,5.50,
2023-01-02,2023-01-02,****1234,Lunch,Food,12.00,`

		// First upload
		body1, contentType1 := createCSVFile(t, "duplicate.csv", duplicateCSV)
		req1, err := http.NewRequest("POST", "/api/upload-csv", body1)
		assertNoError(t, err)
		req1.Header.Set("Content-Type", contentType1)

		resp1 := makeRequestWithCustomRequest(req1)
		assertStatusCode(t, http.StatusOK, resp1.Code)

		// Verify first upload worked
		var result1 map[string]interface{}
		err = json.Unmarshal(resp1.Body.Bytes(), &result1)
		assertNoError(t, err)

		transactions1, ok := result1["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		if len(transactions1) != 2 {
			t.Errorf("Expected 2 transactions from first upload, got %d", len(transactions1))
		}

		// Second upload (duplicates should be skipped, not rejected)
		body2, contentType2 := createCSVFile(t, "duplicate2.csv", duplicateCSV)
		req2, err := http.NewRequest("POST", "/api/upload-csv", body2)
		assertNoError(t, err)
		req2.Header.Set("Content-Type", contentType2)

		resp2 := makeRequestWithCustomRequest(req2)
		assertStatusCode(t, http.StatusOK, resp2.Code)

		var result map[string]interface{}
		err = json.Unmarshal(resp2.Body.Bytes(), &result)
		assertNoError(t, err)

		transactions, ok := result["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		// Should be 0 since all transactions are duplicates
		if len(transactions) != 0 {
			t.Errorf("Expected 0 transactions due to duplicates, got %d", len(transactions))
		}

		// Check that 2 rows were skipped due to duplicates
		skippedRows, ok := result["skipped_rows"].(float64)
		if !ok {
			t.Fatal("Expected skipped_rows field in response")
		}
		if skippedRows != 2 {
			t.Errorf("Expected 2 skipped rows, got %v", skippedRows)
		}
	})

	t.Run("should handle malformed multipart request", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		req, err := http.NewRequest("POST", "/api/upload-csv", bytes.NewBufferString("malformed multipart"))
		assertNoError(t, err)
		req.Header.Set("Content-Type", "multipart/form-data")

		resp := makeRequestWithCustomRequest(req)

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should handle missing required CSV columns", func(t *testing.T) {
		// Clean data before test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		missingColumnsCSV := `Transaction Date,Description
2023-01-01,Coffee
2023-01-02,Lunch`

		body, contentType := createCSVFile(t, "missing_columns.csv", missingColumnsCSV)

		req, err := http.NewRequest("POST", "/api/upload-csv", body)
		assertNoError(t, err)
		req.Header.Set("Content-Type", contentType)

		resp := makeRequestWithCustomRequest(req)

		// Should still return 200 but with 0 transactions since rows don't have enough columns
		assertStatusCode(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &result)
		assertNoError(t, err)

		transactions, ok := result["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions array in response")
		}
		if len(transactions) != 0 {
			t.Errorf("Expected 0 transactions for insufficient columns, got %d", len(transactions))
		}

		// Check that 2 rows were skipped due to insufficient columns
		skippedRows, ok := result["skipped_rows"].(float64)
		if !ok {
			t.Fatal("Expected skipped_rows field in response")
		}
		if skippedRows != 2 {
			t.Errorf("Expected 2 skipped rows, got %v", skippedRows)
		}
	})
}
