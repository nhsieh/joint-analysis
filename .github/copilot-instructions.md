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
- `docker-compose.yml` - Local development environment
- `README.md` - Project documentation

## Features
- CSV file upload and parsing
- Purchase assignment to individuals or joint groups
- Total calculation per person
- REST API for data management
- PostgreSQL database for transaction storage

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

## Development Standards
For comprehensive coding standards, workflows, and AI agent instructions, see:
- **[Development Guidelines](development-guidelines.md)** - Complete TDD methodology, code quality standards, and workflows
- **[AI Agent Configuration](agents/README.md)** - Multi-agent workflows and specialized agent prompts

**Important**: All development work must follow the Test-Driven Development (TDD) practices and quality standards defined in the development guidelines.