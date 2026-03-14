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

// Category handler functions

// @Summary Get all categories
// @Description Retrieve all categories as a nested tree (top-level categories with subcategories embedded)
// @Tags categories
// @Produce json
// @Success 200 {array} Category "Nested list of top-level categories with subcategories"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/categories [get]
func getCategories(c *gin.Context) {
	dbCategories, err := queries.GetCategories(context.Background())
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching categories"})
		return
	}

	// Build a map of all categories for quick lookup, and collect top-level ones
	categoryMap := make(map[string]*Category)
	var topLevel []*Category

	for _, dbCategory := range dbCategories {
		category := &Category{
			ID:            uuid.UUID(dbCategory.ID.Bytes).String(),
			Name:          dbCategory.Name,
			Subcategories: []Category{},
			CreatedAt:     dbCategory.CreatedAt.Time,
			UpdatedAt:     dbCategory.UpdatedAt.Time,
		}
		if dbCategory.Description.Valid {
			category.Description = &dbCategory.Description.String
		}
		if dbCategory.Color.Valid {
			category.Color = &dbCategory.Color.String
		}
		if dbCategory.ParentID.Valid {
			parentIDStr := uuid.UUID(dbCategory.ParentID.Bytes).String()
			category.ParentID = &parentIDStr
		}
		categoryMap[category.ID] = category
	}

	// Attach subcategories to parents and collect top-level categories
	for _, cat := range categoryMap {
		if cat.ParentID != nil {
			if parent, ok := categoryMap[*cat.ParentID]; ok {
				parent.Subcategories = append(parent.Subcategories, *cat)
			}
		} else {
			topLevel = append(topLevel, cat)
		}
	}

	// Convert to value slice for JSON response
	result := make([]Category, 0, len(topLevel))
	for _, cat := range topLevel {
		result = append(result, *cat)
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Create category
// @Description Create a new category in the system. Use parent_id to create a subcategory (max 2 levels).
// @Tags categories
// @Accept json
// @Produce json
// @Param category body Category true "Category data (name required; description, color, parent_id optional)"
// @Success 201 {object} Category "Created category"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 409 {object} map[string]interface{} "Category already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/categories [post]
func createCategory(c *gin.Context) {
	var category Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if err := validateName(category.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate color format if provided
	if category.Color != nil {
		if err := validateHexColor(*category.Color); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Handle parent_id validation (enforce 2-level max)
	var parentIDpg pgtype.UUID
	if category.ParentID != nil && *category.ParentID != "" {
		parentUUID, err := uuid.Parse(*category.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent_id format"})
			return
		}
		parentIDpg = pgtype.UUID{Bytes: parentUUID, Valid: true}

		// Validate parent exists
		parent, err := queries.GetCategoryByID(context.Background(), parentIDpg)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parent category not found"})
			return
		}

		// Enforce 2-level max: parent must itself be a top-level category
		if parent.ParentID.Valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot create subcategory of a subcategory (max 2 levels)"})
			return
		}
	}

	// Create parameters for the generated function
	params := generated.CreateCategoryParams{
		Name:     category.Name,
		ParentID: parentIDpg,
	}

	// Handle optional fields
	if category.Description != nil {
		params.Description = pgtype.Text{String: *category.Description, Valid: true}
	}
	if category.Color != nil {
		params.Color = pgtype.Text{String: *category.Color, Valid: true}
	}

	dbCategory, err := queries.CreateCategory(context.Background(), params)
	if err != nil {
		log.Printf("Error creating category: %v", err)
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Convert back to API type
	resultCategory := Category{
		ID:        uuid.UUID(dbCategory.ID.Bytes).String(),
		Name:      dbCategory.Name,
		CreatedAt: dbCategory.CreatedAt.Time,
		UpdatedAt: dbCategory.UpdatedAt.Time,
	}

	if dbCategory.Description.Valid {
		resultCategory.Description = &dbCategory.Description.String
	}
	if dbCategory.Color.Valid {
		resultCategory.Color = &dbCategory.Color.String
	}
	if dbCategory.ParentID.Valid {
		parentIDStr := uuid.UUID(dbCategory.ParentID.Bytes).String()
		resultCategory.ParentID = &parentIDStr
	}

	c.JSON(http.StatusCreated, resultCategory)
}

// @Summary Update category
// @Description Update an existing category. Cannot re-parent a category that has children.
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param category body Category true "Updated category data"
// @Success 200 {object} Category "Updated category"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Category not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/categories/{id} [put]
func updateCategory(c *gin.Context) {
	id := c.Param("id")
	var category Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Parse UUID from string
	categoryUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	categoryUUIDpg := pgtype.UUID{Bytes: categoryUUID, Valid: true}

	// Fetch existing category to check for children
	existing, err := queries.GetCategoryByID(context.Background(), categoryUUIDpg)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Block re-parenting a category that already has children
	if !existing.ParentID.Valid {
		// It's a top-level category — check if it has subcategories
		subs, err := queries.GetSubcategoriesByParent(context.Background(), categoryUUIDpg)
		if err == nil && len(subs) > 0 {
			// If request is trying to assign a parent_id, block it
			if category.ParentID != nil && *category.ParentID != "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot re-parent a category that already has subcategories"})
				return
			}
		}
	}

	// Create parameters for the generated function
	params := generated.UpdateCategoryParams{
		ID:   categoryUUIDpg,
		Name: category.Name,
	}

	// Handle optional fields
	if category.Description != nil {
		params.Description = pgtype.Text{String: *category.Description, Valid: true}
	}
	if category.Color != nil {
		params.Color = pgtype.Text{String: *category.Color, Valid: true}
	}

	dbCategory, err := queries.UpdateCategory(context.Background(), params)
	if err != nil {
		log.Printf("Error updating category: %v", err)
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Convert back to API type
	resultCategory := Category{
		ID:        uuid.UUID(dbCategory.ID.Bytes).String(),
		Name:      dbCategory.Name,
		CreatedAt: dbCategory.CreatedAt.Time,
		UpdatedAt: dbCategory.UpdatedAt.Time,
	}

	if dbCategory.Description.Valid {
		resultCategory.Description = &dbCategory.Description.String
	}
	if dbCategory.Color.Valid {
		resultCategory.Color = &dbCategory.Color.String
	}
	if dbCategory.ParentID.Valid {
		parentIDStr := uuid.UUID(dbCategory.ParentID.Bytes).String()
		resultCategory.ParentID = &parentIDStr
	}

	c.JSON(http.StatusOK, resultCategory)
}

// @Summary Delete category
// @Description Delete a specific category by ID
// @Tags categories
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} map[string]interface{} "Category deleted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Category not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/categories/{id} [delete]
func deleteCategory(c *gin.Context) {
	id := c.Param("id")

	// Parse UUID from string
	categoryUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Create pgtype.UUID for the queries
	categoryUUIDpg := pgtype.UUID{Bytes: categoryUUID, Valid: true}

	// First, get the category to ensure it exists
	_, err = queries.GetCategoryByID(context.Background(), categoryUUIDpg)
	if err != nil {
		log.Printf("Error finding category: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Delete the category
	err = queries.DeleteCategory(context.Background(), categoryUUIDpg)
	if err != nil {
		log.Printf("Error deleting category: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}
