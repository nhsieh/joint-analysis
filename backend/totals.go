package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Totals handler functions

// @Summary Get totals by person
// @Description Get calculated expense totals for each person from active transactions
// @Tags totals
// @Produce json
// @Success 200 {array} Total "List of totals by person"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/totals [get]
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