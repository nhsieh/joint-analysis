<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

# Joint Analysis - Expense Tracking Application

This is a full-stack expense tracking application that allows users to upload CSV files and assign purchases to different people, with totals calculated per person.

## Tech Stack
- **Frontend**: React with TypeScript
- **Backend**: Golang REST API
- **Database**: PostgreSQL
- **Deployment**: Docker Compose for local development

## Project Structure
- `frontend/` - React application
- `backend/` - Golang API server
- `docs/adr/` - Architectural decisions records
- `docker-compose.yml` - Local development environment
- `README.md` - Project documentation
- `Makefile` - Useful commands

## Features
- CSV file upload and parsing
- Purchase assignment to individuals or joint groups
- Total calculation per person
- REST API for data management
- PostgreSQL database for transaction storage
- **Automated API documentation** with Swagger/OpenAPI

## API Documentation
- **Interactive Swagger UI**: Available at http://localhost:8081/swagger/index.html when running
- **Auto-generated**: Documentation is generated from code annotations using swaggo/swag
- **Always up-to-date**: Regenerated from actual API endpoints and models
- **Commands**:
  - `make generate-docs` - Generate Swagger documentation from code
- **Documentation files**: `backend/docs/` for generated Swagger files

## Development Instructions
1. Use Docker Compose for local development: `docker-compose up`
2. Frontend runs on http://localhost:3001
3. Backend API runs on http://localhost:8081
4. PostgreSQL database on port 5433

## Code Guidelines
- Use TypeScript for React components
- Follow Go best practices for API development
- Implement proper error handling
- Use environment variables for configuration
- **API Documentation**: Add Swagger annotations to all new API endpoints
- **Documentation Updates**: Run `make generate-docs` after API changes to keep documentation current

## Development Standards
For comprehensive coding standards, workflows, and AI agent instructions, see:
- **[Development Guidelines](development-guidelines.md)** - Complete TDD methodology, code quality standards, and workflows
- **[AI Agent Configuration](agents/README.md)** - Multi-agent workflows and specialized agent prompts

**Important**: All development work must follow the Test-Driven Development (TDD) practices and quality standards defined in the development guidelines.