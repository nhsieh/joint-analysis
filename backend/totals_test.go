package main

import (
	"net/http"
	"testing"
)

// TestGetTotals tests the GET /api/totals endpoint
func TestGetTotals(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should return empty list when no transactions exist", func(t *testing.T) {
		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 0 {
			t.Errorf("Expected empty list, got %d totals", len(totals))
		}
	})

	t.Run("should return empty list when transactions have no assignments", func(t *testing.T) {
		// Create unassigned transactions
		_, err := createTestTransaction("Unassigned Transaction 1", 100.00, "test.csv", nil)
		assertNoError(t, err)
		
		_, err = createTestTransaction("Unassigned Transaction 2", 50.00, "test.csv", nil)
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 0 {
			t.Errorf("Expected empty list for unassigned transactions, got %d totals", len(totals))
		}
	})

	t.Run("should calculate totals for single person assignments", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		// Create test people
		person1ID, err := createTestPerson("Alice Johnson", "alice@example.com")
		assertNoError(t, err)
		
		person2ID, err := createTestPerson("Bob Smith", "bob@example.com")
		assertNoError(t, err)

		// Create transactions assigned to single people
		_, err = createTestTransaction("Alice's Lunch", 25.50, "test.csv", []string{person1ID})
		assertNoError(t, err)
		
		_, err = createTestTransaction("Alice's Coffee", 4.50, "test.csv", []string{person1ID})
		assertNoError(t, err)
		
		_, err = createTestTransaction("Bob's Gas", 40.00, "test.csv", []string{person2ID})
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 2 {
			t.Errorf("Expected 2 people in totals, got %d", len(totals))
		}
		
		// Verify totals by person name (returned in alphabetical order)
		expectedTotals := map[string]float64{
			"Alice Johnson": 30.00, // 25.50 + 4.50
			"Bob Smith":     40.00,
		}
		
		for _, total := range totals {
			expectedAmount, exists := expectedTotals[total.Person]
			if !exists {
				t.Errorf("Unexpected person in totals: %s", total.Person)
				continue
			}
			
			if total.Total != expectedAmount {
				t.Errorf("Expected total %f for %s, got %f", expectedAmount, total.Person, total.Total)
			}
		}
	})

	t.Run("should split costs for shared transactions", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		// Create test people
		person1ID, err := createTestPerson("Charlie Brown", "charlie@example.com")
		assertNoError(t, err)
		
		person2ID, err := createTestPerson("Diana Prince", "diana@example.com")
		assertNoError(t, err)

		// Create shared transaction (should be split 50/50)
		_, err = createTestTransaction("Shared Dinner", 60.00, "test.csv", []string{person1ID, person2ID})
		assertNoError(t, err)
		
		// Create another shared transaction
		_, err = createTestTransaction("Shared Groceries", 80.00, "test.csv", []string{person1ID, person2ID})
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 2 {
			t.Errorf("Expected 2 people in totals, got %d", len(totals))
		}
		
		// Each person should get half of each shared transaction
		// Charlie: (60/2) + (80/2) = 30 + 40 = 70
		// Diana: (60/2) + (80/2) = 30 + 40 = 70
		expectedTotals := map[string]float64{
			"Charlie Brown": 70.00,
			"Diana Prince":  70.00,
		}
		
		for _, total := range totals {
			expectedAmount, exists := expectedTotals[total.Person]
			if !exists {
				t.Errorf("Unexpected person in totals: %s", total.Person)
				continue
			}
			
			if total.Total != expectedAmount {
				t.Errorf("Expected total %f for %s, got %f", expectedAmount, total.Person, total.Total)
			}
		}
	})

	t.Run("should handle three-way splits correctly", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		// Create test people
		person1ID, err := createTestPerson("Eve Adams", "eve@example.com")
		assertNoError(t, err)
		
		person2ID, err := createTestPerson("Frank Wilson", "frank@example.com")
		assertNoError(t, err)
		
		person3ID, err := createTestPerson("Grace Lee", "grace@example.com")
		assertNoError(t, err)

		// Create three-way shared transaction
		_, err = createTestTransaction("Group Trip", 150.00, "test.csv", []string{person1ID, person2ID, person3ID})
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 3 {
			t.Errorf("Expected 3 people in totals, got %d", len(totals))
		}
		
		// Each person should get 150/3 = 50.00
		expectedTotals := map[string]float64{
			"Eve Adams":     50.00,
			"Frank Wilson":  50.00,
			"Grace Lee":     50.00,
		}
		
		for _, total := range totals {
			expectedAmount, exists := expectedTotals[total.Person]
			if !exists {
				t.Errorf("Unexpected person in totals: %s", total.Person)
				continue
			}
			
			if total.Total != expectedAmount {
				t.Errorf("Expected total %f for %s, got %f", expectedAmount, total.Person, total.Total)
			}
		}
	})

	t.Run("should handle mixed individual and shared transactions", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		// Create test people
		person1ID, err := createTestPerson("Henry Ford", "henry@example.com")
		assertNoError(t, err)
		
		person2ID, err := createTestPerson("Irene Jones", "irene@example.com")
		assertNoError(t, err)

		// Henry's individual transaction
		_, err = createTestTransaction("Henry's Books", 30.00, "test.csv", []string{person1ID})
		assertNoError(t, err)
		
		// Irene's individual transaction
		_, err = createTestTransaction("Irene's Supplies", 20.00, "test.csv", []string{person2ID})
		assertNoError(t, err)
		
		// Shared transaction
		_, err = createTestTransaction("Shared Lunch", 40.00, "test.csv", []string{person1ID, person2ID})
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 2 {
			t.Errorf("Expected 2 people in totals, got %d", len(totals))
		}
		
		// Henry: 30 (individual) + 20 (40/2 shared) = 50
		// Irene: 20 (individual) + 20 (40/2 shared) = 40
		expectedTotals := map[string]float64{
			"Henry Ford":   50.00,
			"Irene Jones":  40.00,
		}
		
		for _, total := range totals {
			expectedAmount, exists := expectedTotals[total.Person]
			if !exists {
				t.Errorf("Unexpected person in totals: %s", total.Person)
				continue
			}
			
			if total.Total != expectedAmount {
				t.Errorf("Expected total %f for %s, got %f", expectedAmount, total.Person, total.Total)
			}
		}
	})

	t.Run("should handle decimal amounts correctly", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		// Create test people
		person1ID, err := createTestPerson("Jack Miller", "jack@example.com")
		assertNoError(t, err)
		
		person2ID, err := createTestPerson("Kate Brown", "kate@example.com")
		assertNoError(t, err)

		// Transaction that doesn't divide evenly
		_, err = createTestTransaction("Odd Amount", 33.33, "test.csv", []string{person1ID, person2ID})
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 2 {
			t.Errorf("Expected 2 people in totals, got %d", len(totals))
		}
		
		// Each person should get 33.33/2 = 16.665, which should be handled appropriately
		for _, total := range totals {
			if total.Total < 16.66 || total.Total > 16.67 {
				t.Errorf("Expected total around 16.665 for %s, got %f", total.Person, total.Total)
			}
		}
	})

	t.Run("should return totals in alphabetical order by person name", func(t *testing.T) {
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		// Create test people in non-alphabetical order
		person1ID, err := createTestPerson("Zoe Taylor", "zoe@example.com")
		assertNoError(t, err)
		
		person2ID, err := createTestPerson("Adam Clark", "adam@example.com")
		assertNoError(t, err)
		
		person3ID, err := createTestPerson("Mary Johnson", "mary@example.com")
		assertNoError(t, err)

		// Create transactions
		_, err = createTestTransaction("Zoe's Purchase", 10.00, "test.csv", []string{person1ID})
		assertNoError(t, err)
		
		_, err = createTestTransaction("Adam's Purchase", 20.00, "test.csv", []string{person2ID})
		assertNoError(t, err)
		
		_, err = createTestTransaction("Mary's Purchase", 30.00, "test.csv", []string{person3ID})
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/totals", nil)
		
		assertStatusCode(t, http.StatusOK, resp.Code)
		
		var totals []Total
		assertNoError(t, parseJSONResponse(resp, &totals))
		
		if len(totals) != 3 {
			t.Errorf("Expected 3 people in totals, got %d", len(totals))
		}
		
		// Should be in alphabetical order: Adam, Mary, Zoe
		expectedOrder := []string{"Adam Clark", "Mary Johnson", "Zoe Taylor"}
		for i, total := range totals {
			if total.Person != expectedOrder[i] {
				t.Errorf("Expected person %s at position %d, got %s", expectedOrder[i], i, total.Person)
			}
		}
	})
}