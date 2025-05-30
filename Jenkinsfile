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
    stage('Azure Login & ACR Build') {
      agent {
        docker {
          image 'mcr.microsoft.com/azure-cli:latest'
          args  '-u root'                   // run as root so we can login
        }
      }
      steps {
        withCredentials([usernamePassword(
          credentialsId: 'acr-credentials',                         // Service Principal credentials stored in Jenkins 
          usernameVariable: 'AZ_APP_ID',
          passwordVariable: 'AZ_PASSWORD'
        ), string(
          credentialsId: 'azure-tenant',                     // Tenant ID stored as a secret
          variable: 'AZ_TENANT'
        )]) {
          sh '''
            az login --service-principal \
              --username $AZ_APP_ID \
              --password $AZ_PASSWORD \
              --tenant $AZ_TENANT

            az acr build \
              --registry $ACR_NAME \
              --image $IMAGE_NAME:$BUILD_TAG \
              --image $IMAGE_NAME:latest \
              .
          '''
        }
      }
    }
    stage('ACR Build & Push') {
      steps {
        sh '''
          az acr build \
            --registry $ACR_NAME \
            --image $IMAGE_NAME:$BUILD_TAG \
            --image $IMAGE_NAME:latest \
            .                                              # Build & push via ACR Tasks
        '''
      }
    }
  }
}

