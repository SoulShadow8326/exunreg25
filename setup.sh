echo "Setting up database..."
DB_PATH="./data/exunreg25.db"

DB_PATH="${DB_PATH:-./data/data.db}"

mkdir -p "$(dirname "$DB_PATH")"
if [ ! -f "$DB_PATH" ]; then
    touch "$DB_PATH"
    echo "Created DB at $DB_PATH"
else
    echo "DB already exists at $DB_PATH"
fi

echo "Note: Go will run InitTables() to initialize schema on server startup."