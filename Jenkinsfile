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
    stage('Login to Azure') {
      steps {
        withCredentials([usernamePassword(
          credentialsId: 'acr-credentials',                         // Service Principal credentials stored in Jenkins :contentReference[oaicite:4]{index=4}
          usernameVariable: 'AZ_APP_ID',
          passwordVariable: 'AZ_PASSWORD'
        ), string(
          credentialsId: 'azure-tenant',                     // Tenant ID stored as a secret :contentReference[oaicite:5]{index=5}
          variable: 'AZ_TENANT'
        )]) {
          sh '''
            az login --service-principal \
              --username $AZ_APP_ID \
              --password $AZ_PASSWORD \
              --tenant $AZ_TENANT                             # Authenticate to Azure :contentReference[oaicite:6]{index=6}
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
            .                                              # Build & push via ACR Tasks :contentReference[oaicite:7]{index=7}
        '''
      }
    }
  }
}

