#!/bin/bash

# Database shell connection script
# Usage: ./db-shell.sh [source|target] [query|shell]
# Examples:
#   ./db-shell.sh source          # Interactive shell
#   ./db-shell.sh source shell    # Interactive shell
#   ./db-shell.sh source query    # Run a test query and exit

set -e

# Load environment variables from .env file
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo "Error: .env file not found"
    exit 1
fi

SERVICE="mysql-source"
if [ "$1" = "target" ]; then
    SERVICE="mysql-target"
fi

if [ "$2" = "query" ]; then
    echo "Testing connection to $SERVICE database..."
    docker-compose exec -T $SERVICE mysql -u root -p$MYSQL_ROOT_PASSWORD $MYSQL_DATABASE -e "SELECT 'Connection successful' as status, @@version as mysql_version, @@server_uuid as server_uuid;"
else
    echo "Connecting to $SERVICE database (interactive shell)..."
    echo "Type 'exit' or Ctrl+D to quit"
    docker-compose exec $SERVICE mysql -u root -p$MYSQL_ROOT_PASSWORD $MYSQL_DATABASE
fi