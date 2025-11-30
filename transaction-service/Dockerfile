# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY transaction-service/go.mod transaction-service/go.sum ./transaction-service/
COPY proto/ ./proto/
RUN cd transaction-service && go mod tidy

COPY . /app
RUN cd transaction-service && go build -o transaction-service ./cmd/transaction-service/main.go

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/transaction-service/transaction-service .

EXPOSE 50052

CMD ["./transaction-service"]