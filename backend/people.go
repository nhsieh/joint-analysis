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

// People handler functions

// @Summary Get all people
// @Description Retrieve all people from the database
// @Tags people
// @Produce json
// @Success 200 {array} Person "List of people"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/people [get]
func getPeople(c *gin.Context) {
	dbPeople, err := queries.GetPeople(context.Background())
	if err != nil {
		log.Printf("Error fetching people: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching people"})
		return
	}

	var people []Person
	for _, dbPerson := range dbPeople {
		person := Person{
			ID:        uuid.UUID(dbPerson.ID.Bytes).String(),
			Name:      dbPerson.Name,
			CreatedAt: dbPerson.CreatedAt.Time,
			UpdatedAt: dbPerson.UpdatedAt.Time,
		}
		if dbPerson.Email.Valid {
			person.Email = &dbPerson.Email.String
		}
		people = append(people, person)
	}

	c.JSON(http.StatusOK, people)
}

// @Summary Create person
// @Description Create a new person in the system
// @Tags people
// @Accept json
// @Produce json
// @Param person body Person true "Person data (name required, email optional)"
// @Success 201 {object} Person "Created person"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 409 {object} map[string]interface{} "Person already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/people [post]
func createPerson(c *gin.Context) {
	var personRequest Person
	if err := c.ShouldBindJSON(&personRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if err := validateName(personRequest.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create the parameters for the generated function
	params := generated.CreatePersonParams{
		Name: personRequest.Name,
	}

	// Handle optional email
	if personRequest.Email != nil && *personRequest.Email != "" {
		params.Email = pgtype.Text{String: *personRequest.Email, Valid: true}
	}

	dbPerson, err := queries.CreatePerson(context.Background(), params)
	if err != nil {
		log.Printf("Error creating person: %v", err)
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Convert to API person format
	person := Person{
		ID:        uuid.UUID(dbPerson.ID.Bytes).String(),
		Name:      dbPerson.Name,
		Email:     nil,
		CreatedAt: dbPerson.CreatedAt.Time,
		UpdatedAt: dbPerson.UpdatedAt.Time,
	}

	if dbPerson.Email.Valid {
		email := dbPerson.Email.String
		person.Email = &email
	}

	c.JSON(http.StatusCreated, person)
}

// @Summary Delete person
// @Description Delete a specific person by ID
// @Tags people
// @Produce json
// @Param id path string true "Person ID"
// @Success 200 {object} map[string]interface{} "Person deleted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Person not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/people/{id} [delete]
func deletePerson(c *gin.Context) {
	id := c.Param("id")

	// Parse UUID from string
	personUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid person ID"})
		return
	}

	// Create pgtype.UUID for the queries
	personUUIDpg := pgtype.UUID{Bytes: personUUID, Valid: true}

	// First, get the person to ensure they exist
	_, err = queries.GetPersonByID(context.Background(), personUUIDpg)
	if err != nil {
		log.Printf("Error finding person: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	// Unassign all transactions that are assigned to this person (by UUID)
	err = queries.UnassignTransactionsByPerson(context.Background(), personUUIDpg)
	if err != nil {
		log.Printf("Error unassigning transactions for person %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unassigning transactions"})
		return
	}

	// Now delete the person
	err = queries.DeletePerson(context.Background(), personUUIDpg)
	if err != nil {
		log.Printf("Error deleting person: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting person"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Person deleted successfully"})
}