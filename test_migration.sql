-- Test script to verify the migration worked correctly
-- Run this after the migration to check if the new column exists and has data

-- Check if the column exists
DESCRIBE caption_histories;

-- Check if there are any records with NULL video_file_name_origin
SELECT COUNT(*) as null_origin_count 
FROM caption_histories 
WHERE video_file_name_origin IS NULL;

-- Show a few sample records to verify the data
SELECT 
    id,
    video_filename,
    video_file_name_origin,
    created_at
FROM caption_histories 
ORDER BY created_at DESC 
LIMIT 5;

-- Check if all records have video_file_name_origin populated
SELECT 
    COUNT(*) as total_records,
    COUNT(video_file_name_origin) as records_with_origin,
    COUNT(*) - COUNT(video_file_name_origin) as records_without_origin
FROM caption_histories; 