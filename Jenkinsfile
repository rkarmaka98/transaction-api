pipeline {
  agent any
  environment {
    ACR        = 'transactionappdemo.azurecr.io'
    IMAGE_NAME = 'tx-api'
    BUILD_TAG  = "${env.BUILD_NUMBER}"
  }
  stages {
    stage('Checkout') {
      steps { checkout scm }
    }
    stage('Build & Test') {
      steps {
        sh 'go test ./...'
      }
    }
    stage('Build & Push Docker Image') {
      agent {
        docker {
          image 'docker:24.0.5-cli'          // Use official Docker CLI image
          args  '-v /var/run/docker.sock:/var/run/docker.sock'  
                                            // Mount Docker socket for DinD 
        }
      }
      steps {
        script {
          docker.withRegistry("https://${ACR}", 'acr-credentials') {  
            // Log in to Azure Container Registry

            // Build with cache and version arg
            def customImage = docker.build(
              "${ACR}/${IMAGE_NAME}:${BUILD_TAG}",
              "--build-arg VERSION=${BUILD_TAG} --cache-from ${ACR}/${IMAGE_NAME}:latest ."
            )                                   // Tag and build from context 

            // Push both versioned and latest tags
            customImage.push()                  // Push the BUILD_TAG 
            customImage.push('latest')          // Also update the `latest` tag 
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
}
