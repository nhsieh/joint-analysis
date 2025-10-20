package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"jointanalysis/db/generated"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Transaction struct {
	ID              string    `json:"id"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	AssignedTo      string    `json:"assigned_to"`
	DateUploaded    time.Time `json:"date_uploaded"`
	FileName        string    `json:"file_name"`
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

var dbPool *pgxpool.Pool
var queries *generated.Queries
var categoryMapping *CategoryMapping

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

	// TODO: Update migrations to work with pgx connection
	log.Println("Database migrations disabled during refactor - run manually if needed")

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
	r.PUT("/api/transactions/:id/assign", assignTransaction)
	r.GET("/api/people", getPeople)
	r.POST("/api/people", createPerson)
	r.GET("/api/categories", getCategories)
	r.POST("/api/categories", createCategory)
	r.GET("/api/totals", getTotals)

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

	var transactions []Transaction
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
			FileName:    fileName,
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
func convertTransaction(t generated.Transaction) Transaction {
	result := Transaction{
		ID:           uuid.UUID(t.ID.Bytes).String(), // Convert UUID to string
		Description:  t.Description,
		AssignedTo:   "",
		DateUploaded: t.DateUploaded.Time,
		FileName:     "",
		CreatedAt:    t.CreatedAt.Time,
		UpdatedAt:    t.UpdatedAt.Time,
	}

	// Convert numeric amount
	if t.Amount.Valid {
		amountFloat, _ := t.Amount.Float64Value()
		result.Amount = amountFloat.Float64
	}

	// Handle nullable fields
	if t.AssignedTo.Valid {
		result.AssignedTo = t.AssignedTo.String
	}
	if t.FileName.Valid {
		result.FileName = t.FileName.String
	}
	if t.TransactionDate.Valid {
		dateStr := t.TransactionDate.Time.Format("2006-01-02")
		result.TransactionDate = &dateStr
	}
	if t.PostedDate.Valid {
		dateStr := t.PostedDate.Time.Format("2006-01-02")
		result.PostedDate = &dateStr
	}
	if t.CardNumber.Valid {
		result.CardNumber = &t.CardNumber.String
	}
	if t.CategoryID.Valid {
		categoryStr := uuid.UUID(t.CategoryID.Bytes).String()
		result.CategoryID = &categoryStr
	}

	return result
}

func getTransactions(c *gin.Context) {
	dbTransactions, err := queries.GetTransactions(context.Background())
	if err != nil {
		log.Printf("Error fetching transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching transactions"})
		return
	}

	// Convert to API transaction format
	var transactions []Transaction
	for _, t := range dbTransactions {
		transactions = append(transactions, convertTransaction(t))
	}

	c.JSON(http.StatusOK, transactions)
}

func assignTransaction(c *gin.Context) {
	id := c.Param("id")
	var request struct {
		AssignedTo string `json:"assigned_to"`
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

	// Create parameters for the generated function
	params := generated.UpdateTransactionAssignmentParams{
		ID:         pgtype.UUID{Bytes: transactionUUID, Valid: true},
		AssignedTo: pgtype.Text{String: request.AssignedTo, Valid: request.AssignedTo != ""},
	}

	dbTransaction, err := queries.UpdateTransactionAssignment(context.Background(), params)
	if err != nil {
		log.Printf("Error updating transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating transaction"})
		return
	}

	// Convert and return the updated transaction
	transaction := convertTransaction(dbTransaction)
	c.JSON(http.StatusOK, transaction)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating person"})
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

func getTotals(c *gin.Context) {
	dbTotals, err := queries.GetTotalsByAssignedTo(context.Background())
	if err != nil {
		log.Printf("Error calculating totals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating totals"})
		return
	}

	var totals []Total
	for _, dbTotal := range dbTotals {
		person := "Unassigned"
		if dbTotal.AssignedTo.Valid && dbTotal.AssignedTo.String != "" {
			person = dbTotal.AssignedTo.String
		}

		// Convert pgtype.Numeric to float64
		totalValue, _ := dbTotal.Total.Float64Value()

		total := Total{
			Person: person,
			Total:  totalValue.Float64,
		}
		totals = append(totals, total)
	} // Add unassigned total if there are any unassigned transactions
	// This needs a separate query since the generated query excludes NULL/empty assigned_to
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating category"})
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
