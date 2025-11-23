module api-gateway

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	aegis v0.0.0
)

replace aegis => ../

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/segmentio/kafka-go v0.4.46 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
)