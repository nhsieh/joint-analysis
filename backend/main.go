package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"jointanalysis/db/generated"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type Transaction struct {
	ID              string    `json:"id"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	AssignedTo      []string  `json:"assigned_to"`
	DateUploaded    time.Time `json:"date_uploaded"`
	FileName        *string   `json:"file_name"`
	TransactionDate *string   `json:"transaction_date"`
	PostedDate      *string   `json:"posted_date"`
	CardNumber      *string   `json:"card_number"`
	CategoryID      *string   `json:"category_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Person struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Color       *string   `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PersonTotal struct {
	Name  string  `json:"name"`
	Total float64 `json:"total"`
}

type Total struct {
	Person string  `json:"person"`
	Total  float64 `json:"total"`
}

// Archive represents an archived collection of transactions
type Archive struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Description      *string       `json:"description"`
	ArchivedAt       time.Time     `json:"archived_at"`
	TransactionCount int           `json:"transaction_count"`
	TotalAmount      float64       `json:"total_amount"`
	PersonTotals     []PersonTotal `json:"person_totals,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// ArchiveRequest represents the request structure for creating an archive
type ArchiveRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

var dbPool *pgxpool.Pool
var queries *generated.Queries
var categoryMapping *CategoryMapping

// Validation functions

// validateName validates that a name is not empty or just whitespace
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

// validateHexColor validates that a color is in hex format (#RRGGBB)
func validateHexColor(color string) error {
	if color == "" {
		return nil // Empty color is allowed
	}

	hexColorRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	if !hexColorRegex.MatchString(color) {
		return fmt.Errorf("color must be in hex format (#RRGGBB)")
	}
	return nil
}

// handleDatabaseError converts database errors to appropriate HTTP responses
func handleDatabaseError(err error) (statusCode int, message string) {
	errorStr := err.Error()

	// Check for unique constraint violations
	if strings.Contains(errorStr, "duplicate key value violates unique constraint") {
		if strings.Contains(errorStr, "people_name_key") {
			return http.StatusConflict, "Person with this name already exists"
		}
		if strings.Contains(errorStr, "categories_name_key") {
			return http.StatusConflict, "Category with this name already exists"
		}
		return http.StatusConflict, "Resource already exists"
	}

	// Check for not found errors
	if strings.Contains(errorStr, "no rows in result set") {
		return http.StatusNotFound, "Resource not found"
	}

	// Default to internal server error
	return http.StatusInternalServerError, "Internal server error"
}

// CategoryMapping maps CSV categories to our predefined categories
type CategoryMapping struct {
	categoriesByName map[string]generated.Category
}

// mapTransactionCategory determines the best category for a transaction
func (cm *CategoryMapping) mapTransactionCategory(csvCategory string) *generated.Category {
	// First try direct mapping from CSV category
	if category, exists := cm.categoriesByName[csvCategory]; exists {
		return &category
	}

	// Map common CSV categories to our categories
	csvCategoryMap := map[string]string{
		"Gas/Automotive":      "Transportation",
		"Insurance":           "Healthcare",
		"Dining":              "Food & Dining",
		"Other Travel":        "Travel",
		"Merchandise":         "Shopping",
		"Fee/Interest Charge": "Fees",
	}

	if mappedCategory, exists := csvCategoryMap[csvCategory]; exists {
		if category, exists := cm.categoriesByName[mappedCategory]; exists {
			return &category
		}
	}

	// Default to "Other" if no match found
	if category, exists := cm.categoriesByName["Other"]; exists {
		return &category
	}

	return nil
}

// initializeCategoryMapping loads categories and creates keyword mappings
func initializeCategoryMapping() (*CategoryMapping, error) {
	categories, err := queries.GetCategories(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}

	categoriesByName := make(map[string]generated.Category)
	for _, category := range categories {
		categoriesByName[category.Name] = category
	}

	return &CategoryMapping{
		categoriesByName: categoriesByName,
	}, nil
}

func main() {
	var err error

	// Database connection with defaults
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "password"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "jointanalysis"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect to database with retry logic
	maxRetries := 30
	retryInterval := time.Second * 2

	for i := 0; i < maxRetries; i++ {
		dbPool, err = pgxpool.New(context.Background(), connStr)
		if err != nil {
			log.Printf("Attempt %d: Error creating database pool: %v", i+1, err)
			time.Sleep(retryInterval)
			continue
		}

		// Test database connection
		if err = dbPool.Ping(context.Background()); err != nil {
			log.Printf("Attempt %d: Error pinging database: %v", i+1, err)
			dbPool.Close()
			time.Sleep(retryInterval)
			continue
		}

		log.Println("Successfully connected to database")
		break
	}

	if err != nil {
		log.Fatal("Failed to connect to database after retries: ", err)
	}
	defer dbPool.Close()

	// Initialize the generated queries
	queries = generated.New(dbPool)

	// Initialize category mapping
	categoryMapping, err = initializeCategoryMapping()
	if err != nil {
		log.Printf("Warning: Failed to initialize category mapping: %v", err)
		log.Println("Transactions will be created without categories")
	}

	// Run database migrations
	// Create a separate sql.DB connection for migrations (golang-migrate requires it)
	migrationConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	migrationDB, err := sql.Open("postgres", migrationConnStr)
	if err != nil {
		log.Fatalf("Failed to create migration database connection: %v", err)
	}
	defer migrationDB.Close()

	if err := runMigrations(migrationDB, "db/migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Routes
	r.POST("/api/upload-csv", uploadCSV)
	r.GET("/api/transactions", getTransactions)
	r.DELETE("/api/transactions", clearAllTransactions)
	r.DELETE("/api/transactions/:id", deleteTransaction)
	r.PUT("/api/transactions/:id/assign", assignTransaction)
	r.GET("/api/people", getPeople)
	r.POST("/api/people", createPerson)
	r.DELETE("/api/people/:id", deletePerson)
	r.GET("/api/categories", getCategories)
	r.POST("/api/categories", createCategory)
	r.PUT("/api/categories/:id", updateCategory)
	r.DELETE("/api/categories/:id", deleteCategory)
	r.PUT("/api/transactions/:id/category", updateTransactionCategory)
	r.GET("/api/totals", getTotals)
	r.POST("/api/archives", createArchive)
	r.GET("/api/archives", getArchives)
	r.GET("/api/archives/:id/transactions", getArchiveTransactions)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}

func uploadCSV(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading CSV file"})
		return
	}

	transactions := make([]Transaction, 0) // Initialize as empty slice instead of nil
	fileName := header.Filename

	// Skip header row if present
	start := 0
	if len(records) > 0 && records[0][0] == "Transaction Date" {
		start = 1
	}

	for i := start; i < len(records); i++ {
		record := records[i]
		if len(record) < 7 { // Need exactly 7 columns for CSV format: Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
			continue
		}

		var description string
		var amount float64
		var err error

		// Handle CSV format: Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
		transactionDate := record[0] // Transaction Date
		postedDate := record[1]      // Posted Date
		cardNumber := record[2]      // Card No.
		description = record[3]      // Description
		csvCategory := record[4]     // Category from CSV

		// Try to get amount from Debit column (index 5) first, then Credit column (index 6)
		if record[5] != "" {
			amount, err = strconv.ParseFloat(record[5], 64)
		} else if record[6] != "" {
			amount, err = strconv.ParseFloat(record[6], 64)
		} else {
			continue // Skip if no amount found
		}

		if err != nil {
			continue
		}

		transaction := Transaction{
			Description: description,
			Amount:      amount,
			FileName:    &fileName,
		}

		// Add the additional fields if they exist
		if transactionDate != "" {
			transaction.TransactionDate = &transactionDate
		}
		if postedDate != "" {
			transaction.PostedDate = &postedDate
		}
		if cardNumber != "" {
			transaction.CardNumber = &cardNumber
		}

		// Insert into database using generated query
		// Convert float64 to pgtype.Numeric
		amountBig := big.NewFloat(amount)
		amountStr := amountBig.Text('f', 2) // Format to 2 decimal places
		var amountNumeric pgtype.Numeric
		if err := amountNumeric.Scan(amountStr); err != nil {
			log.Printf("Error converting amount to numeric: %v", err)
			continue
		}

		params := generated.CreateTransactionParams{
			Description: description,
			Amount:      amountNumeric,
			FileName:    pgtype.Text{String: fileName, Valid: true},
		}

		// Map category if category mapping is available
		if categoryMapping != nil {
			if mappedCategory := categoryMapping.mapTransactionCategory(csvCategory); mappedCategory != nil {
				params.CategoryID = pgtype.UUID{Bytes: mappedCategory.ID.Bytes, Valid: mappedCategory.ID.Valid}
			}
		}

		// Add optional fields
		if transactionDate != "" {
			if parsedDate, err := time.Parse("2006-01-02", transactionDate); err == nil {
				params.TransactionDate = pgtype.Date{Time: parsedDate, Valid: true}
			}
		}
		if postedDate != "" {
			if parsedDate, err := time.Parse("2006-01-02", postedDate); err == nil {
				params.PostedDate = pgtype.Date{Time: parsedDate, Valid: true}
			}
		}
		if cardNumber != "" {
			params.CardNumber = pgtype.Text{String: cardNumber, Valid: true}
		}

		// Check for duplicate transaction before inserting
		duplicateParams := generated.FindDuplicateTransactionParams{
			Description:     description,
			Amount:          amountNumeric,
			TransactionDate: params.TransactionDate,
			PostedDate:      params.PostedDate,
			CardNumber:      params.CardNumber,
		}

		count, err := queries.FindDuplicateTransaction(context.Background(), duplicateParams)
		if err != nil {
			log.Printf("Error checking for duplicate transaction: %v", err)
			continue
		}

		// If duplicate exists, skip this transaction
		if count > 0 {
			log.Printf("Skipping duplicate transaction: %s, amount: %f", description, amount)
			continue
		}

		_, err = queries.CreateTransaction(context.Background(), params)
		if err != nil {
			log.Printf("Error inserting transaction: %v", err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "CSV uploaded successfully",
		"transactions": transactions,
	})
}

// convertTransaction converts from generated.Transaction to API Transaction
// Helper function to convert UUID array to person names
func convertUUIDArrayToNames(uuidArray []pgtype.UUID) ([]string, error) {
	if len(uuidArray) == 0 {
		return []string{}, nil
	}

	var names []string
	for _, uuidPg := range uuidArray {
		if uuidPg.Valid {
			person, err := queries.GetPersonByID(context.Background(), uuidPg)
			if err != nil {
				log.Printf("Error getting person by ID %v: %v", uuidPg, err)
				continue // Skip invalid UUIDs instead of failing completely
			}
			names = append(names, person.Name)
		}
	}
	return names, nil
}

// Helper function to convert person names to UUID array
func convertNamesToUUIDArray(names []string) ([]pgtype.UUID, error) {
	if len(names) == 0 {
		return []pgtype.UUID{}, nil
	}

	var uuids []pgtype.UUID
	for _, name := range names {
		person, err := queries.GetPersonByName(context.Background(), name)
		if err != nil {
			log.Printf("Error getting person by name %s: %v", name, err)
			continue // Skip invalid names instead of failing completely
		}
		uuids = append(uuids, person.ID)
	}
	return uuids, nil
}

func convertUUIDStringsToArray(uuidStrings []string) ([]pgtype.UUID, error) {
	if len(uuidStrings) == 0 {
		return []pgtype.UUID{}, nil
	}

	var uuids []pgtype.UUID
	for _, uuidStr := range uuidStrings {
		parsedUUID, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID format: %s", uuidStr)
		}
		uuids = append(uuids, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	}
	return uuids, nil
}

func convertTransaction(t generated.Transaction) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

func convertTransactionFromGetRow(t generated.Transaction) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

func convertTransactionFromUpdateRow(t generated.Transaction) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

func convertTransactionFromUpdateAssignmentRow(t generated.UpdateTransactionAssignmentRow) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

func convertTransactionFromUpdateCategoryRow(t generated.UpdateTransactionCategoryRow) Transaction {
	return convertTransactionFromFields(
		t.ID, t.Description, t.Amount, t.AssignedTo, t.DateUploaded, t.FileName,
		t.TransactionDate, t.PostedDate, t.CardNumber, t.CategoryID, t.CreatedAt, t.UpdatedAt,
	)
}

func convertTransactionFromFields(
	id pgtype.UUID,
	description string,
	amount pgtype.Numeric,
	assignedTo []pgtype.UUID,
	dateUploaded pgtype.Timestamp,
	fileName pgtype.Text,
	transactionDate pgtype.Date,
	postedDate pgtype.Date,
	cardNumber pgtype.Text,
	categoryID pgtype.UUID,
	createdAt pgtype.Timestamp,
	updatedAt pgtype.Timestamp,
) Transaction {
	result := Transaction{
		ID:           uuid.UUID(id.Bytes).String(), // Convert UUID to string
		Description:  description,
		AssignedTo:   []string{}, // Initialize as empty array
		DateUploaded: dateUploaded.Time,
		FileName:     nil,
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}

	// Convert numeric amount
	if amount.Valid {
		amountFloat, _ := amount.Float64Value()
		result.Amount = amountFloat.Float64
	}

	// Convert UUID array to person names
	if len(assignedTo) > 0 {
		names, err := convertUUIDArrayToNames(assignedTo)
		if err != nil {
			log.Printf("Error converting UUIDs to names: %v", err)
		} else {
			result.AssignedTo = names
		}
	}

	// Handle nullable fields
	if fileName.Valid {
		result.FileName = &fileName.String
	}
	if transactionDate.Valid {
		dateStr := transactionDate.Time.Format("2006-01-02")
		result.TransactionDate = &dateStr
	}
	if postedDate.Valid {
		dateStr := postedDate.Time.Format("2006-01-02")
		result.PostedDate = &dateStr
	}
	if cardNumber.Valid {
		result.CardNumber = &cardNumber.String
	}
	if categoryID.Valid {
		categoryStr := uuid.UUID(categoryID.Bytes).String()
		result.CategoryID = &categoryStr
	}

	return result
}

func getTransactions(c *gin.Context) {
	dbTransactions, err := queries.GetActiveTransactions(context.Background())
	if err != nil {
		log.Printf("Error fetching active transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching active transactions"})
		return
	}

	// Convert to API transaction format
	var transactions []Transaction
	for _, t := range dbTransactions {
		transactions = append(transactions, convertTransactionFromActiveRow(t))
	}

	c.JSON(http.StatusOK, transactions)
}

func assignTransaction(c *gin.Context) {
	id := c.Param("id")
	var request struct {
		AssignedTo []string `json:"assigned_to"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Parse UUID from string
	transactionUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	// Convert UUID strings to pgtype.UUID array
	assignedUUIDs, err := convertUUIDStringsToArray(request.AssignedTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing person UUIDs"})
		return
	}

	// Create parameters for the generated function
	params := generated.UpdateTransactionAssignmentParams{
		ID:         pgtype.UUID{Bytes: transactionUUID, Valid: true},
		AssignedTo: assignedUUIDs,
	}

	dbTransaction, err := queries.UpdateTransactionAssignment(context.Background(), params)
	if err != nil {
		log.Printf("Error updating transaction: %v", err)
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Convert and return the updated transaction
	transaction := convertTransactionFromUpdateAssignmentRow(dbTransaction)
	c.JSON(http.StatusOK, transaction)
}

