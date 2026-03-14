package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"jointanalysis/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// createTestRule creates a test categorization rule and returns the ID
func createTestRule(matchValue, categoryID string, priority int32) (string, error) {
	catUUID, err := uuid.Parse(categoryID)
	if err != nil {
		return "", err
	}

	rule, err := testQueries.CreateRule(context.Background(), generated.CreateRuleParams{
		MatchValue: matchValue,
		CategoryID: pgtype.UUID{Bytes: catUUID, Valid: true},
		Priority:   priority,
	})
	if err != nil {
		return "", err
	}

	return uuid.UUID(rule.ID.Bytes).String(), nil
}

// TestGetRules tests the GET /api/rules endpoint
func TestGetRules(t *testing.T) {
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should return empty list when no rules exist", func(t *testing.T) {
		resp := makeRequest("GET", "/api/rules", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var rules []Rule
		assertNoError(t, parseJSONResponse(resp, &rules))

		if len(rules) != 0 {
			t.Errorf("Expected 0 rules, got %d", len(rules))
		}
	})

	t.Run("should return list of rules when they exist", func(t *testing.T) {
		catID, err := createTestCategory("Test Category A", "", "#FF0000")
		assertNoError(t, err)

		_, err = createTestRule("Trader Joe", catID, 0)
		assertNoError(t, err)

		_, err = createTestRule("Whole Foods", catID, 1)
		assertNoError(t, err)

		resp := makeRequest("GET", "/api/rules", nil)

		assertStatusCode(t, http.StatusOK, resp.Code)

		var rules []Rule
		assertNoError(t, parseJSONResponse(resp, &rules))

		if len(rules) != 2 {
			t.Errorf("Expected 2 rules, got %d", len(rules))
		}

		// Verify ordering by priority
		if rules[0].MatchValue != "Trader Joe" {
			t.Errorf("Expected first rule to be 'Trader Joe', got %q", rules[0].MatchValue)
		}
		if rules[1].MatchValue != "Whole Foods" {
			t.Errorf("Expected second rule to be 'Whole Foods', got %q", rules[1].MatchValue)
		}

		// Verify category name is included
		if rules[0].CategoryName != "Test Category A" {
			t.Errorf("Expected CategoryName 'Test Category A', got %q", rules[0].CategoryName)
		}
	})
}

// TestCreateRule tests the POST /api/rules endpoint
func TestCreateRule(t *testing.T) {
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should create rule with valid data", func(t *testing.T) {
		catID, err := createTestCategory("Groceries Test", "", "#00FF00")
		assertNoError(t, err)

		requestBody := map[string]interface{}{
			"match_value": "Trader Joes",
			"category_id": catID,
			"priority":    5,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/rules", bytes.NewBuffer(body))

		assertStatusCode(t, http.StatusCreated, resp.Code)

		var rule Rule
		assertNoError(t, parseJSONResponse(resp, &rule))

		if rule.ID == "" {
			t.Error("Expected rule to have an ID")
		}
		if rule.MatchValue != "Trader Joes" {
			t.Errorf("Expected match_value 'Trader Joes', got %q", rule.MatchValue)
		}
		if rule.CategoryID != catID {
			t.Errorf("Expected category_id %q, got %q", catID, rule.CategoryID)
		}
		if rule.Priority != 5 {
			t.Errorf("Expected priority 5, got %d", rule.Priority)
		}
		if rule.CategoryName != "Groceries Test" {
			t.Errorf("Expected category_name 'Groceries Test', got %q", rule.CategoryName)
		}
	})

	t.Run("should return 400 when match_value is missing", func(t *testing.T) {
		catID, err := createTestCategory("Cat For 400 Test", "", "#0000FF")
		assertNoError(t, err)

		requestBody := map[string]interface{}{
			"category_id": catID,
			"priority":    0,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/rules", bytes.NewBuffer(body))
		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should return 400 when category_id is missing", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"match_value": "SomeStore",
			"priority":    0,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/rules", bytes.NewBuffer(body))
		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should return 400 when category_id is not a valid UUID", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"match_value": "SomeStore",
			"category_id": "not-a-uuid",
			"priority":    0,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("POST", "/api/rules", bytes.NewBuffer(body))
		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}

// TestUpdateRule tests the PUT /api/rules/:id endpoint
func TestUpdateRule(t *testing.T) {
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should update rule successfully", func(t *testing.T) {
		catID, err := createTestCategory("Update Rule Cat", "", "#FF5733")
		assertNoError(t, err)

		ruleID, err := createTestRule("OldMatch", catID, 0)
		assertNoError(t, err)

		catID2, err := createTestCategory("Update Rule Cat 2", "", "#33FF57")
		assertNoError(t, err)

		requestBody := map[string]interface{}{
			"match_value": "NewMatch",
			"category_id": catID2,
			"priority":    10,
		}

		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", "/api/rules/"+ruleID, bytes.NewBuffer(body))
		assertStatusCode(t, http.StatusOK, resp.Code)

		var rule Rule
		assertNoError(t, parseJSONResponse(resp, &rule))

		if rule.MatchValue != "NewMatch" {
			t.Errorf("Expected match_value 'NewMatch', got %q", rule.MatchValue)
		}
		if rule.CategoryID != catID2 {
			t.Errorf("Expected category_id %q, got %q", catID2, rule.CategoryID)
		}
		if rule.Priority != 10 {
			t.Errorf("Expected priority 10, got %d", rule.Priority)
		}
	})

	t.Run("should return 400 for invalid UUID", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"match_value": "X",
			"category_id": "00000000-0000-0000-0000-000000000000",
			"priority":    0,
		}
		body, err := json.Marshal(requestBody)
		assertNoError(t, err)

		resp := makeRequest("PUT", "/api/rules/not-a-uuid", bytes.NewBuffer(body))
		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}

// TestDeleteRule tests the DELETE /api/rules/:id endpoint
func TestDeleteRule(t *testing.T) {
	if err := cleanupTestData(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}

	t.Run("should delete rule successfully", func(t *testing.T) {
		catID, err := createTestCategory("Delete Rule Cat", "", "#123456")
		assertNoError(t, err)

		ruleID, err := createTestRule("ToDelete", catID, 0)
		assertNoError(t, err)

		resp := makeRequest("DELETE", "/api/rules/"+ruleID, nil)
		assertStatusCode(t, http.StatusNoContent, resp.Code)

		// Verify it's gone
		listResp := makeRequest("GET", "/api/rules", nil)
		var rules []Rule
		assertNoError(t, parseJSONResponse(listResp, &rules))

		for _, r := range rules {
			if r.ID == ruleID {
				t.Error("Expected rule to be deleted but it still exists")
			}
		}
	})

	t.Run("should return 400 for invalid UUID", func(t *testing.T) {
		resp := makeRequest("DELETE", "/api/rules/not-a-uuid", nil)
		assertStatusCode(t, http.StatusBadRequest, resp.Code)
	})
}
