# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app/transaction-service
COPY transaction-service/go.mod transaction-service/go.sum ./
RUN go mod tidy

COPY . /app
RUN go build -o transaction-service ./cmd/transaction-service/main.go

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/transaction-service .

EXPOSE 5555

CMD ["./transaction-service"]