func deleteTransaction(c *gin.Context) {
	transactionID := c.Param("id")
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction ID is required"})
		return
	}

	// Parse UUID
	transactionUUID, err := uuid.Parse(transactionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID format"})
		return
	}

	// Convert to pgtype.UUID
	var pgUUID pgtype.UUID
	pgUUID.Bytes = transactionUUID
	pgUUID.Valid = true

	err = queries.DeleteTransaction(context.Background(), pgUUID)
	if err != nil {
		log.Printf("Error deleting transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
}

func clearAllTransactions(c *gin.Context) {
	err := queries.DeleteAllTransactions(context.Background())
	if err != nil {
		log.Printf("Error clearing all transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error clearing transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All transactions cleared successfully"})
}

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

func getTotals(c *gin.Context) {
	dbTotals, err := queries.GetActiveTransactionTotals(context.Background())
	if err != nil {
		log.Printf("Error calculating totals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating totals"})
		return
	}

	var totals []Total
	for _, dbTotal := range dbTotals {
		// Convert pgtype.Numeric to float64
		totalValue, _ := dbTotal.Total.Float64Value()

		total := Total{
			Person: dbTotal.AssignedTo, // This is now a string (person name) from the query
			Total:  totalValue.Float64,
		}
		totals = append(totals, total)
	}

	// TODO: Add unassigned total if there are any unassigned transactions
	// This would need a separate query since the current query excludes transactions with empty assigned_to arrays

	c.JSON(http.StatusOK, totals)
}

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

