package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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
	Category        *string   `json:"category"`
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

var db *sql.DB

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
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Attempt %d: Error opening database: %v", i+1, err)
			time.Sleep(retryInterval)
			continue
		}

		// Test database connection
		if err = db.Ping(); err != nil {
			log.Printf("Attempt %d: Error connecting to database: %v", i+1, err)
			db.Close()
			time.Sleep(retryInterval)
			continue
		}

		log.Println("Successfully connected to database")
		break
	}

	if err != nil {
		log.Fatal("Failed to connect to database after retries: ", err)
	}
	defer db.Close()

	// Run database migrations
	migrationsPath := filepath.Join(".", "db", "migrations")

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		log.Printf("Migrations directory not found at %s, skipping migrations", migrationsPath)
	} else {
		log.Println("Running database migrations...")
		if err := runMigrations(db, migrationsPath); err != nil {
			log.Fatal("Error running migrations: ", err)
		}

		// Display current migration version
		if version, dirty, err := getMigrationVersion(db, migrationsPath); err == nil {
			if dirty {
				log.Printf("Current migration version: %d (DIRTY - migration failed)", version)
			} else {
				log.Printf("Current migration version: %d", version)
			}
		}
		log.Println("Database migrations completed successfully")
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
	if len(records) > 0 && (records[0][0] == "Transaction Date" || records[0][0] == "description" || records[0][0] == "Description") {
		start = 1
	}

	for i := start; i < len(records); i++ {
		record := records[i]
		if len(record) < 4 { // Need at least 4 columns for the new format
			continue
		}

		var description string
		var amount float64
		var err error

		// Handle new CSV format: Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
		if len(record) >= 7 {
			description = record[3] // Description is in column 4 (index 3)

			// Try to get amount from Debit column (index 5) first, then Credit column (index 6)
			if record[5] != "" {
				amount, err = strconv.ParseFloat(record[5], 64)
			} else if record[6] != "" {
				amount, err = strconv.ParseFloat(record[6], 64)
			} else {
				continue // Skip if no amount found
			}
		} else {
			// Fallback to old format: description,amount
			description = record[0]
			amount, err = strconv.ParseFloat(record[1], 64)
		}

		if err != nil {
			continue
		}

		transaction := Transaction{
			Description: description,
			Amount:      amount,
			FileName:    fileName,
		}

		// Insert into database
		err = db.QueryRow(
			"INSERT INTO transactions (description, amount, file_name) VALUES ($1, $2, $3) RETURNING id, date_uploaded",
			transaction.Description, transaction.Amount, transaction.FileName,
		).Scan(&transaction.ID, &transaction.DateUploaded)

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

func getTransactions(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, description, amount, COALESCE(assigned_to, ''), date_uploaded,
			   COALESCE(file_name, ''), transaction_date, posted_date, card_number,
			   category, category_id, created_at, updated_at
		FROM transactions
		ORDER BY date_uploaded DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching transactions"})
		return
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		var transactionDate, postedDate, cardNumber, category sql.NullString
		var categoryID sql.NullString

		err := rows.Scan(&t.ID, &t.Description, &t.Amount, &t.AssignedTo, &t.DateUploaded,
			&t.FileName, &transactionDate, &postedDate, &cardNumber, &category, &categoryID,
			&t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning transaction: %v", err)
			continue
		}

		// Handle nullable fields
		if transactionDate.Valid {
			t.TransactionDate = &transactionDate.String
		}
		if postedDate.Valid {
			t.PostedDate = &postedDate.String
		}
		if cardNumber.Valid {
			t.CardNumber = &cardNumber.String
		}
		if category.Valid {
			t.Category = &category.String
		}
		if categoryID.Valid {
			t.CategoryID = &categoryID.String
		}

		transactions = append(transactions, t)
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

	_, err := db.Exec("UPDATE transactions SET assigned_to = $1 WHERE id = $2", request.AssignedTo, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction assigned successfully"})
}

func getPeople(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, email, created_at, updated_at FROM people ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching people"})
		return
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		var email sql.NullString

		err := rows.Scan(&p.ID, &p.Name, &email, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning person: %v", err)
			continue
		}

		if email.Valid {
			p.Email = &email.String
		}

		people = append(people, p)
	}

	c.JSON(http.StatusOK, people)
}

func createPerson(c *gin.Context) {
	var person Person
	if err := c.ShouldBindJSON(&person); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := db.QueryRow(
		"INSERT INTO people (name, email) VALUES ($1, $2) RETURNING id, created_at, updated_at",
		person.Name, person.Email,
	).Scan(&person.ID, &person.CreatedAt, &person.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating person"})
		return
	}

	c.JSON(http.StatusCreated, person)
}

func getTotals(c *gin.Context) {
	rows, err := db.Query(`
		SELECT assigned_to, SUM(amount) as total
		FROM transactions
		WHERE assigned_to IS NOT NULL AND assigned_to != ''
		GROUP BY assigned_to
		ORDER BY assigned_to
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating totals"})
		return
	}
	defer rows.Close()

	var totals []PersonTotal
	for rows.Next() {
		var pt PersonTotal
		err := rows.Scan(&pt.Name, &pt.Total)
		if err != nil {
			log.Printf("Error scanning total: %v", err)
			continue
		}
		totals = append(totals, pt)
	}

	c.JSON(http.StatusOK, totals)
}

func getCategories(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, COALESCE(description, ''), COALESCE(color, ''), created_at, updated_at FROM categories ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching categories"})
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		var description, color string
		err := rows.Scan(&cat.ID, &cat.Name, &description, &color, &cat.CreatedAt, &cat.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning category: %v", err)
			continue
		}

		if description != "" {
			cat.Description = &description
		}
		if color != "" {
			cat.Color = &color
		}

		categories = append(categories, cat)
	}

	c.JSON(http.StatusOK, categories)
}

func createCategory(c *gin.Context) {
	var category Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := db.QueryRow(
		"INSERT INTO categories (name, description, color) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at",
		category.Name, category.Description, category.Color,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating category"})
		return
	}

	c.JSON(http.StatusCreated, category)
}
