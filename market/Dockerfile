FROM golang:1.22-alpine AS builder

WORKDIR /app/market

# Install dependencies
RUN apk add --no-cache git

# Copy entire repo (for local replaces)
COPY . /app

# Ensure dependencies and go.sum entries are complete
RUN go mod tidy

# Build the gRPC Market service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o market-grpc ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/market/market-grpc .

# Expose gRPC port only
EXPOSE 50051

# Run the binary
CMD ["./market-grpc"]