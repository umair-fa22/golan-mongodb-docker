# ---------- Stage 1: Build ----------
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Install git for dependency fetching
RUN apk add --no-cache git

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy all project files
COPY . .

# Build Go binary (adjust main.go path if different)
RUN go build -o main .

# ---------- Stage 2: Runtime ----------
FROM alpine:3.20

# Create working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose backend port (change if your app uses another)
EXPOSE 8080

# Add Health Check for your app
# Make sure your app has an endpoint like `/health` or `/ping`
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./main"]
