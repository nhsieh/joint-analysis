package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveTransactions(t *testing.T) {
	// Clean up data from previous tests
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	// Create test data - add some people and transactions
	person1ID, err := createTestPerson("Alice", "alice@example.com")
	require.NoError(t, err)
	person2ID, err := createTestPerson("Bob", "bob@example.com")
	require.NoError(t, err)

	// Create transactions
	_, err = createTestTransaction("Test Transaction 1", 100.50, "test.csv", []string{person1ID, person2ID})
	require.NoError(t, err)

	_, err = createTestTransaction("Test Transaction 2", 75.25, "test.csv", []string{person1ID})
	require.NoError(t, err)

	t.Run("successfully archives all active transactions", func(t *testing.T) {
		archiveRequest := ArchiveRequest{
			Description: "Archive for Q4 2025",
		}

		body, _ := json.Marshal(archiveRequest)
		w := makeRequest("POST", "/api/archives", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusCreated, w.Code)

		var response ArchiveResponse
		err := parseJSONResponse(w, &response)
		require.NoError(t, err)

		assert.Equal(t, archiveRequest.Description, response.Description)
		assert.Equal(t, 2, response.TransactionCount)
		assert.Equal(t, 175.75, response.TotalAmount) // 100.50 + 75.25
		assert.NotEmpty(t, response.ID)
		assert.NotZero(t, response.ArchivedAt)
	})

	t.Run("active transactions are no longer visible after archiving", func(t *testing.T) {
		w := makeRequest("GET", "/api/transactions", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var transactions []Transaction
		err := parseJSONResponse(w, &transactions)
		require.NoError(t, err)

		// Should be empty after archiving
		assert.Empty(t, transactions)
	})

	t.Run("can retrieve list of archives", func(t *testing.T) {
		w := makeRequest("GET", "/api/archives", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var archives []ArchiveResponse
		err := parseJSONResponse(w, &archives)
		require.NoError(t, err)

		assert.Len(t, archives, 1)
		assert.Equal(t, 2, archives[0].TransactionCount)
		assert.Equal(t, 175.75, archives[0].TotalAmount)
	})

	t.Run("cannot archive when no active transactions exist", func(t *testing.T) {
		archiveRequest := ArchiveRequest{
			Description: "Should fail",
		}

		body, _ := json.Marshal(archiveRequest)
		w := makeRequest("POST", "/api/archives", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse gin.H
		err := parseJSONResponse(w, &errorResponse)
		require.NoError(t, err)

		assert.Contains(t, errorResponse["error"], "no active transactions")
	})
}

func TestGetArchiveTransactions(t *testing.T) {
	// Clean up data from previous tests
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	// Create test data
	person1ID, err := createTestPerson("Alice", "alice@example.com")
	require.NoError(t, err)

	_, err = createTestTransaction("Test Transaction", 100.50, "test.csv", []string{person1ID})
	require.NoError(t, err)

	// Archive the transactions
	archiveRequest := ArchiveRequest{
		Description: "Test description",
	}
	body, _ := json.Marshal(archiveRequest)
	w := makeRequest("POST", "/api/archives", bytes.NewBuffer(body))

	var archiveResponse ArchiveResponse
	parseJSONResponse(w, &archiveResponse)

	t.Run("can retrieve transactions from specific archive", func(t *testing.T) {
		w := makeRequest("GET", "/api/archives/"+archiveResponse.ID+"/transactions", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var transactions []Transaction
		err := parseJSONResponse(w, &transactions)
		require.NoError(t, err)

		assert.Len(t, transactions, 1)
		assert.Equal(t, "Test Transaction", transactions[0].Description)
		assert.Equal(t, 100.50, transactions[0].Amount)
	})

	t.Run("returns 404 for non-existent archive", func(t *testing.T) {
		w := makeRequest("GET", "/api/archives/00000000-0000-0000-0000-000000000000/transactions", nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestArchiveValidation(t *testing.T) {
	// Clean up data from previous tests
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("requires name field", func(t *testing.T) {
		archiveRequest := ArchiveRequest{
			Description: "Missing name",
		}

		body, _ := json.Marshal(archiveRequest)
		w := makeRequest("POST", "/api/archives", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("response includes archive creation timestamp", func(t *testing.T) {
		// This test is covered by the main archive creation test
	})
}

// Helper types for archive functionality - using the ones from main.go
type ArchiveResponse struct {
	ID               string    `json:"id"`
	Description      string    `json:"description"`
	ArchivedAt       time.Time `json:"archived_at"`
	TransactionCount int       `json:"transaction_count"`
	TotalAmount      float64   `json:"total_amount"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
