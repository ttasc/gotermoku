PROTO_DIR := api/proto
PB_DIR := internal/common/pb

.PHONY: all client server proto clean test

all: client server

client:
	@echo "Building gotermoku client..."
	go build -o gotermoku ./cmd/gotermoku

server:
	@echo "Building gotermoku-server..."
	go build -o gotermoku-server ./cmd/gotermoku-server

test:
	@echo "Running integration tests with race detector..."
	go test -v -race ./test/...

clean:
	@echo "Cleaning up binaries..."
	@rm -f gotermoku gotermoku-server

proto:
	mkdir -p $(PB_DIR)
	protoc \
		--go_out=. \
		--go_opt=module=github.com/ttasc/gotermoku \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/ttasc/gotermoku \
		$(PROTO_DIR)/gomoku.proto
