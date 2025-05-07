pipeline {
  agent any
  environment {
    IMAGE_NAME = "tool-content-be"
    DEPLOY_DIR = "/var/jenkins_home/deploy/${env.BRANCH_NAME}"
  }
  stages {
    stage('Debug Workspace') {
      steps {
        sh 'pwd'
        sh 'ls -la'
        sh 'git rev-parse --resolve-git-dir .git || echo "Not a git directory"'
      }
    }
    stage('Checkout') {
      steps {
        checkout scm
      }
    }
    stage('Build binary') {
      steps {
        echo "Building Go binary..."
        sh '''
          go mod tidy
          go build -o app
        '''
      }
    }
    stage('Build Docker image') {
      steps {
        echo "Building Docker image ${IMAGE_NAME}:${BRANCH_NAME}"
        sh '''
          docker build -t ${IMAGE_NAME}:${BRANCH_NAME} .
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