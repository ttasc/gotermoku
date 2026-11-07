# gRPC Python ↔ Go interoperability

Cùng một hợp đồng `../grpctypes/api/profile.proto`, nhưng server/client viết
bằng Python. Vì gRPC + Protocol Buffers độc lập ngôn ngữ, mọi tổ hợp sau đều
chạy được:

| Server | Client | Lệnh |
|---|---|---|
| Go | Go | `go run ./doc/handonlab/grpctypes/server` + `go run ./doc/handonlab/grpctypes/client` |
| Python | Python | `python3 doc/handonlab/grpc-python/server.py` + `python3 doc/handonlab/grpc-python/client.py` |
| **Python** | **Go** | Python server + `go run ./doc/handonlab/grpctypes/client` |
| **Go** | **Python** | Go server + `python3 doc/handonlab/grpc-python/client.py` |

Điểm mấu chốt: hai bên không cần biết ngôn ngữ của nhau, chỉ cần cùng `.proto`
và cùng địa chỉ/cổng.

## 1. Cài Python

Nên dùng virtualenv để không làm bẩn Python hệ thống:

```sh
python3 -m venv .venv
source .venv/bin/activate
pip install -r doc/handonlab/grpc-python/requirements.txt
```

Stub Python đã được sinh sẵn (`profile_pb2.py`, `profile_pb2_grpc.py`) nên
không bắt buộc cài `grpcio-tools` để chạy. Chỉ cài khi muốn sinh lại.

## 2. Chạy Python server

Terminal 1:

```sh
source .venv/bin/activate
python3 doc/handonlab/grpc-python/server.py
# hoặc đổi cổng
python3 doc/handonlab/grpc-python/server.py --addr 0.0.0.0:50052
```

## 3. Chạy client

Terminal 2 — Python client gọi Python server:

```sh
source .venv/bin/activate
python3 doc/handonlab/grpc-python/client.py
python3 doc/handonlab/grpc-python/client.py --name "Bình"
```

Terminal 2 — **Go client gọi Python server**:

```sh
go run ./doc/handonlab/grpctypes/client -name "Go gọi Python"
```

Terminal 2 — **Python client gọi Go server**:

```sh
# Terminal 1: chạy Go server
go run ./doc/handonlab/grpctypes/server

# Terminal 2: chạy Python client
source .venv/bin/activate
python3 doc/handonlab/grpc-python/client.py --name "Python gọi Go"
```

## 4. Sinh lại stub Python

Chỉ chạy khi sửa `profile.proto`:

```sh
source .venv/bin/activate
make lab-py-proto
```

Hoặc:

```sh
python3 -m grpc_tools.protoc \
  -I doc/handonlab/grpctypes/api \
  --python_out=doc/handonlab/grpc-python \
  --grpc_python_out=doc/handonlab/grpc-python \
  doc/handonlab/grpctypes/api/profile.proto
```

Không sửa tay `profile_pb2.py` / `profile_pb2_grpc.py`.

## 5. Lỗi thường gặp

- **`ModuleNotFoundError: No module named 'grpc'`**: chưa activate venv hoặc
  chưa `pip install -r ...`.
- **`ModuleNotFoundError: No module named 'profile_pb2'`**: chạy script từ sai
  thư mục? `server.py`/`client.py` phải nằm cùng chỗ với stub; chạy đúng đường
  dẫn trong README là đủ.
- **Version mismatch giữa `profile_pb2_grpc.py` và `grpcio`**: cài đúng
  `requirements.txt`, hoặc sinh lại stub bằng `make lab-py-proto`.
- **`connection refused`**: server chưa chạy hoặc client trỏ sai `--addr`.
- **Go client không kết nối được**: Python server phải listen trên địa chỉ mà
  client truy cập được; dùng `0.0.0.0:50051` khi chạy local.
- **Sửa `.proto` nhưng Python vẫn dùng schema cũ**: chưa chạy
  `make lab-py-proto`.
