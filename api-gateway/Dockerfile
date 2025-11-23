FROM golang:1.22-alpine AS builder

WORKDIR /app/api-gateway

# Install dependencies
RUN apk add --no-cache git

# Copy entire repo (needed for local replace to ../)
COPY . /app

# Ensure dependencies and go.sum entries are complete
RUN go mod tidy

# Build the API Gateway
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/api-gateway .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./api-gateway"]