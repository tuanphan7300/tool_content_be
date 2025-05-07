pipeline {
  agent any

  environment {
    IMAGE_NAME = "tool-content-be"
    DEPLOY_DIR = "/var/jenkins_home/deploy/${env.BRANCH_NAME}"
    APP_NAME = "tool-content-be-${BRANCH_NAME}"
    APP_PORT = "8080"
    MYSQL_ROOT_PASSWORD = "root"
    MYSQL_DATABASE = "tool"
    MYSQL_PORT = "3306"
    SUBDOMAIN = "${BRANCH_NAME}"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Build binary') {
      steps {
        echo "Building Go binary (linux/amd64)..."
        sh '''
          GOOS=linux GOARCH=amd64 go mod tidy
          GOOS=linux GOARCH=amd64 go build -o app
        '''
      }
    }

    stage('Build Docker image') {
      steps {
        echo "Building Docker image ${IMAGE_NAME}:${BRANCH_NAME} (platform: linux/amd64)..."
        sh '''
          docker build --platform=linux/amd64 -t ${IMAGE_NAME}:${BRANCH_NAME} .
        '''
      }
    }

    stage('Run container') {
      steps {
        echo "Running Docker containers for branch ${BRANCH_NAME}"
        sh '''
          # Force remove existing containers and volumes
          docker-compose -p ${BRANCH_NAME} down -v --remove-orphans
          docker rm -f ${APP_NAME} mysql-db-${BRANCH_NAME} || true
          
          # Start containers with project name
          SUBDOMAIN=${SUBDOMAIN} docker-compose -p ${BRANCH_NAME} up -d
          
          # Wait for nginx container to be ready
          echo "Waiting for nginx container to be ready..."
          sleep 10
          
          # Debug: List all containers
          echo "Listing all containers:"
          docker ps -a
        '''
      }
    }

    stage('Update Nginx Config') {
      steps {
        echo "Updating Nginx configuration for ${SUBDOMAIN}"
        sh '''
          # Generate Nginx config from template
          envsubst < nginx/template.conf > /tmp/nginx-${BRANCH_NAME}.conf
          
          # Get the actual nginx container name
          NGINX_CONTAINER=$(docker ps -q -f name=${BRANCH_NAME}_nginx)
          if [ -z "$NGINX_CONTAINER" ]; then
            echo "Error: Nginx container not found!"
            exit 1
          fi
          echo "Found nginx container: $NGINX_CONTAINER"
          
          # Create conf.d directory if it doesn't exist
          docker exec $NGINX_CONTAINER mkdir -p /etc/nginx/conf.d
          
          # Copy config to Nginx container
          docker cp /tmp/nginx-${BRANCH_NAME}.conf $NGINX_CONTAINER:/etc/nginx/conf.d/
          
          # Reload Nginx
          docker exec $NGINX_CONTAINER nginx -s reload
        '''
      }
    }
  }

  post {
    success {
      echo "Deployed ${IMAGE_NAME}:${BRANCH_NAME} successfully to ${SUBDOMAIN}.localtest.me"
    }
    failure {
      echo "Failed to deploy ${IMAGE_NAME}:${BRANCH_NAME}"
    }
  }
}
