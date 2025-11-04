package main

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertUUIDStringsToArray(t *testing.T) {
	t.Run("empty array returns empty slice", func(t *testing.T) {
		result, err := convertUUIDStringsToArray([]string{})

		require.NoError(t, err)
		assert.Equal(t, []pgtype.UUID{}, result)
		assert.Len(t, result, 0)
	})

	t.Run("nil array returns empty slice", func(t *testing.T) {
		result, err := convertUUIDStringsToArray(nil)

		require.NoError(t, err)
		assert.Equal(t, []pgtype.UUID{}, result)
		assert.Len(t, result, 0)
	})

	t.Run("single valid UUID string returns pgtype.UUID", func(t *testing.T) {
		testUUID := uuid.New()
		uuidString := testUUID.String()

		result, err := convertUUIDStringsToArray([]string{uuidString})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, testUUID, uuid.UUID(result[0].Bytes))
		assert.True(t, result[0].Valid)
	})

	t.Run("multiple valid UUID strings return pgtype.UUIDs", func(t *testing.T) {
		uuid1 := uuid.New()
		uuid2 := uuid.New()
		uuidStrings := []string{uuid1.String(), uuid2.String()}

		result, err := convertUUIDStringsToArray(uuidStrings)

		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Convert back to UUIDs for verification
		resultUUIDs := []uuid.UUID{
			uuid.UUID(result[0].Bytes),
			uuid.UUID(result[1].Bytes),
		}

		assert.Contains(t, resultUUIDs, uuid1)
		assert.Contains(t, resultUUIDs, uuid2)
		assert.True(t, result[0].Valid)
		assert.True(t, result[1].Valid)
	})

	t.Run("invalid UUID string returns error", func(t *testing.T) {
		invalidUUIDString := "not-a-valid-uuid"

		result, err := convertUUIDStringsToArray([]string{invalidUUIDString})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid UUID format")
		assert.Contains(t, err.Error(), invalidUUIDString)
	})

	t.Run("first invalid UUID returns error without processing remaining", func(t *testing.T) {
		validUUID := uuid.New().String()
		invalidUUID := "invalid-uuid"

		result, err := convertUUIDStringsToArray([]string{invalidUUID, validUUID})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid UUID format")
		assert.Contains(t, err.Error(), invalidUUID)
	})

	t.Run("empty string UUID returns error", func(t *testing.T) {
		result, err := convertUUIDStringsToArray([]string{""})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid UUID format")
	})

	t.Run("malformed UUID string returns error", func(t *testing.T) {
		malformedUUID := "123e4567-e89b-12d3-a456-42661417400" // Missing one character

		result, err := convertUUIDStringsToArray([]string{malformedUUID})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid UUID format")
		assert.Contains(t, err.Error(), malformedUUID)
	})

	t.Run("UUID with wrong format returns error", func(t *testing.T) {
		wrongFormatUUID := "123e4567e89b12d3a45642661417400g" // No hyphens and invalid character

		result, err := convertUUIDStringsToArray([]string{wrongFormatUUID})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid UUID format")
		assert.Contains(t, err.Error(), wrongFormatUUID)
	})

	t.Run("mixed valid and invalid UUIDs - fails on first invalid", func(t *testing.T) {
		validUUID1 := uuid.New().String()
		invalidUUID := "invalid"
		validUUID2 := uuid.New().String()

		result, err := convertUUIDStringsToArray([]string{validUUID1, invalidUUID, validUUID2})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid UUID format")
		assert.Contains(t, err.Error(), invalidUUID)
	})

	t.Run("uppercase UUID strings are handled correctly", func(t *testing.T) {
		testUUID := uuid.New()
		uppercaseUUIDString := strings.ToUpper(testUUID.String())

		result, err := convertUUIDStringsToArray([]string{uppercaseUUIDString})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, testUUID, uuid.UUID(result[0].Bytes))
		assert.True(t, result[0].Valid)
	})

	t.Run("mixed case UUID strings are handled correctly", func(t *testing.T) {
		testUUID := uuid.New()
		// Mix of upper and lower case
		mixedCaseUUID := testUUID.String()
		mixedCaseUUID = strings.ToUpper(mixedCaseUUID[:8]) + mixedCaseUUID[8:]

		result, err := convertUUIDStringsToArray([]string{mixedCaseUUID})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, testUUID, uuid.UUID(result[0].Bytes))
		assert.True(t, result[0].Valid)
	})
}
