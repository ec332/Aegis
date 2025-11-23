module aegis

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/golang/protobuf v1.5.3
	github.com/google/uuid v1.5.0
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/segmentio/kafka-go v0.4.46
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.26.0
	golang.org/x/net v0.19.0
	golang.org/x/sys v0.15.0
	golang.org/x/text v0.14.0
    google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f
    google.golang.org/grpc v1.65.0
    google.golang.org/protobuf v1.33.0
)

replace github.com/aegis/proto => ./proto
replace github.com/aegis/shared => ./shared