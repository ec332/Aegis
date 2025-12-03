module api-gateway

go 1.22

require (
	github.com/aegis/proto v0.0.0
	github.com/aegis/shared v0.0.0
	github.com/alicebob/miniredis/v2 v2.35.0
	github.com/go-chi/cors v1.2.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/gorilla/mux v1.8.1
	github.com/redis/go-redis/v9 v9.17.2
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.26.0
	golang.org/x/sync v0.7.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.1
)

replace aegis => ../

replace github.com/aegis/proto => ../proto

replace github.com/aegis/shared => ../shared

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/segmentio/kafka-go v0.4.46 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
