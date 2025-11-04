package main

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// createCSVFile creates a multipart form with CSV content
func createCSVFile(content string, filename string) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create form file
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}

	// Write CSV content
	_, err = fileWriter.Write([]byte(content))
	if err != nil {
		panic(err)
	}

	writer.Close()
	return &buf, writer.FormDataContentType()
}

// TestUploadCSV tests the POST /api/upload-csv endpoint
func TestUploadCSV(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should upload valid CSV successfully", func(t *testing.T) {
		csvContent := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2025-10-17,2025-10-20,1111,VALERO GAS STATION,Gas/Automotive,26.45,
2025-10-20,2025-10-20,2222,REI CLASSES & EVENTS,Other Travel,25.00,
2025-10-17,2025-10-18,2222,COSTCO WHOLESALE,Merchandise,27.97,`

		body, contentType := createCSVFile(csvContent, "test_transactions.csv")

		// Create request
		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		// Execute request
		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		assertNoError(t, parseJSONResponse(w, &response))

		// Verify response structure
		if response["message"] == nil {
			t.Error("Expected success message in response")
		}

		if response["transactions"] == nil {
			t.Fatal("Expected transactions array in response")
		}

		transactions, ok := response["transactions"].([]interface{})
		if !ok {
			t.Fatal("Expected transactions to be an array")
		}

		if len(transactions) != 3 {
			t.Errorf("Expected 3 transactions, got %d", len(transactions))
		}

		// Verify transactions were created in database
		resp := makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var dbTransactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &dbTransactions))

		if len(dbTransactions) != 3 {
			t.Errorf("Expected 3 transactions in database, got %d", len(dbTransactions))
		}

		// Verify specific transaction data
		found := make(map[string]bool)
		for _, transaction := range dbTransactions {
			found[transaction.Description] = true
			if transaction.Description == "VALERO GAS STATION" {
				if transaction.Amount != 26.45 {
					t.Errorf("Expected amount 26.45, got %f", transaction.Amount)
				}
				if transaction.FileName == nil {
					t.Errorf("Expected filename test_transactions.csv, got nil")
				} else if *transaction.FileName != "test_transactions.csv" {
					t.Errorf("Expected filename test_transactions.csv, got %s", *transaction.FileName)
				}
			}
		}

		if !found["VALERO GAS STATION"] || !found["REI CLASSES & EVENTS"] || !found["COSTCO WHOLESALE"] {
			t.Error("Expected to find all uploaded transactions")
		}
	})

	t.Run("should handle CSV without header row", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		csvContent := `2025-10-17,2025-10-20,1111,NO HEADER ROW,Gas/Automotive,15.00,
2025-10-20,2025-10-20,2222,SECOND TRANSACTION,Other Travel,20.00,`

		body, contentType := createCSVFile(csvContent, "no_header.csv")

		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusOK, w.Code)

		// Verify transactions were created
		resp := makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var dbTransactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &dbTransactions))

		if len(dbTransactions) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(dbTransactions))
		}
	})

	t.Run("should handle credit amounts correctly", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		csvContent := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2025-10-17,2025-10-20,1111,DEBIT TRANSACTION,Gas/Automotive,50.00,
2025-10-20,2025-10-20,2222,CREDIT TRANSACTION,Other Travel,,75.00`

		body, contentType := createCSVFile(csvContent, "credit_test.csv")

		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusOK, w.Code)

		// Verify both transactions were created with correct amounts
		resp := makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var dbTransactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &dbTransactions))

		if len(dbTransactions) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(dbTransactions))
		}

		for _, transaction := range dbTransactions {
			if transaction.Description == "DEBIT TRANSACTION" && transaction.Amount != 50.00 {
				t.Errorf("Expected debit amount 50.00, got %f", transaction.Amount)
			}
			if transaction.Description == "CREDIT TRANSACTION" && transaction.Amount != 75.00 {
				t.Errorf("Expected credit amount 75.00, got %f", transaction.Amount)
			}
		}
	})

	t.Run("should skip rows with no amounts", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		csvContent := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2025-10-17,2025-10-20,1111,VALID TRANSACTION,Gas/Automotive,25.00,
2025-10-20,2025-10-20,2222,NO AMOUNT TRANSACTION,Other Travel,,
2025-10-18,2025-10-19,3333,ANOTHER VALID,Merchandise,30.00,`

		body, contentType := createCSVFile(csvContent, "no_amount_rows.csv")

		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusOK, w.Code)

		// Should only create valid transactions, skipping rows with no amounts
		resp := makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var dbTransactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &dbTransactions))

		if len(dbTransactions) != 2 {
			t.Errorf("Expected 2 valid transactions (skipping no-amount row), got %d", len(dbTransactions))
		}
	})

	t.Run("should fail with no file uploaded", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/upload-csv", nil)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		assertNoError(t, parseJSONResponse(w, &response))

		if !strings.Contains(fmt.Sprintf("%v", response["error"]), "No file uploaded") {
			t.Error("Expected 'No file uploaded' error message")
		}
	})

	t.Run("should fail with invalid CSV content", func(t *testing.T) {
		// Create invalid CSV (unclosed quotes)
		invalidCSVContent := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2025-10-17,2025-10-20,1111,"UNCLOSED QUOTE FIELD,Gas/Automotive,25.00,`

		body, contentType := createCSVFile(invalidCSVContent, "invalid.csv")

		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		assertNoError(t, parseJSONResponse(w, &response))

		if !strings.Contains(fmt.Sprintf("%v", response["error"]), "Error reading CSV file") {
			t.Error("Expected 'Error reading CSV file' error message")
		}
	})

	t.Run("should handle empty CSV file", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		emptyCSVContent := ""

		body, contentType := createCSVFile(emptyCSVContent, "empty.csv")

		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		assertNoError(t, parseJSONResponse(w, &response))

		// Should return success with empty transactions array
		transactions, ok := response["transactions"].([]interface{})
		if !ok {
			t.Fatalf("Expected transactions to be an array, got: %T", response["transactions"])
		}

		if len(transactions) != 0 {
			t.Errorf("Expected 0 transactions for empty file, got %d", len(transactions))
		}
	})

	t.Run("should handle malformed amounts gracefully", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		csvContent := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2025-10-17,2025-10-20,1111,VALID TRANSACTION,Gas/Automotive,25.00,
2025-10-20,2025-10-20,2222,INVALID AMOUNT,Other Travel,NOT_A_NUMBER,
2025-10-18,2025-10-19,3333,ANOTHER VALID,Merchandise,15.50,`

		body, contentType := createCSVFile(csvContent, "malformed_amounts.csv")

		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusOK, w.Code)

		// Should skip malformed amounts and continue processing
		resp := makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var dbTransactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &dbTransactions))

		if len(dbTransactions) != 2 {
			t.Errorf("Expected 2 valid transactions (skipping malformed amount), got %d", len(dbTransactions))
		}

		// Verify the valid transactions were processed
		validAmounts := []float64{25.00, 15.50}
		foundAmounts := make(map[float64]bool)
		for _, transaction := range dbTransactions {
			foundAmounts[transaction.Amount] = true
		}

		for _, amount := range validAmounts {
			if !foundAmounts[amount] {
				t.Errorf("Expected to find transaction with amount %f", amount)
			}
		}
	})

	t.Run("should prevent duplicate transactions", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		csvContent := `Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2025-10-17,2025-10-20,1111,DUPLICATE TEST TRANSACTION,Gas/Automotive,25.00,
2025-10-20,2025-10-20,2222,UNIQUE TRANSACTION,Other Travel,30.00,`

		// Upload CSV first time
		body, contentType := createCSVFile(csvContent, "duplicate_test.csv")

		req := httptest.NewRequest("POST", "/api/upload-csv", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assertStatusCode(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		assertNoError(t, parseJSONResponse(w, &response))

		transactions, ok := response["transactions"].([]interface{})
		if !ok {
			t.Fatalf("Expected transactions to be an array")
		}

		if len(transactions) != 2 {
			t.Errorf("Expected 2 transactions on first upload, got %d", len(transactions))
		}

		// Upload same CSV second time - should prevent duplicates
		body2, contentType2 := createCSVFile(csvContent, "duplicate_test.csv")

		req2 := httptest.NewRequest("POST", "/api/upload-csv", body2)
		req2.Header.Set("Content-Type", contentType2)
		w2 := httptest.NewRecorder()

		testRouter.ServeHTTP(w2, req2)

		assertStatusCode(t, http.StatusOK, w2.Code)

		var response2 map[string]interface{}
		assertNoError(t, parseJSONResponse(w2, &response2))

		transactions2, ok := response2["transactions"].([]interface{})
		if !ok {
			t.Fatalf("Expected transactions to be an array")
		}

		if len(transactions2) != 0 {
			t.Errorf("Expected 0 new transactions on duplicate upload, got %d", len(transactions2))
		}

		// Verify total count in database is still 2
		resp := makeRequest("GET", "/api/transactions", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var dbTransactions []Transaction
		assertNoError(t, parseJSONResponse(resp, &dbTransactions))

		if len(dbTransactions) != 2 {
			t.Errorf("Expected total of 2 transactions in database after duplicate upload, got %d", len(dbTransactions))
		}
	})
}
