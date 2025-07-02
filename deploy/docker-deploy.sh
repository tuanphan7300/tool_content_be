#!/bin/bash

# Production Docker Deployment Script
# Cho developer có kinh nghiệm

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="tool-creator"
DOMAIN=${DOMAIN:-"your-domain.com"}
ENVIRONMENT=${ENVIRONMENT:-"production"}

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed"
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose is not installed"
    fi
    
    # Check if .env exists
    if [ ! -f .env ]; then
        error ".env file not found. Please create it with your configuration."
    fi
    
    log "Prerequisites check passed"
}

# Create .env file if not exists
create_env_file() {
    if [ ! -f .env ]; then
        log "Creating .env file..."
        
        cat > .env <<EOF
# Database Configuration
DB_PASSWORD=your-secure-password
DB_ROOT_PASSWORD=your-root-password
DB_NAME=tool_creator

# Redis Configuration
REDIS_PASSWORD=your-redis-password

# API Keys
OPENAI_API_KEY=your-openai-api-key
GEMINI_API_KEY=your-gemini-api-key

# JWT Configuration
JWT_SECRET=your-jwt-secret-key

# Application Configuration
ENVIRONMENT=production
PORT=8888

# Monitoring
GRAFANA_PASSWORD=admin123

# Domain
DOMAIN=$DOMAIN
EOF
        
        warn "Please edit .env file with your actual values before continuing"
        exit 1
    fi
}

# Setup SSL certificates
setup_ssl() {
    log "Setting up SSL certificates..."
    
    # Create SSL directory
    mkdir -p ssl
    
    # Check if certificates already exist
    if [ -f "ssl/fullchain.pem" ] && [ -f "ssl/privkey.pem" ]; then
        log "SSL certificates already exist"
        return
    fi
    
    # Get SSL certificate using certbot
    docker run --rm \
        -v $(pwd)/ssl:/etc/letsencrypt \
        -v $(pwd)/ssl:/var/lib/letsencrypt \
        certbot/certbot certonly \
        --standalone \
        -d $DOMAIN \
        --email admin@$DOMAIN \
        --agree-tos \
        --no-eff-email \
        --force-renewal
    
    # Copy certificates to nginx ssl directory
    cp ssl/live/$DOMAIN/fullchain.pem ssl/
    cp ssl/live/$DOMAIN/privkey.pem ssl/
    
    log "SSL certificates configured"
}

# Build and deploy
deploy() {
    log "Building and deploying application..."
    
    # Pull latest changes
    if [ -d ".git" ]; then
        git pull origin main
    fi
    
    # Build images
    docker-compose build --no-cache
    
    # Stop existing containers
    docker-compose down
    
    # Start services
    docker-compose up -d
    
    # Wait for services to be healthy
    log "Waiting for services to be healthy..."
    sleep 30
    
    # Check service status
    docker-compose ps
    
    log "Deployment completed"
}

# Scale services
scale_services() {
    log "Scaling services..."
    
    # Scale workers based on load
    read -p "Number of worker instances (default: 2): " WORKER_COUNT
    WORKER_COUNT=${WORKER_COUNT:-2}
    
    docker-compose up -d --scale worker=$WORKER_COUNT
    
    log "Scaled to $WORKER_COUNT worker instances"
}

# Setup monitoring
setup_monitoring() {
    log "Setting up monitoring..."
    
    # Create Grafana directories
    mkdir -p grafana/dashboards grafana/datasources
    
    # Create Prometheus datasource
    cat > grafana/datasources/prometheus.yml <<EOF
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
EOF
    
    # Create dashboard
    cat > grafana/dashboards/dashboard.yml <<EOF
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
EOF
    
    # Restart Grafana to load new configuration
    docker-compose restart grafana
    
    log "Monitoring setup completed"
}

# Backup data
backup_data() {
    log "Creating backup..."
    
    BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p $BACKUP_DIR
    
    # Backup database
    docker-compose exec -T mysql mysqldump -u tool_creator -p$DB_PASSWORD tool_creator > $BACKUP_DIR/database.sql
    
    # Backup application data
    docker run --rm -v $(pwd)/app_storage:/data -v $(pwd)/$BACKUP_DIR:/backup alpine tar czf /backup/app_data.tar.gz -C /data .
    
    # Backup Redis data
    docker-compose exec -T redis redis-cli BGSAVE
    sleep 5
    docker cp tool-creator-redis:/data/dump.rdb $BACKUP_DIR/redis_dump.rdb
    
    log "Backup created in $BACKUP_DIR"
}

