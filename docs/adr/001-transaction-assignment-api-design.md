# ADR-001: Transaction Assignment API Design - UUIDs vs Names

## Status
Accepted

## Context
The transaction assignment API needs to handle the `assigned_to` field in a way that balances data integrity, user experience, and API consistency. We had to decide how to handle the representation of assigned people in both API requests and responses.

The key considerations were:
1. **Data Integrity**: Database should use UUIDs for foreign key relationships to ensure referential integrity
2. **User Experience**: API responses should be human-readable and user-friendly
3. **API Consistency**: The input/output format should be intuitive for API consumers
4. **Performance**: Minimize database queries while maintaining good UX

### Technical Context
- Backend: Golang with Gin framework
- Database: PostgreSQL with UUID primary keys
- API consumers: React frontend and potentially other clients
- Database layer: SQLC-generated queries with pgtype for PostgreSQL types

## Decision
We decided to implement an **asymmetric API design** for the transaction assignment functionality:

### Input (Request):
- **PUT /api/transactions/:id/assign** accepts an array of person **UUIDs** in the `assigned_to` field
- Example: `{"assigned_to": ["550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"]}`

### Output (Response):
- All transaction endpoints return an array of person **names** in the `assigned_to` field
- Example: `{"assigned_to": ["John Doe", "Jane Smith"]}`

### Storage:
- Database stores UUIDs in the `assigned_to` array column for referential integrity
- Conversion happens in the API layer via helper functions

## Implementation Details

### Key Functions:
1. **convertUUIDStringsToArray()**: Converts UUID strings from API requests to pgtype.UUID array for database storage
2. **convertUUIDArrayToNames()**: Converts UUID array from database to person names for API responses
3. **convertTransactionFromFields()**: Orchestrates the conversion in all transaction response scenarios

### API Behavior:
- **Request**: Client sends UUIDs (precise, unambiguous references)
- **Response**: Client receives names (human-readable, user-friendly)
- **Database**: Stores UUIDs (maintains referential integrity)

## Consequences

### Positive:
1. **Data Integrity**: UUIDs in database ensure proper foreign key relationships and prevent orphaned references
2. **User Experience**: API responses show readable person names instead of cryptic UUIDs
3. **Frontend Simplicity**: React components can display names directly without additional lookups
4. **Referential Safety**: Person renames don't break existing transaction assignments
5. **API Precision**: UUID input ensures unambiguous person identification

### Negative:
1. **API Asymmetry**: Input and output formats differ, which may be unexpected for some API consumers
2. **Performance Overhead**: Each transaction response requires person name lookups from UUIDs
3. **Complexity**: Requires conversion logic in multiple places
4. **Error Handling**: Must handle cases where UUIDs exist in database but persons are deleted
5. **Documentation Need**: API consumers must understand the asymmetric design

### Risk Mitigation:
- **Performance**: Helper functions include error handling to skip invalid UUIDs rather than fail completely
- **Data Consistency**: Database constraints ensure UUIDs are valid when stored
- **Error Recovery**: API gracefully handles missing persons by logging and continuing rather than failing

## Alternatives Considered

### Alternative 1: UUID-only API
- **Input**: UUIDs
- **Output**: UUIDs
- **Rejected**: Poor user experience, frontend would need constant person lookups

### Alternative 2: Name-only API
- **Input**: Names
- **Output**: Names
- **Rejected**: Fragile to person renames, potential ambiguity with duplicate names

### Alternative 3: Full Person Objects
- **Input**: Person objects with UUIDs
- **Output**: Full person objects
- **Rejected**: Over-fetching, increased payload size, unnecessary complexity for simple assignments

## Implementation Notes

### Error Handling:
```go
// Graceful degradation - skip invalid UUIDs rather than fail
if err != nil {
    log.Printf("Error getting person by ID %v: %v", uuidPg, err)
    continue // Skip invalid UUIDs instead of failing completely
}
```

### Conversion Pattern:
```go
// Database → API Response
names, err := convertUUIDArrayToNames(assignedTo)
if err != nil {
    log.Printf("Error converting UUIDs to names: %v", err)
} else {
    result.AssignedTo = names
}
```

## Future Considerations

1. **Caching**: Consider caching person UUID→name mappings to reduce database queries
2. **Batch Queries**: If performance becomes an issue, implement batch person lookups
3. **API Evolution**: Future API versions could offer both formats via query parameters
4. **Consistency**: Other entity relationships should follow similar patterns

## Related Decisions
- Database schema design (UUID primary keys)
- API response format standardization
- Error handling patterns across endpoints

---
**Date**: October 29, 2025
**Participants**: Development Team
**Supersedes**: None
**Superseded by**: None