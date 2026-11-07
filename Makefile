PROTO_DIR := api/proto
PB_DIR := internal/common/pb

.PHONY: all client server proto lab-proto lab-py-proto clean test

PYTHON ?= python3

all: client server

client:
	@echo "Building gotermoku client..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o gotermoku ./cmd/gotermoku

server:
	@echo "Building gotermoku-server..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o gotermoku-server ./cmd/gotermoku-server

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

# Sinh mã cho hands-on lab gRPC (doc/handonlab). Cần protoc-gen-go và
# protoc-gen-go-grpc trong $(go env GOPATH)/bin.
lab-proto:
	PATH="$$PATH:$$(go env GOPATH)/bin" protoc \
		--go_out=. \
		--go_opt=module=github.com/ttasc/gotermoku \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/ttasc/gotermoku \
		doc/handonlab/grpctypes/api/profile.proto

# Sinh stub Python từ cùng profile.proto. Cần `pip install -r
# doc/handonlab/grpc-python/requirements.txt` (có grpcio-tools).
lab-py-proto:
	$(PYTHON) -m grpc_tools.protoc \
		-I doc/handonlab/grpctypes/api \
		--python_out=doc/handonlab/grpc-python \
		--grpc_python_out=doc/handonlab/grpc-python \
		doc/handonlab/grpctypes/api/profile.proto
