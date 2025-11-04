package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// TestCreatePersonValidation tests proper validation for createPerson endpoint
func TestCreatePersonValidation(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should fail with empty name", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":  "",
			"email": "test@example.com",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/people", bytes.NewBuffer(body))

		// Should return 400 Bad Request for empty name
		assertStatusCode(t, http.StatusBadRequest, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("should fail with missing name", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"email": "test@example.com",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/people", bytes.NewBuffer(body))

		// Should return 400 Bad Request for missing name
		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should return 409 for duplicate name", func(t *testing.T) {
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

		// Should return 409 Conflict for duplicate name
		assertStatusCode(t, http.StatusConflict, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})
}

// TestCreateCategoryValidation tests proper validation for createCategory endpoint
func TestCreateCategoryValidation(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should fail with empty name", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":        "",
			"description": "Test category",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		// Should return 400 Bad Request for empty name
		assertStatusCode(t, http.StatusBadRequest, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("should fail with missing name", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"description": "Test category",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		// Should return 400 Bad Request for missing name
		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should return 409 for duplicate name", func(t *testing.T) {
		// Create a category with a unique name
		_, err := createTestCategory("Custom Shopping", "Retail purchases", "#33FF57")
		assertNoError(t, err)

		// Try to create duplicate
		requestBody := map[string]interface{}{
			"name":        "Custom Shopping",
			"description": "Online purchases",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		// Should return 409 Conflict for duplicate name
		assertStatusCode(t, http.StatusConflict, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("should validate color format", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":  "Test Category",
			"color": "invalid-color",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		// Should return 400 Bad Request for invalid color
		assertStatusCode(t, http.StatusBadRequest, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("should accept valid hex color", func(t *testing.T) {
		// Clean data for this specific test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		requestBody := map[string]interface{}{
			"name":  "Valid Color Category",
			"color": "#FF5733",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		// Should accept valid hex color
		assertStatusCode(t, http.StatusCreated, resp.Code)
	})
}
