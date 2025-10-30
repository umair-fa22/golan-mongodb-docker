# syntax=docker/dockerfile:1.6

# Stage 1: Build the Go binary
FROM golang:1.24.9-alpine AS builder
WORKDIR /app

# Pre-cache modules
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source code
COPY . .

# Build a static binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/main .

# Stage 2: Minimal runtime image
FROM alpine:3.20
WORKDIR /app

# If the app does HTTPS calls, you'll need CA certs
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -H -u 10001 app && chown -R app:app /app
USER app

# Set the MongoDB connection URI
ENV MONGODB_URI=mongodb://localhost:27017

# Copy the built binary and seed.json
COPY --from=builder /app/main /app/main
# COPY --from=builder /app/seed.json /app/seed.json

# Expose the port the app runs on (informational)
EXPOSE 3000

# Command to run the application
# CMD [ "pwd" ]
# RUN pwd
ENTRYPOINT ["/app/main"]