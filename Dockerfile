FROM alpine:latest

# Set the working directory to /app
WORKDIR /app

# Install go
RUN apk add go

# Copy the Go module files
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o main main.go

# Expose the port
EXPOSE 50051 50053

# Run the command to start the application
CMD ["go", "run", "main.go", "server"]
