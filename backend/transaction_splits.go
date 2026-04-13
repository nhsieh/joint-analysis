package main

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"jointanalysis/db/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type splitRequest struct {
	Splits []struct {
		Amount     float64 `json:"amount"`
		CategoryID string  `json:"category_id"`
		Notes      *string `json:"notes"`
	} `json:"splits"`
}

func convertTransactionSplitRow(s generated.TransactionSplit) TransactionSplit {
	result := TransactionSplit{
		ID:            uuid.UUID(s.ID.Bytes).String(),
		TransactionID: uuid.UUID(s.TransactionID.Bytes).String(),
		CategoryID:    uuid.UUID(s.CategoryID.Bytes).String(),
		CreatedAt:     s.CreatedAt.Time,
		UpdatedAt:     s.UpdatedAt.Time,
	}

	if amountValue, err := s.Amount.Float64Value(); err == nil {
		result.Amount = amountValue.Float64
	}
	if s.Notes.Valid {
		result.Notes = &s.Notes.String
	}

	return result
}

func loadTransactionSplits(transactionID pgtype.UUID, amount pgtype.Numeric, categoryID pgtype.UUID) ([]TransactionSplit, error) {
	splits, err := queries.GetTransactionSplitsByTransactionID(context.Background(), transactionID)
	if err != nil {
		return nil, err
	}

	if len(splits) == 0 {
		if !categoryID.Valid {
			return []TransactionSplit{}, nil
		}

		amountValue, _ := amount.Float64Value()
		defaultSplit := TransactionSplit{
			ID:            "",
			TransactionID: uuid.UUID(transactionID.Bytes).String(),
			Amount:        math.Abs(amountValue.Float64),
			CategoryID:    uuid.UUID(categoryID.Bytes).String(),
		}
		return []TransactionSplit{defaultSplit}, nil
	}

	result := make([]TransactionSplit, 0, len(splits))
	for _, split := range splits {
		result = append(result, convertTransactionSplitRow(split))
	}

	return result, nil
}

// @Summary Get transaction splits
// @Description Retrieve split rows for a transaction. Returns a default single split derived from transaction category/amount when no explicit splits are stored.
// @Tags transactions
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 200 {array} TransactionSplit "List of splits"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Transaction not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/transactions/{id}/splits [get]
func getTransactionSplits(c *gin.Context) {
	id := c.Param("id")
	transactionUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	transactionID := pgtype.UUID{Bytes: transactionUUID, Valid: true}

	tx, err := queries.GetTransactionByID(context.Background(), transactionID)
	if err != nil {
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	splits, err := loadTransactionSplits(transactionID, tx.Amount, tx.CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching transaction splits"})
		return
	}

	c.JSON(http.StatusOK, splits)
}

// @Summary Replace transaction splits
// @Description Replace all split rows for a transaction and sync transaction category to the first split for backward compatibility.
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param payload body splitRequest true "Split rows"
// @Success 200 {array} TransactionSplit "Updated splits"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Transaction not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/transactions/{id}/splits [put]
func replaceTransactionSplits(c *gin.Context) {
	id := c.Param("id")
	transactionUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var request splitRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if len(request.Splits) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one split is required"})
		return
	}

	transactionID := pgtype.UUID{Bytes: transactionUUID, Valid: true}
	tx, err := queries.GetTransactionByID(context.Background(), transactionID)
	if err != nil {
		statusCode, message := handleDatabaseError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	totalAbs := 0.0
	if amountValue, err := tx.Amount.Float64Value(); err == nil {
		totalAbs = math.Abs(amountValue.Float64)
	}

	sum := 0.0
	validatedCategoryUUIDs := make([]uuid.UUID, 0, len(request.Splits))
	for _, split := range request.Splits {
		if split.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All split amounts must be positive"})
			return
		}
		if split.CategoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required for every split"})
			return
		}
		categoryUUID, err := uuid.Parse(split.CategoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		validatedCategoryUUIDs = append(validatedCategoryUUIDs, categoryUUID)
		sum += split.Amount
	}

	if math.Abs(sum-totalAbs) > 0.01 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Split amounts must equal the absolute transaction amount"})
		return
	}

	if err := queries.DeleteTransactionSplitsByTransactionID(context.Background(), transactionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error replacing transaction splits"})
		return
	}

	created := make([]TransactionSplit, 0, len(request.Splits))
	for i, split := range request.Splits {
		categoryUUID := validatedCategoryUUIDs[i]

		var amountNumeric pgtype.Numeric
		if err := amountNumeric.Scan(fmt.Sprintf("%.2f", split.Amount)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid split amount"})
			return
		}

		notes := pgtype.Text{Valid: false}
		if split.Notes != nil {
			notes = pgtype.Text{String: *split.Notes, Valid: true}
		}

		row, err := queries.CreateTransactionSplit(context.Background(), generated.CreateTransactionSplitParams{
			TransactionID: transactionID,
			Amount:        amountNumeric,
			CategoryID:    pgtype.UUID{Bytes: categoryUUID, Valid: true},
			Notes:         notes,
		})
		if err != nil {
			statusCode, message := handleDatabaseError(err)
			c.JSON(statusCode, gin.H{"error": message})
			return
		}
		created = append(created, convertTransactionSplitRow(row))

		if i == 0 {
			// Keep legacy category_id aligned to first split for backward compatibility.
			_, err := queries.UpdateTransactionCategory(context.Background(), generated.UpdateTransactionCategoryParams{
				ID:         transactionID,
				CategoryID: pgtype.UUID{Bytes: categoryUUID, Valid: true},
			})
			if err != nil {
				statusCode, message := handleDatabaseError(err)
				c.JSON(statusCode, gin.H{"error": message})
				return
			}
		}
	}

	c.JSON(http.StatusOK, created)
}
