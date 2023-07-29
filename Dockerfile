# Start from the latest golang base image
FROM golang:latest

# Add Maintainer Info
LABEL maintainer="Your Name <youremail@site.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY ./src/go.mod ./src/go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY ./src .

# Build the Go app
RUN go build -o main .

# Expose port 8080 to the outside
EXPOSE 8080

# Run the executable
CMD ["./main"]