# Ghi chú triển khai hands-on lab gRPC

Tài liệu này ghi lại mục đích, cách tổ chức thư mục và các quy ước của phần lab
gRPC trong `doc/handonlab/`. Lab **không thuộc game chính**; nó là ví dụ độc lập
để học gRPC và kiểm tra giao tiếp Go ↔ Python.

## 1. Mục đích

- Minh họa cách gRPC truyền nhiều kiểu dữ liệu: scalar, enum, `bytes`,
  `repeated`, nested message, `map`.
- Minh họa 2 kiểu RPC:
  - `SaveProfile`: unary (1 request → 1 response).
  - `ListProfiles`: server-streaming (1 request → nhiều response).
- Chứng minh một hợp đồng `.proto` dùng được cho cả Go và Python.
- Làm starter code cho bài lab 30–45 phút: người học chạy được ngay, sau đó
  sửa `.proto`, sinh lại code và thêm bài tập.

## 2. Cấu trúc thư mục

```text
doc/handonlab/
├── README.md                     # đề bài lab, các bước, bài tập + đáp án
├── docs.md                       # bản nộp theo mẫu/Rubric D0–D3
├── NOTES.md                      # ghi chú triển khai (file này)
├── handonlab.png                 # ảnh yêu cầu/rubric gốc
├── ddg.chat.txt                  # ghi chú tham khảo về gRPC
├── grpctypes/                    # bản Go
│   ├── api/
│   │   ├── profile.proto         # hợp đồng API duy nhất
│   │   ├── profile.pb.go         # message Go sinh tự động
│   │   └── profile_grpc.pb.go    # stub Go sinh tự động
│   ├── server/main.go            # server Go
│   └── client/main.go            # client Go
└── grpc-python/                  # bản Python, dùng cùng profile.proto
    ├── README.md                 # hướng dẫn riêng cho Python + interop
    ├── requirements.txt          # grpcio, protobuf, grpcio-tools
    ├── profile_pb2.py            # message Python sinh tự động
    ├── profile_pb2_grpc.py       # stub Python sinh tự động
    ├── server.py                 # server Python
    └── client.py                 # client Python
```

## 3. Thành phần chính

| Thành phần | Vai trò |
|---|---|
| `grpctypes/api/profile.proto` | Nguồn sự thật duy nhất cho service và message |
| `grpctypes/server` | Server Go, lưu profile trong RAM, in toàn bộ trường |
| `grpctypes/client` | Client Go, gửi profile đầy đủ kiểu và đọc stream |
| `grpc-python/server.py` | Server Python, cùng hành vi với server Go |
| `grpc-python/client.py` | Client Python, cùng dữ liệu mẫu với client Go |
| `*_pb2*.py`, `*.pb.go` | Code sinh tự động, không sửa tay |

## 4. Luồng hoạt động

### Unary RPC — `SaveProfile`

```text
Client (Go/Python)
  → tạo Profile
  → gọi SaveProfile(ProfileRequest)
  → gRPC/HTTP2 + protobuf
  → Server kiểm tra name, điền updated_at_unix, lưu RAM
  → trả ProfileResponse
  → Client in success/summary/saved_profile
```

### Server-streaming RPC — `ListProfiles`

```text
Client gọi ListProfiles(limit)
  → Server giữ kết nối
  → gửi từng Profile bằng stream.Send()/yield
  → Client đọc từng message bằng Recv()/vòng lặp for
  → kết thúc khi hết dữ liệu (io.EOF / StopIteration)
```

## 5. Kiểu dữ liệu trong `Profile`

| `.proto` | Go | Python |
|---|---|---|
| `string name` | `string` | `str` |
| `int32 age` | `int32` | `int` |
| `int64 updated_at_unix` | `int64` | `int` |
| `bool active` | `bool` | `bool` |
| `double score` | `float64` | `float` |
| `repeated string tags` | `[]string` | `list[str]` / `RepeatedScalarContainer` |
| `Address address` | `*api.Address` | `profile_pb2.Address` |
| `Role role` | `api.Role` | `profile_pb2.Role` |
| `bytes avatar` | `[]byte` | `bytes` |
| `map<string, string> attributes` | `map[string]string` | `dict`/`ScalarMapContainer` |

