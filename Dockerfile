# Stage 1: Build the Go binary
FROM golang:1.23 as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o app

# Stage 2: Minimal runtime image
FROM debian:bullseye-slim

WORKDIR /app

# Copy from build stage
COPY --from=builder /app/app .

# Expose port your app listens on
EXPOSE 8080

# Start the app
CMD ["./app"]
