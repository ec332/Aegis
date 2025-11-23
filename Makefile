protos:
	docker run --rm -v $(PWD):/work -w /work golang:1.22-alpine sh -c 'apk add --no-cache git protobuf && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && PATH=/go/bin:$$PATH protoc -I proto --go_out=paths=source_relative:proto/gen --go-grpc_out=paths=source_relative:proto/gen proto/market.proto proto/settlement.proto proto/wallet.proto'

protos-local:
	protoc -I proto --go_out=paths=source_relative:proto/gen --go-grpc_out=paths=source_relative:proto/gen proto/market.proto proto/settlement.proto proto/wallet.proto