package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestGetCategories tests the GET /api/categories endpoint
func TestGetCategories(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should return default categories when no custom categories exist", func(t *testing.T) {
		resp := makeRequest("GET", "/api/categories", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var categories []Category
		assertNoError(t, parseJSONResponse(resp, &categories))

		// Should have the 12 default categories from initial migration
		if len(categories) != 12 {
			t.Errorf("Expected 12 default categories, got %d categories", len(categories))
		}
	})

	t.Run("should return list of categories when they exist", func(t *testing.T) {
		// Create test categories with unique names
		_, err := createTestCategory("Custom Food", "Restaurant and grocery expenses", "#FF5733")
		assertNoError(t, err)

		_, err = createTestCategory("Custom Transportation", "", "#33C1FF")
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/categories", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var categories []Category
		assertNoError(t, parseJSONResponse(resp, &categories))

		// Should have 12 default + 2 custom = 14 categories
		if len(categories) != 14 {
			t.Errorf("Expected 14 categories (12 default + 2 custom), got %d", len(categories))
		}

		// Verify our custom category data
		found := make(map[string]bool)
		for _, category := range categories {
			found[category.Name] = true
			if category.Name == "Custom Food" {
				if category.Description == nil || *category.Description != "Restaurant and grocery expenses" {
					t.Errorf("Expected Custom Food description to be 'Restaurant and grocery expenses', got %v", category.Description)
				}
				if category.Color == nil || *category.Color != "#FF5733" {
					t.Errorf("Expected Custom Food color to be '#FF5733', got %v", category.Color)
				}
			}
			if category.Name == "Custom Transportation" {
				if category.Description != nil {
					t.Errorf("Expected Custom Transportation description to be nil, got %v", category.Description)
				}
				if category.Color == nil || *category.Color != "#33C1FF" {
					t.Errorf("Expected Custom Transportation color to be '#33C1FF', got %v", category.Color)
				}
			}
		}

		if !found["Custom Food"] || !found["Custom Transportation"] {
			t.Error("Expected to find both Custom Food and Custom Transportation categories")
		}
	})
}

