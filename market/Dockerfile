FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the gRPC Market service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o market-grpc ./market/cmd/grpc_main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/market-grpc .

# Expose ports
EXPOSE 8081 50051

# Run the binary
CMD ["./market-grpc"]