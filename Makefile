# Makefile
.PHONY: proto clean

proto:
	@echo "Generating gRPC code..."
	@mkdir -p internal/netio/pb
	protoc --go_out=internal/netio/pb --go-grpc_out=internal/netio/pb api/proto/gomoku.proto

clean:
	@rm -rf internal/netio/pb
