FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy entire repo and tidy at root module
COPY . /app
RUN go mod tidy

# Build the Settlement gRPC service
WORKDIR /app/settlement
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o settlement ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/settlement/settlement .

# Expose ports
EXPOSE 8084 50053

# Run the binary
CMD ["./settlement"]