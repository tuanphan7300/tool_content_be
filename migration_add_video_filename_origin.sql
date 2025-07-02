-- Migration script to add video_file_name_origin column to caption_histories table
-- Run this script to add the new column for storing original video filenames

ALTER TABLE caption_histories 
ADD COLUMN video_filename_origin VARCHAR(255) DEFAULT NULL
AFTER video_filename;

-- Update existing records to set video_file_name_origin to the same value as video_filename
-- This ensures backward compatibility for existing data
UPDATE caption_histories 
SET video_filename_origin = video_filename
WHERE video_filename_origin IS NULL;