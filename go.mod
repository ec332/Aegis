module aegis

go 1.22

require (
	github.com/aegis/proto v0.0.0-00010101000000-000000000000
	github.com/aegis/shared v0.0.0-00010101000000-000000000000
	github.com/ethereum/go-ethereum v1.14.13
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/stretchr/testify v1.9.0
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240223125850-b1e8a79f509c // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/ethereum/go-verkle v0.1.1-0.20240829091221-dffa7562dbe9 // indirect
	github.com/holiman/uint256 v1.3.1 // indirect
	github.com/klauspost/compress v1.16.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/segmentio/kafka-go v0.4.46 // indirect
	github.com/supranational/blst v0.3.13 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace github.com/aegis/proto => ./proto

replace github.com/aegis/shared => ./shared