// TestCreateCategory tests the POST /api/categories endpoint
func TestCreateCategory(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should create category with all fields", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":        "Custom Entertainment",
			"description": "Movies, games, and fun activities",
			"color":       "#FF33E6",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusCreated, resp.Code)

		var category Category
		assertNoError(t, parseJSONResponse(resp, &category))

		if category.Name != "Custom Entertainment" {
			t.Errorf("Expected name 'Custom Entertainment', got '%s'", category.Name)
		}

		if category.Description == nil || *category.Description != "Movies, games, and fun activities" {
			t.Errorf("Expected description 'Movies, games, and fun activities', got %v", category.Description)
		}

		if category.Color == nil || *category.Color != "#FF33E6" {
			t.Errorf("Expected color '#FF33E6', got %v", category.Color)
		}

		if category.ID == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("should create category with minimal fields", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name": "Custom Utilities",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusCreated, resp.Code)

		var category Category
		assertNoError(t, parseJSONResponse(resp, &category))

		if category.Name != "Custom Utilities" {
			t.Errorf("Expected name 'Custom Utilities', got '%s'", category.Name)
		}

		if category.Description != nil {
			t.Errorf("Expected nil description, got %v", category.Description)
		}

		if category.Color != nil {
			t.Errorf("Expected nil color, got %v", category.Color)
		}
	})

	t.Run("should fail with empty name", func(t *testing.T) {
		// Clean data for this specific test
		if err := cleanupTestData(); err != nil {
			t.Fatalf("Failed to cleanup test data: %v", err)
		}

		requestBody := map[string]interface{}{
			"name":        "",
			"description": "Test category",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		// Should now return 400 Bad Request for empty name
		assertStatusCode(t, http.StatusBadRequest, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("should return 409 for duplicate name", func(t *testing.T) {
		// Create first category
		_, err := createTestCategory("Test Shopping", "Retail purchases", "#33FF57")
		assertNoError(t, err)

		// Try to create duplicate
		requestBody := map[string]interface{}{
			"name":        "Test Shopping",
			"description": "Online purchases",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/categories", bytes.NewBuffer(body))

		// Should now return 409 Conflict for duplicate name
		assertStatusCode(t, http.StatusConflict, resp.Code)

		var errorResp map[string]interface{}
		assertNoError(t, parseJSONResponse(resp, &errorResp))

		if errorResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("should fail with invalid JSON", func(t *testing.T) {
		resp := makeRequest("POST", "/api/categories", bytes.NewBufferString("invalid json"))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}

// TestUpdateCategory tests the PUT /api/categories/:id endpoint
func TestUpdateCategory(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should update existing category", func(t *testing.T) {
		// Create test category with unique name
		categoryID, err := createTestCategory("Custom Travel", "Trip expenses", "#FFD700")
		assertNoError(t, err)

		requestBody := map[string]interface{}{
			"name":        "Travel & Vacation",
			"description": "Vacation and business trips",
			"color":       "#FF8C00",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/categories/%s", categoryID), bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusOK, resp.Code)

		var category Category
		assertNoError(t, parseJSONResponse(resp, &category))

		if category.Name != "Travel & Vacation" {
			t.Errorf("Expected name 'Travel & Vacation', got '%s'", category.Name)
		}

		if category.Description == nil || *category.Description != "Vacation and business trips" {
			t.Errorf("Expected updated description, got %v", category.Description)
		}

		if category.Color == nil || *category.Color != "#FF8C00" {
			t.Errorf("Expected updated color, got %v", category.Color)
		}
	})

	t.Run("should return 404 for non-existent category ID", func(t *testing.T) {
		fakeID := "550e8400-e29b-41d4-a716-446655440000"

		requestBody := map[string]interface{}{
			"name": "Non-existent",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/categories/%s", fakeID), bytes.NewBuffer(body))

		// Should now return 404 Not Found for non-existent records
		assertStatusCode(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should fail with invalid UUID format", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name": "Test",
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", "/api/categories/invalid-uuid", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should fail with invalid JSON", func(t *testing.T) {
		categoryID, err := createTestCategory("Test", "", "")
		assertNoError(t, err)

		resp := makeRequest("PUT", fmt.Sprintf("/api/categories/%s", categoryID), bytes.NewBufferString("invalid json"))

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}

// TestDeleteCategory tests the DELETE /api/categories/:id endpoint
func TestDeleteCategory(t *testing.T) {
	// Clean data before test
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should delete existing category", func(t *testing.T) {
		// Create test category
		categoryID, err := createTestCategory("Medical Delete Test", "Healthcare expenses", "#FF69B4")
		assertNoError(t, err)

		resp := makeRequest("DELETE", fmt.Sprintf("/api/categories/%s", categoryID), nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		// Verify category is deleted by trying to get all categories
		resp = makeRequest("GET", "/api/categories", nil)
		assertStatusCode(t, http.StatusOK, resp.Code)

		var categories []Category
		assertNoError(t, parseJSONResponse(resp, &categories))

		// Should have at least the 12 default categories, and the custom one should be gone
		if len(categories) < 12 {
			t.Errorf("Expected at least 12 categories (default) after deletion, got %d", len(categories))
		}

		// Verify the custom category was deleted
		for _, category := range categories {
			if category.Name == "Medical Delete Test" {
				t.Error("Expected Medical Delete Test category to be deleted, but it still exists")
			}
		}
	})

	t.Run("should fail with non-existent category ID", func(t *testing.T) {
		fakeID := "550e8400-e29b-41d4-a716-446655440000"

		resp := makeRequest("DELETE", fmt.Sprintf("/api/categories/%s", fakeID), nil)

		assertStatusCode(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should fail with invalid UUID format", func(t *testing.T) {
		resp := makeRequest("DELETE", "/api/categories/invalid-uuid", nil)

		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should fail when category is assigned to transactions", func(t *testing.T) {
		// This test would require creating a transaction and assigning the category
		// We'll implement this when we have transaction tests
		t.Skip("Skipping until transaction category assignment is implemented")
	})
}
