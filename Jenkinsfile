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

    stage('Deploy') {
      steps {
        echo "Deploying containers for branch ${BRANCH_NAME}"
        sh '''
          # Clean up existing containers
          docker-compose -p ${BRANCH_NAME} down -v --remove-orphans
          
          # Start containers
          docker-compose -p ${BRANCH_NAME} up -d
          
          # Wait for nginx to be ready
          echo "Waiting for nginx to be ready..."
          for i in {1..30}; do
            if docker ps -q -f name=nginx-${BRANCH_NAME} | grep -q .; then
              if docker exec nginx-${BRANCH_NAME} nginx -t; then
                echo "Nginx is ready!"
                break
              fi
            fi
            if [ $i -eq 30 ]; then
              echo "Error: Nginx failed to start"
              exit 1
            fi
            sleep 2
          done
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
      sh '''
        docker-compose -p ${BRANCH_NAME} logs
      '''
    }
  }
}
