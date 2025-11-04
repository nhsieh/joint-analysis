// @title Joint Analysis API
// @version 1.0
// @description API for the Joint Analysis expense tracking application
// @host localhost:8081
// @BasePath /
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"jointanalysis/db/generated"

	_ "jointanalysis/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var dbPool *pgxpool.Pool
var queries *generated.Queries
var categoryMapping *CategoryMapping

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

	// Run database migrations first
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

	// Initialize category mapping after migrations
	categoryMapping, err = initializeCategoryMapping()
	if err != nil {
		log.Printf("Warning: Failed to initialize category mapping: %v", err)
		log.Println("Transactions will be created without categories")
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
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
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