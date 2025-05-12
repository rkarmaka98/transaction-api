# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o tx-api .

# Final stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/tx-api /usr/local/bin/tx-api
EXPOSE 8080
ENTRYPOINT ["tx-api"]