# Restore data
restore_data() {
    log "Restoring data..."
    
    read -p "Backup directory to restore from: " BACKUP_DIR
    
    if [ ! -d "$BACKUP_DIR" ]; then
        error "Backup directory not found"
    fi
    
    # Restore database
    if [ -f "$BACKUP_DIR/database.sql" ]; then
        docker-compose exec -T mysql mysql -u tool_creator -p$DB_PASSWORD tool_creator < $BACKUP_DIR/database.sql
        log "Database restored"
    fi
    
    # Restore application data
    if [ -f "$BACKUP_DIR/app_data.tar.gz" ]; then
        docker run --rm -v $(pwd)/app_storage:/data -v $(pwd)/$BACKUP_DIR:/backup alpine tar xzf /backup/app_data.tar.gz -C /data
        log "Application data restored"
    fi
    
    # Restore Redis data
    if [ -f "$BACKUP_DIR/redis_dump.rdb" ]; then
        docker cp $BACKUP_DIR/redis_dump.rdb tool-creator-redis:/data/dump.rdb
        docker-compose restart redis
        log "Redis data restored"
    fi
}

# Health check
health_check() {
    log "Running health checks..."
    
    # Check if containers are running
    if ! docker-compose ps | grep -q "Up"; then
        error "Some containers are not running"
    fi
    
    # Check API health
    if curl -f http://localhost:8888/ping > /dev/null 2>&1; then
        log "API health check passed"
    else
        error "API health check failed"
    fi
    
    # Check queue status
    if curl -f http://localhost:8888/queue/status > /dev/null 2>&1; then
        log "Queue status check passed"
    else
        error "Queue status check failed"
    fi
    
    # Check monitoring
    if curl -f http://localhost:9090/-/healthy > /dev/null 2>&1; then
        log "Prometheus health check passed"
    else
        error "Prometheus health check failed"
    fi
    
    log "All health checks passed"
}

# Show logs
show_logs() {
    log "Showing recent logs..."
    
    echo -e "\n=== API Logs ==="
    docker-compose logs --tail=20 api
    
    echo -e "\n=== Worker Logs ==="
    docker-compose logs --tail=20 worker
    
    echo -e "\n=== Nginx Logs ==="
    docker-compose logs --tail=20 nginx
}

# Clean up
cleanup() {
    log "Cleaning up unused Docker resources..."
    
    # Remove unused containers
    docker container prune -f
    
    # Remove unused images
    docker image prune -f
    
    # Remove unused volumes
    docker volume prune -f
    
    # Remove unused networks
    docker network prune -f
    
    log "Cleanup completed"
}

# Display information
display_info() {
    echo -e "${GREEN}"
    echo "=========================================="
    echo "           DEPLOYMENT COMPLETED           "
    echo "=========================================="
    echo -e "${NC}"
    echo "🌐 Domain: https://$DOMAIN"
    echo "🔧 API Endpoint: https://$DOMAIN/api"
    echo "📊 Prometheus: http://localhost:9090"
    echo "📈 Grafana: http://localhost:3000 (admin/admin123)"
    echo ""
    echo "🛠️ Useful Commands:"
    echo "  View logs: docker-compose logs -f [service]"
    echo "  Scale workers: docker-compose up -d --scale worker=3"
    echo "  Restart service: docker-compose restart [service]"
    echo "  Backup: ./docker-deploy.sh backup"
    echo "  Health check: ./docker-deploy.sh health"
    echo ""
    echo "📁 Volumes:"
    echo "  Database: mysql_data"
    echo "  Redis: redis_data"
    echo "  Storage: app_storage"
    echo "  Models: app_models"
    echo ""
}

# Main function
main() {
    echo -e "${BLUE}"
    echo "=========================================="
    echo "    Production Docker Deployment Script   "
    echo "=========================================="
    echo -e "${NC}"
    
    case "$1" in
        "deploy")
            check_prerequisites
            create_env_file
            setup_ssl
            deploy
            setup_monitoring
            health_check
            display_info
            ;;
        "scale")
            scale_services
            ;;
        "backup")
            backup_data
            ;;
        "restore")
            restore_data
            ;;
        "health")
            health_check
            ;;
        "logs")
            show_logs
            ;;
        "cleanup")
            cleanup
            ;;
        "ssl")
            setup_ssl
            ;;
        "monitoring")
            setup_monitoring
            ;;
        *)
            echo "Usage: $0 {deploy|scale|backup|restore|health|logs|cleanup|ssl|monitoring}"
            echo ""
            echo "Commands:"
            echo "  deploy      - Deploy the application"
            echo "  scale       - Scale worker instances"
            echo "  backup      - Create backup"
            echo "  restore     - Restore from backup"
            echo "  health      - Run health checks"
            echo "  logs        - Show recent logs"
            echo "  cleanup     - Clean up unused resources"
            echo "  ssl         - Setup SSL certificates"
            echo "  monitoring  - Setup monitoring"
            exit 1
            ;;
    esac
}

# Run main function
main "$@" 