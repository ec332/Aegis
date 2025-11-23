module github.com/ec332/aegis/market

go 1.22

replace (
    github.com/aegis/proto => ../proto
    github.com/aegis/shared => ../shared
)

require (
    github.com/aegis/proto v0.0.0
    github.com/aegis/shared v0.0.0
    github.com/go-chi/chi/v5 v5.2.3
    github.com/go-chi/cors v1.2.1
    github.com/google/uuid v1.6.0
    github.com/joho/godotenv v1.5.1
    github.com/lib/pq v1.10.9
    github.com/redis/go-redis/v9 v9.16.0
    go.uber.org/zap v1.24.0
    google.golang.org/grpc v1.65.0
    google.golang.org/protobuf v1.33.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/segmentio/kafka-go v0.4.38 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
)
