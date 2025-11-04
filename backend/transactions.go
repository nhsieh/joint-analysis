package main

import (
	"context"
	"encoding/csv"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"jointanalysis/db/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Transaction handler functions

// @Summary Upload CSV file
// @Description Upload a CSV file containing transaction data. Returns the successfully imported transactions and count of skipped rows.
// @Tags transactions
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV file to upload"
// @Success 200 {object} map[string]interface{} "Upload successful - returns message, transactions array, and skipped_rows count"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/upload-csv [post]
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
	skippedRows := 0

	// Skip header row if present
	start := 0
	if len(records) > 0 && records[0][0] == "Transaction Date" {
		start = 1
	}

	for i := start; i < len(records); i++ {
		record := records[i]
		if len(record) < 7 { // Need exactly 7 columns for CSV format: Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
			skippedRows++
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
			skippedRows++
			continue // Skip if no amount found
		}

		if err != nil {
			skippedRows++
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
			skippedRows++
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
			skippedRows++
			continue
		}

		// If duplicate exists, skip this transaction
		if count > 0 {
			log.Printf("Skipping duplicate transaction: %s, amount: %f", description, amount)
			skippedRows++
			continue
		}

		_, err = queries.CreateTransaction(context.Background(), params)
		if err != nil {
			log.Printf("Error inserting transaction: %v", err)
			skippedRows++
			continue
		}

		transactions = append(transactions, transaction)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "CSV uploaded successfully",
		"transactions": transactions,
		"skipped_rows": skippedRows,
	})
}

// @Summary Get all transactions
// @Description Retrieve all active (non-archived) transactions from the database
// @Tags transactions
// @Produce json
// @Success 200 {array} Transaction "List of transactions"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/transactions [get]
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

// @Summary Assign transaction to person
// @Description Assign a specific transaction to one or more people
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param assignment body object{assigned_to=[]string} true "Assignment data with array of person names"
// @Success 200 {object} Transaction "Updated transaction with assignments"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Transaction not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/transactions/{id}/assign [put]
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

// @Summary Delete single transaction
// @Description Delete a specific transaction by ID
// @Tags transactions
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 200 {object} map[string]interface{} "Transaction deleted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Transaction not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/transactions/{id} [delete]
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

// @Summary Delete all transactions
// @Description Clear all active transactions from the database
// @Tags transactions
// @Produce json
// @Success 200 {object} map[string]interface{} "All transactions cleared successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/transactions [delete]
func clearAllTransactions(c *gin.Context) {
	err := queries.DeleteAllTransactions(context.Background())
	if err != nil {
		log.Printf("Error clearing all transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error clearing transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All transactions cleared successfully"})
}

// @Summary Update transaction category
// @Description Update the category assignment for a specific transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param category body object{category_id=string} true "Category assignment data"
// @Success 200 {object} Transaction "Updated transaction with new category"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Transaction not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/transactions/{id}/category [put]
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
