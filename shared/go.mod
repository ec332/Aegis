module github.com/aegis/shared

go 1.21

replace github.com/aegis/proto => ../proto

require (
	github.com/aegis/proto v0.0.0
	github.com/segmentio/kafka-go v0.4.38
	go.uber.org/zap v1.24.0
	google.golang.org/grpc v1.56.3
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