func updateTransactionCategory(c *gin.Context) {
	id := c.Param("id")
	var request struct {
		CategoryID *string `json:"category_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Parse transaction UUID from string
	transactionUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	// Create parameters for the generated function
	params := generated.UpdateTransactionCategoryParams{
		ID: pgtype.UUID{Bytes: transactionUUID, Valid: true},
	}

	// Handle category ID
	if request.CategoryID != nil && *request.CategoryID != "" {
		categoryUUID, err := uuid.Parse(*request.CategoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		params.CategoryID = pgtype.UUID{Bytes: categoryUUID, Valid: true}
	} else {
		// Set to NULL if no category provided
		params.CategoryID = pgtype.UUID{Valid: false}
	}

	dbTransaction, err := queries.UpdateTransactionCategory(context.Background(), params)
	if err != nil {
		log.Printf("Error updating transaction category: %v", err)
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Convert and return the updated transaction
	transaction := convertTransactionFromUpdateCategoryRow(dbTransaction)
	c.JSON(http.StatusOK, transaction)
}

// Archive handlers

func createArchive(c *gin.Context) {
	var request ArchiveRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate name
	if err := validateName(request.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get all active transactions to archive
	activeTransactions, err := queries.GetActiveTransactions(context.Background())
	if err != nil {
		log.Printf("Error fetching active transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching active transactions"})
		return
	}

	if len(activeTransactions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active transactions"})
		return
	}

	// Get current totals for active transactions (this gives us individual person totals)
	activeTotals, err := queries.GetActiveTransactionTotals(context.Background())
	if err != nil {
		log.Printf("Error fetching active totals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating totals"})
		return
	}

	// Calculate total amount (sum of all individual person totals)
	var totalAmount float64
	for _, total := range activeTotals {
		totalValue, _ := total.Total.Float64Value()
		totalAmount += totalValue.Float64
	}

	// Create archive
	var descText pgtype.Text
	if request.Description != "" {
		descText = pgtype.Text{String: request.Description, Valid: true}
	}

	params := generated.CreateArchiveParams{
		Name:             request.Name,
		Description:      descText,
		TransactionCount: int32(len(activeTransactions)),
		TotalAmount:      pgtype.Numeric{},
	}

	// Convert float64 to pgtype.Numeric
	amountBig := big.NewFloat(totalAmount)
	amountStr := amountBig.Text('f', 2) // Format to 2 decimal places
	err = params.TotalAmount.Scan(amountStr)
	if err != nil {
		log.Printf("Error converting total amount: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing total amount"})
		return
	}

	archive, err := queries.CreateArchive(context.Background(), params)
	if err != nil {
		log.Printf("Error creating archive: %v", err)
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Archive all active transactions
	archiveID := pgtype.UUID{Bytes: archive.ID.Bytes, Valid: true}
	err = queries.ArchiveTransactions(context.Background(), archiveID)
	if err != nil {
		log.Printf("Error archiving transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error archiving transactions"})
		return
	}

	// Store individual person totals for this archive
	var personTotals []PersonTotal
	for _, total := range activeTotals {
		// Parse person ID from the total (we need to get it from people table)
		person, err := queries.GetPersonByName(context.Background(), total.AssignedTo)
		if err != nil {
			log.Printf("Error finding person %s: %v", total.AssignedTo, err)
			continue
		}

		totalValue, _ := total.Total.Float64Value()
		totalNumeric := pgtype.Numeric{}
		totalBig := big.NewFloat(totalValue.Float64)
		totalStr := totalBig.Text('f', 2)
		totalNumeric.Scan(totalStr)

		_, err = queries.CreateArchivePersonTotal(context.Background(), generated.CreateArchivePersonTotalParams{
			ArchiveID:   archiveID,
			PersonID:    person.ID,
			TotalAmount: totalNumeric,
		})
		if err != nil {
			log.Printf("Error creating person total for %s: %v", person.Name, err)
			continue
		}

		personTotals = append(personTotals, PersonTotal{
			Name:  person.Name,
			Total: totalValue.Float64,
		})
	}

	// Convert and return the archive
	archiveResponse := Archive{
		ID:               uuid.UUID(archive.ID.Bytes).String(),
		Name:             archive.Name,
		ArchivedAt:       archive.ArchivedAt.Time,
		TransactionCount: int(archive.TransactionCount),
		PersonTotals:     personTotals,
		CreatedAt:        archive.CreatedAt.Time,
		UpdatedAt:        archive.UpdatedAt.Time,
	}

	if archive.Description.Valid {
		archiveResponse.Description = &archive.Description.String
	}

	totalValue, _ := archive.TotalAmount.Float64Value()
	archiveResponse.TotalAmount = totalValue.Float64

	c.JSON(http.StatusCreated, archiveResponse)
}

func getArchives(c *gin.Context) {
	dbArchives, err := queries.GetArchives(context.Background())
	if err != nil {
		log.Printf("Error fetching archives: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching archives"})
		return
	}

	var archives []Archive
	for _, dbArchive := range dbArchives {
		// Get person totals for this archive
		dbPersonTotals, err := queries.GetArchivePersonTotals(context.Background(), dbArchive.ID)
		if err != nil {
			log.Printf("Error fetching person totals for archive %s: %v", uuid.UUID(dbArchive.ID.Bytes).String(), err)
			// Continue without person totals rather than failing
		}

		var personTotals []PersonTotal
		for _, dbPersonTotal := range dbPersonTotals {
			totalValue, _ := dbPersonTotal.TotalAmount.Float64Value()
			personTotals = append(personTotals, PersonTotal{
				Name:  dbPersonTotal.PersonName,
				Total: totalValue.Float64,
			})
		}

		archive := Archive{
			ID:               uuid.UUID(dbArchive.ID.Bytes).String(),
			Name:             dbArchive.Name,
			ArchivedAt:       dbArchive.ArchivedAt.Time,
			TransactionCount: int(dbArchive.TransactionCount),
			PersonTotals:     personTotals,
			CreatedAt:        dbArchive.CreatedAt.Time,
			UpdatedAt:        dbArchive.UpdatedAt.Time,
		}

		if dbArchive.Description.Valid {
			archive.Description = &dbArchive.Description.String
		}

		totalValue, _ := dbArchive.TotalAmount.Float64Value()
		archive.TotalAmount = totalValue.Float64

		archives = append(archives, archive)
	}

	c.JSON(http.StatusOK, archives)
}

func getArchiveTransactions(c *gin.Context) {
	id := c.Param("id")

	// Parse archive UUID
	archiveUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid archive ID"})
		return
	}

	// Check if archive exists
	_, err = queries.GetArchiveByID(context.Background(), pgtype.UUID{Bytes: archiveUUID, Valid: true})
	if err != nil {
		log.Printf("Error fetching archive: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Archive not found"})
		return
	}

	// Get archived transactions
	dbTransactions, err := queries.GetArchivedTransactions(context.Background(), pgtype.UUID{Bytes: archiveUUID, Valid: true})
	if err != nil {
		log.Printf("Error fetching archived transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching archived transactions"})
		return
	}

	var transactions []Transaction
	for _, t := range dbTransactions {
		transactions = append(transactions, convertTransactionFromArchivedRow(t))
	}

	c.JSON(http.StatusOK, transactions)
}

// Helper function to convert archived transaction row to Transaction struct
func convertTransactionFromArchivedRow(t generated.GetArchivedTransactionsRow) Transaction {
	transaction := Transaction{
		ID:          uuid.UUID(t.ID.Bytes).String(),
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
		UpdatedAt:   t.UpdatedAt.Time,
	}

	// Convert amount
	if amountValue, err := t.Amount.Float64Value(); err == nil {
		transaction.Amount = amountValue.Float64
	}

	// Convert assigned_to array from UUIDs to names
	if len(t.AssignedTo) > 0 {
		names, err := convertUUIDArrayToNames(t.AssignedTo)
		if err != nil {
			log.Printf("Error converting UUIDs to names: %v", err)
		} else {
			transaction.AssignedTo = names
		}
	} else {
		transaction.AssignedTo = []string{}
	}

	// Convert optional fields
	if t.DateUploaded.Valid {
		transaction.DateUploaded = t.DateUploaded.Time
	}
	if t.FileName.Valid {
		transaction.FileName = &t.FileName.String
	}
	if t.TransactionDate.Valid {
		dateStr := t.TransactionDate.Time.Format("2006-01-02")
		transaction.TransactionDate = &dateStr
	}
	if t.PostedDate.Valid {
		dateStr := t.PostedDate.Time.Format("2006-01-02")
		transaction.PostedDate = &dateStr
	}
	if t.CardNumber.Valid {
		transaction.CardNumber = &t.CardNumber.String
	}
	if t.CategoryID.Valid {
		categoryID := uuid.UUID(t.CategoryID.Bytes).String()
		transaction.CategoryID = &categoryID
	}

	return transaction
}

// Helper function to convert active transaction row to Transaction struct
func convertTransactionFromActiveRow(t generated.GetActiveTransactionsRow) Transaction {
	transaction := Transaction{
		ID:          uuid.UUID(t.ID.Bytes).String(),
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
		UpdatedAt:   t.UpdatedAt.Time,
	}

	// Convert amount
	if amountValue, err := t.Amount.Float64Value(); err == nil {
		transaction.Amount = amountValue.Float64
	}

	// Convert assigned_to array from UUIDs to names
	if len(t.AssignedTo) > 0 {
		names, err := convertUUIDArrayToNames(t.AssignedTo)
		if err != nil {
			log.Printf("Error converting UUIDs to names: %v", err)
			transaction.AssignedTo = []string{} // Initialize as empty array
		} else {
			transaction.AssignedTo = names
		}
	} else {
		transaction.AssignedTo = []string{} // Initialize as empty array
	}

	// Convert optional fields
	if t.DateUploaded.Valid {
		transaction.DateUploaded = t.DateUploaded.Time
	}
	if t.FileName.Valid {
		transaction.FileName = &t.FileName.String
	}
	if t.TransactionDate.Valid {
		dateStr := t.TransactionDate.Time.Format("2006-01-02")
		transaction.TransactionDate = &dateStr
	}
	if t.PostedDate.Valid {
		dateStr := t.PostedDate.Time.Format("2006-01-02")
		transaction.PostedDate = &dateStr
	}
	if t.CardNumber.Valid {
		transaction.CardNumber = &t.CardNumber.String
	}
	if t.CategoryID.Valid {
		categoryID := uuid.UUID(t.CategoryID.Bytes).String()
		transaction.CategoryID = &categoryID
	}

	return transaction
}
