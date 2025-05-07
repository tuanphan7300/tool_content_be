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
        '''
      }
    }

    stage('Update Nginx Config') {
      steps {
        echo "Updating Nginx configuration for ${SUBDOMAIN}"
        sh '''
          # Generate Nginx config from template
          envsubst < nginx/template.conf > /tmp/nginx-${BRANCH_NAME}.conf
          
          # Copy config to Nginx container
          docker cp /tmp/nginx-${BRANCH_NAME}.conf ${BRANCH_NAME}_nginx_1:/etc/nginx/conf.d/
          
          # Reload Nginx
          docker exec ${BRANCH_NAME}_nginx_1 nginx -s reload
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
