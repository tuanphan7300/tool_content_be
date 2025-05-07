pipeline {
  agent any

  environment {
    IMAGE_NAME = "tool-content-be"
    DEPLOY_DIR = "/var/jenkins_home/deploy/${env.BRANCH_NAME}"
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
        echo "Running Docker container for branch ${BRANCH_NAME}"
        sh '''
          docker rm -f ${IMAGE_NAME}-${BRANCH_NAME} || true
          docker run -d --name ${IMAGE_NAME}-${BRANCH_NAME} -p 3000:8080 ${IMAGE_NAME}:${BRANCH_NAME}
        '''
      }
    }
  }

  post {
    success {
      echo "Deployed ${IMAGE_NAME}:${BRANCH_NAME} successfully"
    }
  }
}
