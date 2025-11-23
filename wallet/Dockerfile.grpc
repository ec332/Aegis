FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the gRPC Wallet service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wallet-grpc ./wallet/cmd/grpc_main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/wallet-grpc .

# Expose ports
EXPOSE 8082 50052

# Run the binary
CMD ["./wallet-grpc"]