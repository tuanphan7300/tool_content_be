#!/bin/bash

# ========================================
# SECRET GENERATION SCRIPT
# ========================================
# This script generates secure secrets for production deployment

echo "🔐 Generating secure secrets for production..."

# Generate JWT Secret (64 characters)
JWT_SECRET=$(openssl rand -base64 48 | tr -d "=+/" | cut -c1-64)
echo "✅ JWT Secret generated:"
echo "JWTACCESSKEY=$JWT_SECRET"
echo ""

# Generate Database Password (32 characters)
DB_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-32)
echo "✅ Database Password generated:"
echo "DB_PASSWORD=$DB_PASSWORD"
echo ""

# Generate Redis Password (32 characters)
REDIS_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-32)
echo "✅ Redis Password generated:"
echo "REDIS_PASSWORD=$REDIS_PASSWORD"
echo ""

echo "📝 Copy these values to your .env file:"
echo "========================================"
echo "JWTACCESSKEY=$JWT_SECRET"
echo "DB_PASSWORD=$DB_PASSWORD"
echo "REDIS_PASSWORD=$REDIS_PASSWORD"
echo "========================================"
echo ""
echo "⚠️  IMPORTANT:"
echo "- Keep these secrets secure and never commit them to git"
echo "- Use different secrets for each environment"
echo "- Store backups of these secrets in a secure location" 