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
// @Description Retrieve all categories from the database
// @Tags categories
// @Produce json
// @Success 200 {array} Category "List of categories"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/categories [get]
func getCategories(c *gin.Context) {
	dbCategories, err := queries.GetCategories(context.Background())
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching categories"})
		return
	}

	var categories []Category
	for _, dbCategory := range dbCategories {
		category := Category{
			ID:        uuid.UUID(dbCategory.ID.Bytes).String(),
			Name:      dbCategory.Name,
			CreatedAt: dbCategory.CreatedAt.Time,
			UpdatedAt: dbCategory.UpdatedAt.Time,
		}

		if dbCategory.Description.Valid {
			category.Description = &dbCategory.Description.String
		}
		if dbCategory.Color.Valid {
			category.Color = &dbCategory.Color.String
		}

		categories = append(categories, category)
	}

	c.JSON(http.StatusOK, categories)
}

// @Summary Create category
// @Description Create a new category in the system
// @Tags categories
// @Accept json
// @Produce json
// @Param category body Category true "Category data (name required, description and color optional)"
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

	// Create parameters for the generated function
	params := generated.CreateCategoryParams{
		Name: category.Name,
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

	c.JSON(http.StatusCreated, resultCategory)
}

// @Summary Update category
// @Description Update an existing category
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

	// Create parameters for the generated function
	params := generated.UpdateCategoryParams{
		ID:   pgtype.UUID{Bytes: categoryUUID, Valid: true},
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