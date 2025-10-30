package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestGetPeople tests the GET /api/people endpoint
func TestGetPeople(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should return empty list when no people exist", func(t *testing.T) {
		resp := makeRequest("GET", "/api/people", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var people []Person
		assertNoError(t, parseJSONResponse(resp, &people))

		if len(people) != 0 {
			t.Errorf("Expected empty list, got %d people", len(people))
		}
	})

	t.Run("should return list of people when they exist", func(t *testing.T) {
		// Create test people
		_, err := createTestPerson("John Doe", "john@example.com")
		assertNoError(t, err)

		_, err = createTestPerson("Jane Smith", "")
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/people", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var people []Person
		assertNoError(t, parseJSONResponse(resp, &people))

		if len(people) != 2 {
			t.Errorf("Expected 2 people, got %d", len(people))
		}

		// Verify person data
		found := make(map[string]bool)
		for _, person := range people {
			found[person.Name] = true
			if person.Name == "John Doe" {
				if person.Email == nil || *person.Email != "john@example.com" {
					t.Errorf("Expected John Doe's email to be 'john@example.com', got %v", person.Email)
				}
			}
			if person.Name == "Jane Smith" {
				if person.Email != nil {
					t.Errorf("Expected Jane Smith's email to be nil, got %v", person.Email)
				}
			}
		}

		if !found["John Doe"] || !found["Jane Smith"] {
			t.Error("Expected to find both John Doe and Jane Smith")
		}
	})
}

// TestCreatePerson tests the POST /api/people endpoint
func TestCreatePerson(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should create person with valid data", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":  "Alice Johnson",
			"email": "alice@example.com",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/people", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusCreated, resp.Code)

		var person Person
		assertNoError(t, parseJSONResponse(resp, &person))

		if person.Name != "Alice Johnson" {
			t.Errorf("Expected name 'Alice Johnson', got '%s'", person.Name)
		}

		if person.Email == nil || *person.Email != "alice@example.com" {
			t.Errorf("Expected email 'alice@example.com', got %v", person.Email)
		}

		if person.ID == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("should create person without email", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name": "Bob Wilson",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/people", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusCreated, resp.Code)

		var person Person
		assertNoError(t, parseJSONResponse(resp, &person))

		if person.Name != "Bob Wilson" {
			t.Errorf("Expected name 'Bob Wilson', got '%s'", person.Name)
		}

		if person.Email != nil {
			t.Errorf("Expected nil email, got %v", person.Email)
		}
	})

	t.Run("should create person with empty name (current behavior)", func(t *testing.T) {
		// Clean data for this specific test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		requestBody := map[string]interface{}{
			"name":  "",
			"email": "test@example.com",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/people", bytes.NewBuffer(body))

		// Current implementation allows empty names
		assertStatusCode(t, http.StatusCreated, resp.Code)

		var person Person
		assertNoError(t, parseJSONResponse(resp, &person))

		if person.Name != "" {
			t.Errorf("Expected empty name, got '%s'", person.Name)
		}
	})

	t.Run("should create person with missing name (current behavior)", func(t *testing.T) {
		// Clean data for this specific test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		requestBody := map[string]interface{}{
			"email": "test@example.com",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/people", bytes.NewBuffer(body))

		// Current implementation allows missing names (defaults to empty string)
		assertStatusCode(t, http.StatusCreated, resp.Code)
	})

	t.Run("should return 500 for duplicate name (current behavior)", func(t *testing.T) {
		// Create first person
		_, err := createTestPerson("Charlie Brown", "charlie@example.com")
		assertNoError(t, err)

		// Try to create duplicate
		requestBody := map[string]interface{}{
			"name":  "Charlie Brown",
			"email": "charlie2@example.com",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/people", bytes.NewBuffer(body))

		// Current implementation returns 500 for database constraint violations
		assertStatusCode(t, http.StatusInternalServerError, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("should fail with invalid JSON", func(t *testing.T) {
		resp := makeRequest("POST", "/api/people", bytes.NewBufferString("invalid json"))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}

// TestDeletePerson tests the DELETE /api/people/:id endpoint
func TestDeletePerson(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should delete existing person", func(t *testing.T) {
		// Create test person
		personID, err := createTestPerson("David Miller", "david@example.com")
		assertNoError(t, err)

		resp := makeRequest("DELETE", fmt.Sprintf("/api/people/%s", personID), nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		// Verify person is deleted by trying to get all people
		resp = makeRequest("GET", "/api/people", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var people []Person
		assertNoError(t, parseJSONResponse(resp, &people))

		if len(people) != 0 {
			t.Errorf("Expected 0 people after deletion, got %d", len(people))
		}
	})

	t.Run("should fail with non-existent person ID", func(t *testing.T) {
		fakeID := "550e8400-e29b-41d4-a716-446655440000"

		resp := makeRequest("DELETE", fmt.Sprintf("/api/people/%s", fakeID), nil)

		assertStatusCode(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should fail with invalid UUID format", func(t *testing.T) {
		resp := makeRequest("DELETE", "/api/people/invalid-uuid", nil)

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should fail when person is assigned to transactions", func(t *testing.T) {
		// This test would require creating a transaction and assigning the person
		// We'll implement this when we have transaction tests
		t.Skip("Skipping until transaction assignment is implemented")
	})
}
