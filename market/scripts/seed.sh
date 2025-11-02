#!/bin/bash

# Seed the database with test data

set -e

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL environment variable is not set"
    echo "Example: export DATABASE_URL=\"postgres://user:password@localhost:5432/aegis?sslmode=disable\""
    exit 1
fi

echo "Seeding database..."
echo "Database: $DATABASE_URL"

# Run the seed file
psql "$DATABASE_URL" -f "$(dirname "$0")/seed.sql"

echo ""
echo "✅ Database seeded successfully!"
echo ""
