pipeline {
  agent any
  environment {
    DEPLOY_DIR = "/var/www/${env.BRANCH_NAME}.localtest.me"
  }
  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Build') {
      steps {
        echo "Building Golang project on branch ${env.BRANCH_NAME}"
        sh '''
          go mod tidy
          go build -o app
        '''
      }
    }

    stage('Deploy') {
      steps {
        echo "Deploying to ${DEPLOY_DIR}"
        sh '''
          mkdir -p ${DEPLOY_DIR}
          cp app ${DEPLOY_DIR}/
        '''
      }
    }
  }

  post {
    success {
      echo "Deployed branch ${env.BRANCH_NAME} successfully"
    }
  }
}
