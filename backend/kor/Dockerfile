# Use an official Golang image as the build environment
FROM golang:1.22.2-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files to install dependencies
COPY go.mod go.sum ./

# Download the Go dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
RUN go build -o /go-service

# Use a smaller base image for the final container
FROM alpine:latest

# Set the working directory for the final container
WORKDIR /root/

# Copy the compiled Go binary from the builder stage
COPY --from=builder /go-service .
COPY server.crt .
COPY server.key .
# Expose the application's port (optional)
EXPOSE 8080

# Command to run the Go service
CMD ["./go-service"]