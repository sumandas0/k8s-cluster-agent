# Build stage
FROM golang:1.24.3-alpine3.22 as builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install swag for OpenAPI generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy all source code
COPY . .

# Generate OpenAPI docs
RUN swag init -g cmd/agent/main.go -o docs --parseInternal --parseDependency || true

# Build the binary using make
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build

# Move binary to expected location
RUN mv bin/k8s-cluster-agent .

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -u 1001 -g agent agent

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/k8s-cluster-agent .

# Change ownership
RUN chown -R agent:agent /app

# Switch to non-root user
USER 1001

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["./k8s-cluster-agent"] 