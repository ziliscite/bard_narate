# Base Go Image
FROM golang:1.24.0-alpine AS builder

# Set working directory
WORKDIR /app

# Add source code
COPY . /app

# Build the binary and add environment variable through CGO_ENABLED
RUN CGO_ENABLED=0 go build -o subscription ./cmd/api

RUN chmod +x /app/subscription

# Build a small image
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy the pre-built binary file from the previous stage
COPY --from=builder app/subscription ./

# Copy migrations files
COPY migrations ./migrations

# Expose gRPC port
EXPOSE 50051

# Command to run the executable
CMD ["./subscription"]
