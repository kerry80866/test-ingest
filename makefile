PROTOC        := protoc
PROTOC_GEN_GO := $(shell which protoc-gen-go)
PROTOC_GEN_GRPC_GO := $(shell which protoc-gen-go-grpc)

build:
	go build -o bin/initdb ./cmd/dbtable/main.go
	go build -o bin/initdb ./cmd/dbtable/main_check.go

run-init:
	go run ./cmd/dbtable/main.go

run-check:
	go run ./cmd/dbtable/check.go

proto-event:
	@mkdir -p pb
	@echo "Generating event.proto → pb/"
	$(PROTOC) \
		--proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		proto/event.proto

ingest-query:
	@mkdir -p pb
	@echo "Generating ingest_query.proto → pb/"
	$(PROTOC) \
      --proto_path=proto \
      --go_out=pb --go_opt=paths=source_relative \
      --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
      proto/ingest_query.proto

clean:
	@echo "Cleaning generated .pb.go files..."
	@find pb -name "*.pb.go" -delete
