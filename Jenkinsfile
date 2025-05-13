pipeline {
  agent any
  environment {
    ACR        = 'transactionappdemo.azurecr.io'
    IMAGE_NAME = 'tx-api'
    BUILD_TAG  = "${env.BUILD_NUMBER}"
  }
  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }
    stage('Build & Test') {
      steps {
        sh 'go test ./...'
        sh 'go build -o tx-api .'
      }
    }
    stage('Docker Build & Push') {
      steps {
        script {
          docker.withRegistry("https://${ACR}", 'acr-credentials') {
            def img = docker.build("${ACR}/${IMAGE_NAME}:${BUILD_TAG}")
            img.push()
          }
        }
      }
    }
    stage('Deploy via Helm') {
      steps {
        withKubeConfig(credentialsId: 'aks-kubeconfig') {
          sh """
            helm upgrade --install ${IMAGE_NAME} helm-chart/ \
              --namespace default \
              --set image.repository=${ACR}/${IMAGE_NAME} \
              --set image.tag=${BUILD_TAG}
          """
        }
      }
    }
  }
  post {
    success { echo '✅ Deployment succeeded' }
    failure { echo '❌ Deployment failed' }
  }
}
