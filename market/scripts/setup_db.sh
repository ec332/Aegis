# Setup local PostgreSQL database for development

set -e

# Default configuration
DB_NAME="${DB_NAME:-aegis}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

echo "Setting up database: $DB_NAME"

# Check if database exists
if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -lqt | cut -d \| -f 1 | grep -qw $DB_NAME; then
    echo "Database $DB_NAME already exists"
else
    echo "Creating database $DB_NAME..."
    createdb -h $DB_HOST -p $DB_PORT -U $DB_USER $DB_NAME
    echo "Database created successfully"
fi

# Export DATABASE_URL for the application
export DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"

echo "Database URL: $DATABASE_URL"
echo ""
echo "To use this database, set the following environment variable:"
echo "export DATABASE_URL=\"$DATABASE_URL\""
echo ""
echo "To seed the database with test data, run:"
echo "psql $DATABASE_URL -f scripts/seed.sql"
