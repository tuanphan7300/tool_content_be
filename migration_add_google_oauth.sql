-- Migration to add Google OAuth fields to users table
-- Run this script to update your database schema

-- Add Google OAuth columns to users table
ALTER TABLE users 
ADD COLUMN google_id VARCHAR(255) NULL,
ADD COLUMN name VARCHAR(255) NULL,
ADD COLUMN picture TEXT NULL,
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN auth_provider VARCHAR(50) DEFAULT 'local';

-- Add index for google_id for faster lookups
CREATE INDEX idx_users_google_id ON users(google_id);

-- Add index for auth_provider
CREATE INDEX idx_users_auth_provider ON users(auth_provider);

-- Update existing users to have 'local' as auth_provider
UPDATE users SET auth_provider = 'local' WHERE auth_provider IS NULL; 