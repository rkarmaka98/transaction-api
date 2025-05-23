# My Transaction Application Workflow

## Table of Contents

1. [Introduction](#introduction)
2. [Requirements](#requirements)
3. [Project Layout](#project-layout)
4. [Backend Development](#backend-development)
5. [Testing](#testing)
6. [Docker Packaging](#docker-packaging)
7. [Publishing to ACR](#publishing-to-acr)
8. [Deploying to AKS](#deploying-to-aks)
9. [Common Challenges and Solutions](#common-challenges-and-solutions)
10. [Architectural Choices](#architectural-choices)
11. [Further Steps](#further-steps)

---

## Introduction

I’m excited to share how I built a simple transaction app from the ground up. This project covers everything, including coding the backend in Go, designing a user-friendly frontend, packaging components with Docker, storing images in Azure Container Registry (ACR), and deploying to Azure Kubernetes Service (AKS). Along the way, I’ll walk you through troubleshooting steps, useful tips, and the rationale behind my design decisions.

By the end of this guide, you’ll not only understand the code but also how to manage containers and Kubernetes resources in a real cloud environment.

---

## Requirements

Before we begin, make sure you have the following tools and access:

* **Go 1.20**: I used Go modules (`go.mod` and `go.sum`) to manage dependencies and ensure reproducible builds.
* **Docker**: Required to build and test container images locally. You’ll need at least version 20.x.
* **Azure CLI (`az`)**: I authenticated with `az login` and used commands to manage ACR and AKS. If you’re using a managed identity for AKS, adjust accordingly.
* **Azure Subscription**: You need permissions to create or attach an existing Azure Container Registry and an AKS cluster.
* **AKS Cluster**: A running AKS cluster with `kubectl` configured on your local machine or CI environment.
* (Optional) **Jenkins or CI tool**: If you plan to automate these steps in a pipeline later.

Having these prerequisites ready will make the workflow smoother and allow you to focus on the app logic.

---

## Project Layout

I organized my code and configuration files like this:

```
transaction-app/
├── go.mod               # Defines module path and minimum Go version
├── go.sum               # Records checksums of module dependencies
├── main.go              # Implementation of the Go HTTP server
├── main_test.go         # Unit tests for core server functionality
├── index.html           # Static HTML and JavaScript for the frontend
├── Dockerfile           # Multi-stage Docker build for backend
├── ui.Dockerfile        # Docker build for frontend
└── k8s/                 # Kubernetes manifests for deployment
    ├── namespace.yaml         # Namespace definition
    ├── api-deploy.yaml        # Deployment for the backend API
    ├── api-service.yaml       # Service (ClusterIP) for the API
    ├── ui-deploy.yaml         # Deployment for the UI, including Nginx proxy setup
    └── ui-service.yaml        # Service (LoadBalancer) for the UI
```

This layout helps me quickly locate code, tests, Docker configurations, and Kubernetes manifests. Separating concerns reduces confusion and makes collaboration easier if you work with a team.
![Architecture](https://github.com/user-attachments/assets/5c4ce407-6d8f-43cb-823e-3e53a4976681)
![Workflow](https://github.com/user-attachments/assets/4ba7f95d-af3e-4a75-856c-fcb4bc4e809b)

---

## Backend Development

In **`main.go`**, I built a lightweight HTTP server using only Go’s standard library:

1. **In-Memory Store**: I used a `map[string]float64` to track balances for each account. This approach is fast and avoids external dependencies for a demo.
2. **Mutex Locking**: I wrapped access to the map with a `sync.Mutex`, preventing simultaneous requests from corrupting data.
3. **HTTP Handlers**:

   * **`GET /balance/{account}`**: Returns a JSON object with the current balance. If the account doesn’t exist, it responds with 404.
   * **`POST /transfer`**: Accepts JSON input `{from, to, amount}`. It validates that the amount is positive and that the source account has sufficient funds, returning 400 or 422 as needed.
4. **Error Handling**: I made sure to include helpful error messages and correct HTTP status codes so clients can respond appropriately.

Below is an expanded view of my transfer handler, showing comments and logging:

```go
func transferHandler(w http.ResponseWriter, r *http.Request) {
    var req transferRequest
    // Decode incoming JSON into our request struct
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Failed to decode JSON: %v", err)
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        return
    }
    mu.Lock()
    defer mu.Unlock()

    // Validate transfer amount
    if req.Amount <= 0 {
        log.Printf("Invalid amount: %v", req.Amount)
        http.Error(w, "Amount must be positive", http.StatusBadRequest)
        return
    }
    // Check for sufficient funds
    if balances[req.From] < req.Amount {
        log.Printf("Insufficient funds in %s: have %v, want %v", req.From, balances[req.From], req.Amount)
        http.Error(w, "Insufficient funds", http.StatusUnprocessableEntity)
        return
    }
    // Perform the transfer
    balances[req.From] -= req.Amount
    balances[req.To] += req.Amount
    log.Printf("Transferred %v from %s to %s", req.Amount, req.From, req.To)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}
```

Adding logging entries helped me trace requests in development and diagnose issues quickly.

---

## Testing

I wrote **unit tests** in `main_test.go` to verify core functionality:

* **Balance Checks**: Ensured `GET /balance/alice` returned the expected amount.
* **Transfer Logic**: Simulated valid and invalid transfers and checked HTTP responses and map updates.

Example test for transfers:

```go
func TestTransferHandler(t *testing.T) {
    // Reset balances for test
    balances = map[string]float64{"alice": 100, "bob": 0}
    body := `{"from":"alice","to":"bob","amount":30}`
    req := httptest.NewRequest("POST", "/transfer", strings.NewReader(body))
    w := httptest.NewRecorder()

    transferHandler(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("Expected 200 OK, got %d", w.Code)
    }
    if balances["alice"] != 70 {
        t.Errorf("Expected alice to have 70, got %v", balances["alice"])
    }
    if balances["bob"] != 30 {
        t.Errorf("Expected bob to have 30, got %v", balances["bob"])
    }
}
```

I ran tests frequently with:

```bash
go test ./... -v
```

This practice helped me detect errors early and gave me confidence before moving to Docker packaging.

---

## Docker Packaging

I used **multi-stage Docker builds** to produce minimal container images.

### Backend Dockerfile

```dockerfile
# Stage 1: Build the Go binary
FROM golang:1.20-alpine AS builder
WORKDIR /app
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o tx-api .

# Stage 2: Create a small runtime image
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/tx-api /usr/local/bin/tx-api
EXPOSE 8080
ENTRYPOINT ["tx-api"]
```

**Why multi-stage?**

* **Security**: The final image contains only the compiled binary and essential dependencies, reducing the attack surface.
* **Size**: By dropping compiler and source code, the image stays under 6MB, which speeds up pushes and pulls.

I built and tested the image locally:

```bash
docker build -t tx-api:latest .
docker run --rm -p 8080:8080 tx-api:latest
```

Seeing the server logs live in my terminal verified that the container ran correctly.

---

## Publishing to ACR

Once my local images were working, I pushed them to **Azure Container Registry**:

1. **Login to ACR**:

   ```bash
   az acr login --name myAcrName
   ```

   I made sure my Azure CLI session had the right permissions.

2. **Tag and push the backend**:

   ```bash
   docker tag tx-api:latest myAcrName.azurecr.io/tx-api:latest
   docker push myAcrName.azurecr.io/tx-api:latest
   ```

3. **Push the UI image** prepared with `ui.Dockerfile`:

   ```bash
   docker build -f ui.Dockerfile -t tx-ui:latest .
   docker tag tx-ui:latest myAcrName.azurecr.io/tx-ui:latest
   docker push myAcrName.azurecr.io/tx-ui:latest
   ```

If I encountered `permission denied` or `unauthorized` errors, I re-ran `az acr login` and checked that my CLI was targeting the correct subscription.

---

## Deploying to AKS

With images in ACR, I moved on to deploying in AKS.

### 1. Attach ACR to AKS Cluster

I avoided creating Kubernetes secrets by attaching ACR directly:

```bash
az aks update -g MyResourceGroup -n MyAksCluster --attach-acr myAcrName
```

This uses AKS’s managed identity to pull images securely.

### 2. Kubernetes Manifests

I stored the following in `k8s/`:

* **namespace.yaml**:

  ```yaml
  apiVersion: v1
  kind: Namespace
  metadata:
    name: transaction-app
  ```
* **api-deploy.yaml** and **api-service.yaml** for the backend:

  ```yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: tx-api
    namespace: transaction-app
  spec:
    replicas: 2
    selector:
      matchLabels:
        app: tx-api
    template:
      metadata:
        labels:
          app: tx-api
      spec:
        containers:
          - name: tx-api
            image: myAcrName.azurecr.io/tx-api:latest
            ports:
              - containerPort: 8080
  ---
  apiVersion: v1
  kind: Service
  metadata:
    name: tx-api
    namespace: transaction-app
  spec:
    type: ClusterIP
    selector:
      app: tx-api
    ports:
      - port: 80
        targetPort: 8080
  ```
* **ui-deploy.yaml** and **ui-service.yaml** for the frontend:

  ```yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: tx-ui
    namespace: transaction-app
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: tx-ui
    template:
      metadata:
        labels:
          app: tx-ui
      spec:
        containers:
          - name: tx-ui
            image: myAcrName.azurecr.io/tx-ui:latest
            ports:
              - containerPort: 80
  ---
  apiVersion: v1
  kind: Service
  metadata:
    name: tx-ui
    namespace: transaction-app
  spec:
    type: LoadBalancer
    selector:
      app: tx-ui
    ports:
      - port: 80
        targetPort: 80
  ```

### 3. Apply Manifests

I ran:

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/api-deploy.yaml
kubectl apply -f k8s/api-service.yaml
kubectl apply -f k8s/ui-deploy.yaml
kubectl apply -f k8s/ui-service.yaml
```

This created all resources in the `transaction-app` namespace.

### 4. Access the UI

After a minute, I checked the LoadBalancer IP:

```bash
kubectl get svc -n transaction-app tx-ui
```

I copied the `EXTERNAL-IP` and opened `http://<EXTERNAL-IP>` in my browser. I tested transactions, and everything worked as expected.

---

## Common Challenges and Solutions

Here are issues I faced and how I resolved them:

1. **Missing `go.sum` in Docker build**:

   * **Problem**: The build failed because `go.sum` was absent.
   * **Solution**: Ran `go mod tidy` locally to generate it, and rebuilt.

2. **Invalid Go version in `go.mod`**:

   * **Problem**: I accidentally specified `go 1.20.1`, which Go modules didn’t accept.
   * **Solution**: Updated the directive to `go 1.20` with `go mod edit -go=1.20`.

3. **Port binding conflicts**:

   * **Problem**: Port 8080 was already in use on my laptop.
   * **Solution**: Ran `docker run -p 9090:8080 tx-api` and updated local test URLs.

4. **Public IP quota reached**:

   * **Problem**: I couldn’t create more LoadBalancer IPs in my Azure region.
   * **Solution**: Kept the API service as `ClusterIP` and only exposed the UI.

5. **API unreachable from UI**:

   * **Problem**: The frontend couldn’t call the backend.
   * **Solution**: Exec’d into the UI pod (`kubectl exec -it ...`) and used `curl` to verify internal DNS and ports.

These troubleshooting steps saved me time and headaches.

---

## Architectural Choices

I made these design decisions for simplicity and clarity:

* **In-Memory Store**: Eliminates external database setup for a quick demo. Data is lost on restart, but that's acceptable for a proof of concept.
* **Mutex Locking**: Demonstrates Go’s concurrency primitives without adding complexity.
* **Multi-Stage Docker Builds**: Keeps final images minimal and separates build-time dependencies from runtime.
* **ACR Attach**: Uses AKS’s managed identity to simplify pulling private images.
* **ClusterIP + LoadBalancer**: Minimizes public endpoints, exposing only the UI and keeping the API internal.

Each choice balances ease of use, security, and maintainability.

---

## Further Steps

After completing this demo, I plan to expand it by:

* **Adding Persistence**: Integrate a database (e.g., Azure SQL, Cosmos DB) for durable storage of transactions.
* **Enhanced Observability**: Implement Prometheus metrics in Go and create Grafana dashboards for latency, error rates, and request throughput.
* **CI/CD Automation**: Use Jenkins, GitHub Actions, or Azure Pipelines to automatically run tests, build images, and deploy to AKS on every commit.
* **Auto-Scaling**: Configure Kubernetes Horizontal Pod Autoscaler to scale based on CPU or custom metrics, ensuring the app handles variable load.
* **Security Hardening**: Add TLS termination with Ingress, enable Pod Security Policies, and scan images for vulnerabilities.

By taking these steps, I’ll turn this simple demo into a production-ready microservice application. Now you have a thorough, expanded workflow to code, containerize, store, and deploy your transaction app in the cloud using Go, Docker, and Kubernetes.
