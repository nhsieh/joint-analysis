package main

import (
	"context"
	"log"
	"net/http"

	"jointanalysis/db/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Rule handler functions

// @Summary Get all rules
// @Description Retrieve all categorization rules ordered by priority
// @Tags rules
// @Produce json
// @Success 200 {array} Rule "List of categorization rules"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/rules [get]
func getRules(c *gin.Context) {
	dbRules, err := queries.GetRules(context.Background())
	if err != nil {
		log.Printf("Error fetching rules: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching rules"})
		return
	}

	rules := make([]Rule, 0, len(dbRules))
	for _, r := range dbRules {
		rules = append(rules, Rule{
			ID:           uuid.UUID(r.ID.Bytes).String(),
			MatchValue:   r.MatchValue,
			CategoryID:   uuid.UUID(r.CategoryID.Bytes).String(),
			CategoryName: r.CategoryName,
			Priority:     r.Priority,
			CreatedAt:    r.CreatedAt.Time,
			UpdatedAt:    r.UpdatedAt.Time,
		})
	}

	c.JSON(http.StatusOK, rules)
}

// @Summary Create rule
// @Description Create a new categorization rule
// @Tags rules
// @Accept json
// @Produce json
// @Param rule body Rule true "Rule data (match_value, category_id, priority required)"
// @Success 201 {object} Rule "Created rule"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/rules [post]
func createRule(c *gin.Context) {
	var req Rule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.MatchValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "match_value cannot be empty"})
		return
	}
	if req.CategoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id cannot be empty"})
		return
	}

	categoryUUID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id format"})
		return
	}

	params := generated.CreateRuleParams{
		MatchValue: req.MatchValue,
		CategoryID: pgtype.UUID{Bytes: categoryUUID, Valid: true},
		Priority:   req.Priority,
	}

	dbRule, err := queries.CreateRule(context.Background(), params)
	if err != nil {
		log.Printf("Error creating rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating rule"})
		return
	}

	// Fetch the full row with category name
	fullRule, err := queries.GetRuleByID(context.Background(), dbRule.ID)
	if err != nil {
		log.Printf("Error fetching created rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching created rule"})
		return
	}

	c.JSON(http.StatusCreated, Rule{
		ID:           uuid.UUID(fullRule.ID.Bytes).String(),
		MatchValue:   fullRule.MatchValue,
		CategoryID:   uuid.UUID(fullRule.CategoryID.Bytes).String(),
		CategoryName: fullRule.CategoryName,
		Priority:     fullRule.Priority,
		CreatedAt:    fullRule.CreatedAt.Time,
		UpdatedAt:    fullRule.UpdatedAt.Time,
	})
}

// @Summary Update rule
// @Description Update an existing categorization rule
// @Tags rules
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param rule body Rule true "Rule data"
// @Success 200 {object} Rule "Updated rule"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/rules/{id} [put]
func updateRule(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	var req Rule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.MatchValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "match_value cannot be empty"})
		return
	}
	if req.CategoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id cannot be empty"})
		return
	}

	categoryUUID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id format"})
		return
	}

	params := generated.UpdateRuleParams{
		ID:         pgtype.UUID{Bytes: parsedID, Valid: true},
		MatchValue: req.MatchValue,
		CategoryID: pgtype.UUID{Bytes: categoryUUID, Valid: true},
		Priority:   req.Priority,
	}

	dbRule, err := queries.UpdateRule(context.Background(), params)
	if err != nil {
		statusCode, msg := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": msg})
		return
	}

	// Fetch the full row with category name
	fullRule, err := queries.GetRuleByID(context.Background(), dbRule.ID)
	if err != nil {
		log.Printf("Error fetching updated rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching updated rule"})
		return
	}

	c.JSON(http.StatusOK, Rule{
		ID:           uuid.UUID(fullRule.ID.Bytes).String(),
		MatchValue:   fullRule.MatchValue,
		CategoryID:   uuid.UUID(fullRule.CategoryID.Bytes).String(),
		CategoryName: fullRule.CategoryName,
		Priority:     fullRule.Priority,
		CreatedAt:    fullRule.CreatedAt.Time,
		UpdatedAt:    fullRule.UpdatedAt.Time,
	})
}

// @Summary Delete rule
// @Description Delete a categorization rule
// @Tags rules
// @Param id path string true "Rule ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/rules/{id} [delete]
func deleteRule(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	if err := queries.DeleteRule(context.Background(), pgtype.UUID{Bytes: parsedID, Valid: true}); err != nil {
		log.Printf("Error deleting rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting rule"})
		return
	}

	c.Status(http.StatusNoContent)
}
