package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"jointanalysis/db/generated"
	"log"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

var (
	testDB      *pgxpool.Pool
	testQueries *generated.Queries
	testRouter  *gin.Engine
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup test database
	if err := setupTestDB(); err != nil {
		log.Fatalf("Failed to setup test database: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := teardownTestDB(); err != nil {
		log.Printf("Failed to cleanup test database: %v", err)
	}

	os.Exit(code)
}

// setupTestDB creates a test database and runs migrations
func setupTestDB() error {
	// Use test database configuration
	dbHost := getEnvOrDefault("TEST_DB_HOST", "localhost")
	dbPort := getEnvOrDefault("TEST_DB_PORT", "5433")
	dbUser := getEnvOrDefault("TEST_DB_USER", "postgres")
	dbPassword := getEnvOrDefault("TEST_DB_PASSWORD", "password")
	dbName := getEnvOrDefault("TEST_DB_NAME", "jointanalysis_test")

	// Create test database if it doesn't exist
	adminConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword)

	adminDB, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to admin database: %w", err)
	}
	defer adminDB.Close()

	// Drop and recreate test database for clean state
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to drop test database: %w", err)
	}

	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to create test database: %w", err)
	}

	// Connect to test database
	testConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err = pgxpool.New(context.Background(), testConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Run migrations
	testSQLDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		return fmt.Errorf("failed to create SQL connection for migrations: %w", err)
	}
	defer testSQLDB.Close()

	if err := runMigrations(testSQLDB, "db/migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize test queries
	testQueries = generated.New(testDB)

	// Setup test router
	setupTestRouter()

	return nil
}

// teardownTestDB cleans up the test database
func teardownTestDB() error {
	if testDB != nil {
		testDB.Close()
	}
	return nil
}

// setupTestRouter configures the test router with all routes
func setupTestRouter() {
	// Set global variables for testing
	dbPool = testDB
	queries = testQueries

	testRouter = gin.New()

	// Add routes (same as main function)
	testRouter.POST("/api/upload-csv", uploadCSV)
	testRouter.GET("/api/transactions", getTransactions)
	testRouter.DELETE("/api/transactions", clearAllTransactions)
	testRouter.PUT("/api/transactions/:id/assign", assignTransaction)
	testRouter.GET("/api/people", getPeople)
	testRouter.POST("/api/people", createPerson)
	testRouter.DELETE("/api/people/:id", deletePerson)
	testRouter.GET("/api/categories", getCategories)
	testRouter.POST("/api/categories", createCategory)
	testRouter.PUT("/api/categories/:id", updateCategory)
	testRouter.DELETE("/api/categories/:id", deleteCategory)
	testRouter.PUT("/api/transactions/:id/category", updateTransactionCategory)
	testRouter.GET("/api/totals", getTotals)
}

// cleanupTestData removes all data from test tables
func cleanupTestData() error {
	ctx := context.Background()

	// Clean in reverse dependency order
	if _, err := testDB.Exec(ctx, "DELETE FROM transactions"); err != nil {
		return fmt.Errorf("failed to clean transactions: %w", err)
	}

	if _, err := testDB.Exec(ctx, "DELETE FROM categories"); err != nil {
		return fmt.Errorf("failed to clean categories: %w", err)
	}

	if _, err := testDB.Exec(ctx, "DELETE FROM people"); err != nil {
		return fmt.Errorf("failed to clean people: %w", err)
	}

	return nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createTestPerson creates a test person and returns the ID
func createTestPerson(name, email string) (string, error) {
	var emailText pgtype.Text
	if email != "" {
		emailText = pgtype.Text{String: email, Valid: true}
	}

	person, err := testQueries.CreatePerson(context.Background(), generated.CreatePersonParams{
		Name:  name,
		Email: emailText,
	})
	if err != nil {
		return "", err
	}

	return person.ID.String(), nil
}

// createTestCategory creates a test category and returns the ID
func createTestCategory(name, description, color string) (string, error) {
	var descText pgtype.Text
	var colorText pgtype.Text

	if description != "" {
		descText = pgtype.Text{String: description, Valid: true}
	}
	if color != "" {
		colorText = pgtype.Text{String: color, Valid: true}
	}

	category, err := testQueries.CreateCategory(context.Background(), generated.CreateCategoryParams{
		Name:        name,
		Description: descText,
		Color:       colorText,
	})
	if err != nil {
		return "", err
	}

	return category.ID.String(), nil
}

// makeRequest helper function for making HTTP requests
func makeRequest(method, url string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	testRouter.ServeHTTP(recorder, req)

	return recorder
}

// makeMultipartRequest helper function for making multipart requests (file uploads)
func makeMultipartRequest(url string, fieldName, fileName string, fileContent []byte) *httptest.ResponseRecorder {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		panic(err)
	}

	part.Write(fileContent)
	writer.Close()

	req := httptest.NewRequest("POST", url, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	testRouter.ServeHTTP(recorder, req)

	return recorder
}

// parseJSONResponse helper function to parse JSON response
func parseJSONResponse(recorder *httptest.ResponseRecorder, target interface{}) error {
	return json.Unmarshal(recorder.Body.Bytes(), target)
}

// assertStatusCode helper function to assert HTTP status code
func assertStatusCode(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected status code %d, got %d", expected, actual)
	}
}

// assertNoError helper function to assert no error occurred
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// assertError helper function to assert an error occurred
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected an error, but got nil")
	}
}
