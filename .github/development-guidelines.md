# GitHub Copilot Agent Instructions

This document defines the coding standards, workflows, and best practices for the Joint Analysis expense tracking application. All AI agents and developers working on this repository should follow these guidelines.

## Project Overview
- **Frontend**: React with TypeScript
- **Backend**: Golang REST API
- **Database**: PostgreSQL
- **Development**: Docker Compose for local environment

## AI Agent Configuration

This project supports both single-purpose AI agents and multi-agent workflows. All agent configurations are stored in `.github/agents/`.

### Agent Structure
- **Individual Agents**: Specialized prompts for specific roles (backend, frontend, testing, review)
- **Multi-Agent Workflows**: Coordinated workflows for complex tasks (feature development, bug fixes, releases)
- **Configuration**: See `.github/agents/README.md` for detailed documentation

### Agent Usage Guidelines
- All agents must follow the TDD methodology and code quality standards defined in this document
- Use specialized agents for focused expertise in specific areas
- Use multi-agent workflows for complex, cross-functional tasks
- Maintain clear handoffs and communication between agents in workflows

## Development Methodology

### Test-Driven Development (TDD)
**MANDATORY**: All code changes must follow strict TDD practices:

1. **Red Phase**: Write a failing test first
2. **Green Phase**: Write minimal code to make the test pass
3. **Refactor Phase**: Clean up code while keeping tests green

**Rules**:
- No production code without a failing test first
- Tests must fail for the right reason before implementation
- Only write enough code to make the current test pass
- Refactor only after tests are green

### Testing Standards
- **Backend (Go)**: Use Go's built-in testing framework
- **Frontend (React)**: Use established testing libraries (Jest, React Testing Library, etc.)
- **Test Coverage**: Aim for high coverage, but focus on meaningful tests
- **Test Naming**: Use descriptive names that explain the behavior being tested

## Git Workflow

### Commit Standards
- **Frequency**: Commit after completing related changes (can group multiple prompt completions)
- **Atomic Commits**: Each commit should represent a complete, working change
- **TDD Cycle**: Consider committing after completing a full TDD cycle (red-green-refactor)

### Commit Message Format
Use clear, descriptive commit messages:
```
type(scope): description

- feat: new feature
- fix: bug fix
- test: adding or updating tests
- refactor: code refactoring
- docs: documentation changes
- style: formatting, linting
```

## Code Quality Standards

### Linting and Formatting
- **Frontend**: Use `npm run lint` - must pass before commits
- **Backend**: Use `golangci-lint` - must pass before commits
- **Auto-fix**: Run formatters/linters and fix issues before committing

### Error Handling
- **Backend (Go)**:
  - **ALWAYS** check and handle errors explicitly
  - Never ignore errors with `_` unless absolutely justified
  - Use proper error wrapping and context
  - Validate all inputs - never trust frontend validation
- **Frontend**:
  - Implement client-side validation for UX
  - Handle API errors gracefully
  - Show meaningful error messages to users

## Documentation Requirements

### Inline Documentation
- **Go**: Use godoc-style comments for exported functions, types, and packages
- **TypeScript/React**: Use JSDoc comments for complex functions and components
- **SQL**: Comment complex queries and schema changes

### Architecture Documentation
- **Location**: All architectural decisions go in `docs/adr/` directory as Architectural Decision Records (ADRs)
- **When Required**:
  - New features that change system architecture
  - Database schema modifications
  - API design changes
  - Integration with external services
- **Format**: Plain text Markdown files using standardized ADR structure:
  - **Title**: A concise title for the decision
  - **Context**: The problem or situation that prompted the decision
  - **Decision**: The chosen solution and how it was implemented
  - **Status**: The current state (proposed, accepted, deprecated, superseded)
  - **Consequences**: The positive and negative implications of the decision

## Code Organization

### File Structure
- Follow existing project structure
- Group related functionality together
- Use clear, descriptive file and directory names

### Naming Conventions
- **Go**: Use standard Go conventions (PascalCase for exported, camelCase for unexported)
- **TypeScript**: Use PascalCase for components, camelCase for functions/variables
- **Database**: Use snake_case for table and column names

## Backend-Specific Guidelines

### API Development
- Validate all inputs at API boundaries
- Use proper HTTP status codes
- Implement consistent error response format
- Never trust frontend validation - always validate server-side

### Database Operations
- Use transactions for multi-step operations
- Handle database errors appropriately
- Use prepared statements to prevent SQL injection
- Keep migrations reversible when possible

## Frontend-Specific Guidelines

### React Development
- Use TypeScript for all components
- Implement proper error boundaries
- Handle loading and error states
- Validate user inputs before API calls

### State Management
- Keep state as close to where it's used as possible
- Use proper TypeScript types for all state

## Development Workflow

### Before Starting Work
1. Ensure Docker Compose environment is running
2. Run existing tests to ensure clean baseline
3. Create failing test for new functionality

### During Development
1. Follow TDD cycle: Red → Green → Refactor
2. Run linters frequently
3. Keep commits atomic and well-described

### Before Committing
1. Run full test suite
2. Run linters and fix any issues
3. Update documentation if needed
4. Ensure all tests pass

### For Architectural Changes
1. Create an ADR (Architectural Decision Record) in `docs/adr/` directory
2. Update relevant README sections
3. Consider impact on existing functionality
4. Ensure backward compatibility when possible

## Testing Strategy

### Unit Tests
- Test individual functions/methods in isolation
- Mock external dependencies
- Focus on edge cases and error conditions

### Integration Tests
- Test API endpoints end-to-end
- Test database operations
- Test component interactions

### Error Case Testing
- Test all error paths
- Verify proper error handling and responses
- Test validation failures

## Environment Configuration

### Local Development
- Use Docker Compose for consistent environment
- Environment variables for configuration
- Separate configurations for test/dev/prod

### Port Configuration
- Frontend: http://localhost:3001
- Backend API: http://localhost:8081
- PostgreSQL: localhost:5433

## Common Patterns to Follow

### Backend Error Handling Pattern
```go
result, err := someOperation()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Frontend Error Handling Pattern
```typescript
try {
    const result = await apiCall();
    // handle success
} catch (error) {
    // handle error appropriately
    console.error('Operation failed:', error);
}
```

---

**Remember**: These standards exist to maintain code quality, ensure reliability, and make the codebase maintainable. When in doubt, prioritize clarity, testability, and proper error handling.