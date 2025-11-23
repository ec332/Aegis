FROM golang:1.22-alpine AS builder
RUN apk add --no-cache build-base
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOFLAGS="-trimpath" go build -o /out/transaction-service ./cmd/transaction-service

FROM alpine:3.19
RUN adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/transaction-service /usr/local/bin/transaction-service
ENV APP_HTTP_PORT=5555
USER appuser
EXPOSE 5555
ENTRYPOINT ["/usr/local/bin/transaction-service"]