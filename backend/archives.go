package main

import (
	"context"
	"log"
	"math/big"
	"net/http"

	"jointanalysis/db/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Archive handler functions

// @Summary Create archive
// @Description Create a new archive of all current active transactions
// @Tags archives
// @Accept json
// @Produce json
// @Param archive body ArchiveRequest true "Archive data with description"
// @Success 201 {object} Archive "Created archive with transaction totals"
// @Failure 400 {object} map[string]interface{} "Bad request (no transactions to archive or invalid data)"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/archives [post]
func createArchive(c *gin.Context) {
	var request ArchiveRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
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

// @Summary Get all archives
// @Description Retrieve all archives from the database with their person totals
// @Tags archives
// @Produce json
// @Success 200 {array} Archive "List of archives with transaction counts and person totals"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/archives [get]
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

// @Summary Get archive transactions
// @Description Get all transactions for a specific archive by archive ID
// @Tags archives
// @Produce json
// @Param id path string true "Archive ID"
// @Success 200 {array} Transaction "List of archived transactions"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Archive not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/archives/{id}/transactions [get]
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
