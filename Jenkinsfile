pipeline {
  agent any

  environment {
    DEPLOY_BASE = "/var/jenkins_home/workspace/${JOB_NAME}/deploy"
    DEPLOY_DIR = "${DEPLOY_BASE}/${env.BRANCH_NAME}"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Build') {
      steps {
        echo "🔨 Building Golang project on branch ${env.BRANCH_NAME}"
        sh '''
          go mod tidy
          go build -o app
        '''
      }
    }

    stage('Deploy') {
      steps {
        echo "🚀 Deploying to ${DEPLOY_DIR}"
        sh '''
          mkdir -p ${DEPLOY_DIR}
          cp app ${DEPLOY_DIR}/
        '''
      }
    }

    stage('Run App') {
      steps {
        echo "🏃 Running app in background"
        sh '''
          pkill -f "${DEPLOY_DIR}/app" || true
          nohup ${DEPLOY_DIR}/app > ${DEPLOY_DIR}/output.log 2>&1 &
        '''
      }
    }
  }

  post {
    success {
      echo "✅ Deployed and running branch ${env.BRANCH_NAME} successfully at ${DEPLOY_DIR}"
    }
    failure {
      echo "❌ Build or deploy failed for branch ${env.BRANCH_NAME}"
    }
  }
}
