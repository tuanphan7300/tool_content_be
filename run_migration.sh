#!/bin/bash

# Script to run database migration for adding video_file_name_origin column
# Make sure to update the database connection details according to your setup

echo "Running database migration to add video_file_name_origin column..."

# Update these values according to your database configuration
DB_HOST="localhost"
DB_PORT="3306"
DB_USER="root"
DB_PASSWORD="root"
DB_NAME="tool"

# Run the migration
mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_add_video_filename_origin.sql
mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_add_google_oauth.sql

if [ $? -eq 0 ]; then
    echo "Migration completed successfully!"
    echo "The video_file_name_origin column has been added to the caption_histories table."
else
    echo "Migration failed! Please check your database connection and try again."
    exit 1
fi 