# Joint Analysis - Expense Tracking Application

A full-stack expense tracking application that allows users to upload CSV files and assign purchases to different people, with totals calculated per person.

## Features

- **CSV File Upload**: Upload expense data from CSV files
- **Person Management**: Add and manage people who make purchases
- **Transaction Assignment**: Assign each transaction to a specific person
- **Automatic Totals**: Calculate and display total expenses per person
- **Real-time Updates**: Live updates when assignments change
- **Archive**: See totals and transactions in archive

## Tech Stack

- **Frontend**: React with TypeScript
- **Backend**: Go (Golang) with Gin framework
- **Database**: PostgreSQL
- **Containerization**: Docker & Docker Compose

## Project Structure

```
joint-analysis/
├── AGENTS.md                 # AI agent configuration and instructions
├── Makefile                  # Build automation and common tasks
├── README.md                 # Project documentation
├── docker-compose.yml       # Local development environment setup
├── frontend/                 # React application
│   ├── src/
│   │   └── App.tsx          # Main React component
│   ├── public/
│   ├── package.json
│   └── Dockerfile
├── backend/                  # Go API server
│   ├── main.go              # Main server file and API routes
│   ├── models.go            # Data models and structures
│   ├── people.go            # People management endpoints
│   ├── transactions.go      # Transaction management endpoints
│   ├── categories.go        # Category management endpoints
│   ├── archives.go          # Archive functionality endpoints
│   ├── totals.go            # Total calculation endpoints
│   ├── utils.go             # Utility functions
│   ├── migrations.go        # Database migration handling
│   ├── go.mod               # Go module dependencies
│   ├── *_test.go            # Test files for each module
│   ├── db/                  # Database layer
│   │   ├── query.sql        # SQL queries for SQLC
│   │   ├── generated/       # SQLC generated code
│   │   └── migrations/      # Database migration files (*.up.sql, *.down.sql)
│   ├── docs/                # API documentation (Swagger)
│   └── Dockerfile           # Backend container config
├── docs/                     # Project documentation
│   └── adr/                 # Architectural Decision Records
│       └── *.md             # Decision documents
└── references/              # Reference files and examples
```

## Quick Start

### Prerequisites

- Docker and Docker Compose installed
- Git

### Running the Application

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd joint-analysis
   ```

2. **Start the application**:
   ```bash
   docker-compose up --build
   ```

3. **Access the application**:
   - Frontend: http://localhost:3001
   - Backend API: http://localhost:8081
   - PostgreSQL: localhost:5433

### CSV File Format

The application expects CSV files with the following format:
```csv
Transaction Date,Posted Date,Card No.,Description,Category,Debit,Credit
2025-10-17,2025-10-20,9364,RIVER INN VALERO,Gas/Automotive,26.45,
2025-10-16,2025-10-18,9364,STARBUCKS #12345,Food/Dining,5.75,
2025-10-15,2025-10-17,9364,GROCERY OUTLET,Groceries,45.20,
```

**Required columns:**
- `Transaction Date` - Date of transaction
- `Posted Date` - Date transaction was posted
- `Card No.` - Card number used
- `Description` - Transaction description
- `Category` - Transaction category

**Optional columns (at least one must have a value):**
- `Debit` - Debit amount
- `Credit` - Credit amount

## Usage

1. **Add People**: Use the "Add Person" section to create people who make purchases
2. **Upload CSV**: Upload your expense CSV file using the upload section
3. **Assign Transactions**: Use the dropdown in each transaction row to assign it to a person
4. **View Totals**: The totals section automatically updates to show each person's total expenses
5. **Archive**: Moves transactions and totals into archive

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

This project is licensed under the MIT License.