package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math"
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

	// Track how many times each dedup key appears in the current CSV file.
	// This allows multiple identical rows in the same CSV to all be imported
	// while still preventing re-import of rows that already exist in the DB.
	seenCounts := make(map[string]int64)

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

		// Parse amount from Debit (positive) or Credit (negative) columns
		if record[5] != "" {
			// Debit amount - keep as positive (expense)
			amount, err = strconv.ParseFloat(record[5], 64)
		} else if record[6] != "" {
			// Credit amount - make negative (income/refund)
			amount, err = strconv.ParseFloat(record[6], 64)
			amount = -amount // Make credits negative
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

		selectedCategory := (*generated.GetCategoriesRow)(nil)

		// Map category if category mapping is available
		if categoryMapping != nil {
			if mappedCategory := categoryMapping.mapTransactionCategory(description, csvCategory); mappedCategory != nil {
				selectedCategory = mappedCategory
			}
		}

		if selectedCategory == nil && categoryMapping != nil {
			if fallback, exists := categoryMapping.categoriesByName["Other"]; exists {
				selectedCategory = &fallback
			}
		}

		if selectedCategory == nil || !selectedCategory.ID.Valid {
			skippedRows++
			continue
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

		// Build a dedup key from all identifying fields and track how many
		// times this row has appeared so far in the current CSV file.
		dedupKey := fmt.Sprintf("%s|%s|%s|%s|%s",
			description,
			params.Amount.Int.String()+"e"+strconv.Itoa(int(params.Amount.Exp)),
			params.TransactionDate.Time.Format("2006-01-02"),
			params.PostedDate.Time.Format("2006-01-02"),
			params.CardNumber.String,
		)
		seenCounts[dedupKey]++

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

		// Skip only if the DB already has at least as many copies as we've
		// seen so far in this CSV. This lets identical rows within one CSV
		// all be imported on first upload while still preventing re-import.
		if count >= seenCounts[dedupKey] {
			log.Printf("Skipping duplicate transaction: %s, amount: %f", description, amount)
			skippedRows++
			continue
		}

		createdTransaction, err := queries.CreateTransaction(context.Background(), params)
		if err != nil {
			log.Printf("Error inserting transaction: %v", err)
			skippedRows++
			continue
		}

		splitAmount := math.Abs(amount)
		var splitNumeric pgtype.Numeric
		if err := splitNumeric.Scan(fmt.Sprintf("%.2f", splitAmount)); err != nil {
			log.Printf("Error converting split amount to numeric: %v", err)
			skippedRows++
			continue
		}

		_, err = queries.CreateTransactionSplit(context.Background(), generated.CreateTransactionSplitParams{
			TransactionID: createdTransaction.ID,
			Amount:        splitNumeric,
			CategoryID:    selectedCategory.ID,
			Notes:         pgtype.Text{Valid: false},
		})
		if err != nil {
			log.Printf("Error inserting transaction split: %v", err)
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
		transaction := convertTransactionFromActiveRow(t)

		splits, err := loadTransactionSplits(t.ID)
		if err != nil {
			log.Printf("Error loading splits for transaction %s: %v", transaction.ID, err)
		} else {
			transaction.Splits = splits
		}

		transactions = append(transactions, transaction)
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