## 6. Lệnh thường dùng

Chạy từ thư mục gốc repo:

```sh
# Sinh lại stub Go sau khi sửa proto
make lab-proto

# Sinh lại stub Python sau khi sửa proto
source .venv/bin/activate
make lab-py-proto

# Server/client Go
go run ./doc/handonlab/grpctypes/server
go run ./doc/handonlab/grpctypes/client

# Server/client Python
python3 doc/handonlab/grpc-python/server.py
python3 doc/handonlab/grpc-python/client.py
```

## 7. Giao tiếp Go ↔ Python

Vì cả hai bên dùng chung `profile.proto`, không cần biết ngôn ngữ của nhau.
Bốn tổ hợp đều hợp lệ:

| Server | Client | Ghi chú |
|---|---|---|
| Go | Go | Cấu hình mặc định của lab |
| Python | Python | Kiểm tra Python chạy độc lập |
| Python | Go | Chứng minh Go client gọi server Python |
| Go | Python | Chứng minh Python client gọi server Go |

Chỉ cần cùng địa chỉ/cổng (mặc định `localhost:50051`). Dừng server cũ trước
khi đổi server Go ↔ Python.

## 8. Quy ước

- Sửa `.proto` là bước đầu tiên khi đổi schema; sau đó chạy `make lab-proto`
  và `make lab-py-proto`.
- Không sửa tay các file sinh tự động:
  `profile.pb.go`, `profile_grpc.pb.go`, `profile_pb2.py`,
  `profile_pb2_grpc.py`.
- Không thêm lab này vào build/game chính; đây là tài liệu học tập độc lập.
- Giữ nguyên tên trường và số field khi chỉ thêm trường mới; không tái sử dụng
  số field đã xóa.
- `insecure_channel` / `insecure.NewCredentials()` chỉ dùng cho local lab.
  Production phải dùng TLS.
- Thư mục `.venv/`, `__pycache__/`, `*.pyc` đã nằm trong `.gitignore`.

## 9. Lỗi thường gặp

| Lỗi | Nguyên nhân | Cách xử lý |
|---|---|---|
| `protoc-gen-go: program not found` | Thiếu plugin hoặc thiếu `PATH` | `go install ...` rồi thêm `$(go env GOPATH)/bin` vào `PATH` |
| `ModuleNotFoundError: grpc` | Chưa cài/activate venv | `source .venv/bin/activate` và cài `requirements.txt` |
| `ModuleNotFoundError: profile_pb2` | Chạy sai thư mục hoặc thiếu stub | Chạy đúng file trong `grpc-python/`; sinh lại stub nếu cần |
| `connection refused` | Server chưa chạy/sai cổng | Chạy server trước, kiểm tra `--addr` |
| `Address already in use` | Còn server cũ trên cổng 50051 | Dừng tiến trình cũ rồi chạy lại |
| `Unimplemented` | Quên đăng ký service | Gọi `RegisterProfileServiceServer` / `add_..._to_server` |
| Stream không kết thúc | Client chỉ đọc 1 message | Đọc đến `io.EOF` (Go) hoặc hết vòng lặp (Python) |
| Version mismatch Python | `grpcio`/`protobuf` cũ hơn stub | Cài đúng `requirements.txt` hoặc sinh lại stub |

## 10. Ghi chú khi mở rộng

- Thêm RPC unary mới: khai báo trong `.proto`, sinh lại 2 bên, cài đặt ở cả
  server Go và Python nếu muốn giữ interop.
- Thêm client-streaming/bidirectional streaming: Python dùng
  `stream_stream`/`stream_unary`, Go dùng `grpc.ClientStream`; nhớ cập nhật
  README và bài tập.
- Nếu đổi `go_package`, phải cập nhật import trong `server/main.go` và
  `client/main.go`, sau đó chạy lại `make lab-proto`.
- Nếu đổi `package` trong `.proto`, stub Python vẫn chạy nhưng descriptor đổi;
  nên giữ nguyên `handonlab.grpctypes.v1` để tránh lệch với Go.
