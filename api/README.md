# Shopify App Template - API

This is the API backend for the Shopify App Template built with Go.

## Database Support

The application supports both PostgreSQL and SQLite databases, configurable via environment variables.

### Configuration

The database type is controlled by the `DATABASE_TYPE` environment variable:

#### PostgreSQL (default)
```bash
# Uses PostgreSQL by default
./main

# Or explicitly set
DATABASE_TYPE=postgres ./main
```

**Environment Variables:**
- `POSTGRES_USER` - Database user (default: "postgres")
- `POSTGRES_PASSWORD` - Database password (default: "postgres")
- `POSTGRES_HOST` - Database host (default: "localhost")
- `POSTGRES_DATABASE` - Database name (default: "api")

#### SQLite
```bash
# Use SQLite
DATABASE_TYPE=sqlite ./main

# Custom SQLite file path
DATABASE_TYPE=sqlite SQLITE_PATH=./custom.db ./main
```

**Environment Variables:**
- `SQLITE_PATH` - Path to SQLite database file (default: "./app.db")

### Migrations

Migrations are automatically run on startup. The system uses different migration files for each database type:

- **PostgreSQL**: `migrations/*.sql`
- **SQLite**: `migrations/sqlite/*.sql`

### Building and Running

```bash
# Build the application
go build ./cmd/main.go

# Run with PostgreSQL (requires running PostgreSQL instance)
DATABASE_TYPE=postgres ./main

# Run with SQLite (no external dependencies)
DATABASE_TYPE=sqlite ./main
```

### Development

The application uses:
- PostgreSQL driver: `github.com/jackc/pgx/v5`
- SQLite driver: `github.com/mattn/go-sqlite3`
- Migrations: `github.com/golang-migrate/migrate/v4`

Database operations are abstracted through a common interface, ensuring compatibility across both database systems.