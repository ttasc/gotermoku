# MỤC RIÊNG: HANDS-ON LAB (BẮT BUỘC, NỘP CÙNG BÁO CÁO)

Đây là tài liệu 30–45 phút để một nhóm **KHÁC** làm theo và tự học lại đúng
phần công nghệ nhóm mình đã tìm hiểu. Đây không phải một chương trong báo cáo,
mà là một file riêng — chấm theo mã **D0–D3** của Rubric.

> **Chủ đề:** gRPC truyền nhiều kiểu dữ liệu và giao tiếp đa ngôn ngữ Go ↔
> Python.

---

## D0 — LỰA CHỌN CHỦ ĐỀ LAB (5 điểm)

**Chủ đề lab:** Xây dựng một gRPC service bằng Go và Python, dùng chung một hợp
đồng `profile.proto`, truyền được nhiều kiểu dữ liệu (scalar, enum, bytes,
repeated, nested message, map), có unary RPC và server-streaming RPC.

**Vì sao chọn chủ đề này:**

- gRPC là công nghệ nhóm đã tìm hiểu, có thể chứng minh bằng code chạy thật.
- Chủ đề vượt qua mức CRUD cơ bản: người học phải hiểu hợp đồng `.proto`,
  sinh code, đóng khung message, xử lý stream và ghép hai ngôn ngữ khác nhau.
- Kết quả học được kiểm chứng được ngay: Go client gọi Python server và
  Python client gọi Go server mà không cần viết lại giao thức.

**Người học làm được gì sau lab:**

- Tự định nghĩa service/message trong `.proto` và sinh stub Go/Python.
- Gọi unary RPC và đọc server-streaming RPC.
- Thêm trường mới, sinh lại code và kiểm tra hai bên Go/Python vẫn tương thích.
- Giải thích vì sao gRPC giảm công việc so với tự đóng gói dữ liệu trên raw
  TCP: có hợp đồng API, sinh code, mã lỗi chuẩn, deadline, metadata và
  streaming.

---

## NỘI DUNG TỐI THIỂU CỦA MỘT BÀI HANDS-ON LAB

### 1. Mục tiêu học tập rõ ràng — học xong làm được việc gì

Sau 30–45 phút, người học:

1. Đọc và sửa được `ProfileService` trong `api/profile.proto`.
2. Sinh lại mã Go bằng `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`.
3. Sinh lại mã Python bằng `grpc_tools.protoc`.
4. Chạy server/client Go và server/client Python.
5. Gọi được unary RPC `SaveProfile` và server-streaming RPC `ListProfiles`.
6. Thêm một trường vào message, sinh lại code và xác nhận Go ↔ Python vẫn
   giao tiếp được.

### 2. Điều kiện môi trường chuẩn bị trước

**Bắt buộc:**

- Go 1.26+ (repo dùng `go 1.26.2`).
- Python 3.11+.
- `protoc` (Protocol Buffers compiler).
- Plugin Go:

  ```sh
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
  export PATH="$PATH:$(go env GOPATH)/bin"
  ```

- Thư viện Python:

  ```sh
  python3 -m venv .venv
  source .venv/bin/activate
  pip install -r doc/handonlab/grpc-python/requirements.txt
  ```

**Starter code đã có sẵn:**

```text
doc/handonlab/
├── grpctypes/
│   ├── api/profile.proto          # hợp đồng API dùng chung
│   ├── api/profile.pb.go
│   ├── api/profile_grpc.pb.go
│   ├── server/main.go             # server Go
│   └── client/main.go             # client Go
└── grpc-python/
    ├── profile_pb2.py
    ├── profile_pb2_grpc.py
    ├── server.py                  # server Python
    └── client.py                  # client Python
```

Chạy từ thư mục gốc repo. Mã sinh tự động đã có sẵn để người học không mất
thời gian setup; chỉ sinh lại khi sửa `.proto`.

### 3. Các bước thực hiện — đánh số, kèm code khởi đầu

#### Bước 1 — Đọc hợp đồng API

Mở `doc/handonlab/grpctypes/api/profile.proto`. Xác định:

- `SaveProfile` là unary RPC: 1 request → 1 response.
- `ListProfiles` là server-streaming RPC: 1 request → nhiều response.
- `Profile` chứa nhiều kiểu dữ liệu.

Trích đoạn starter code:

```protobuf
service ProfileService {
  rpc SaveProfile(ProfileRequest) returns (ProfileResponse);
  rpc ListProfiles(ListProfilesRequest) returns (stream Profile);
}

message Profile {
  string name = 1;
  int32 age = 2;
  int64 updated_at_unix = 3;
  bool active = 4;
  double score = 5;
  repeated string tags = 6;
  Address address = 7;
  Role role = 8;
  bytes avatar = 9;
  map<string, string> attributes = 10;
}
```

#### Bước 2 — Sinh lại mã (chỉ khi sửa `.proto`)

```sh
# Go
make lab-proto

# Python (trong venv đã cài grpcio-tools)
source .venv/bin/activate
make lab-py-proto
```

Hoặc chạy trực tiếp:

```sh
PATH="$PATH:$(go env GOPATH)/bin" protoc \
  --go_out=. --go_opt=module=github.com/ttasc/gotermoku \
  --go-grpc_out=. --go-grpc_opt=module=github.com/ttasc/gotermoku \
  doc/handonlab/grpctypes/api/profile.proto

python3 -m grpc_tools.protoc \
  -I doc/handonlab/grpctypes/api \
  --python_out=doc/handonlab/grpc-python \
  --grpc_python_out=doc/handonlab/grpc-python \
  doc/handonlab/grpctypes/api/profile.proto
```

Không sửa tay các file `*.pb.go`, `*_pb2.py`, `*_pb2_grpc.py`.

#### Bước 3 — Chạy server Go + client Go

Terminal 1:

```sh
go run ./doc/handonlab/grpctypes/server
```

Terminal 2:

```sh
go run ./doc/handonlab/grpctypes/client
```

Quan sát server in đủ trường: `name`, `age`, `updated_at`, `active`, `score`,
`tags`, `city`, `country`, `role`, `avatar`, `attributes`.

#### Bước 4 — Chạy client Python gọi server Go

Giữ nguyên server Go ở Terminal 1. Terminal 2:

```sh
source .venv/bin/activate
python3 doc/handonlab/grpc-python/client.py --name "Python gọi Go"
```

#### Bước 5 — Chạy server Python + client Go

Dừng server Go trước (cùng cổng `50051`). Terminal 1:

```sh
source .venv/bin/activate
python3 doc/handonlab/grpc-python/server.py
```

Terminal 2:

```sh
go run ./doc/handonlab/grpctypes/client -name "Go gọi Python"
```

Nếu cả hai chiều đều chạy, hợp đồng `.proto` đã được chứng minh là độc lập
ngôn ngữ.

#### Bước 6 — Quan sát server-streaming

Chạy client nhiều lần, rồi gọi `ListProfiles`:

- Go client đọc stream bằng vòng lặp `stream.Recv()` đến `io.EOF`.
- Python client đọc stream bằng vòng lặp `for item in stub.ListProfiles(...)`.
- Server giữ một lời gọi và gửi nhiều `Profile`.

### 4. 2–3 bài tập nhỏ kèm đáp án

#### Bài tập 1 — Validate dữ liệu đầu vào

**Yêu cầu:** Nếu `name` rỗng, server trả lỗi `InvalidArgument`, không lưu hồ sơ.

**Đáp án (Go — `server/main.go`):**

```go
if profile == nil || profile.GetName() == "" {
    return nil, status.Error(codes.InvalidArgument, "name is required")
}
```

**Đáp án (Python — `server.py`):**

```python
if profile is None or not profile.name:
    context.abort(grpc.StatusCode.INVALID_ARGUMENT, "name is required")
```

**Kiểm tra:** sửa client thành `name=""`; client phải nhận lỗi thay vì lưu.

#### Bài tập 2 — Thêm trường mới và sinh lại code

**Yêu cầu:** Thêm `string email = 11;` vào `Profile`; in email ở server và
client.

**Đáp án:**

1. Thêm vào `Profile` trong `profile.proto`:

   ```protobuf
   string email = 11;
   ```

2. Chạy `make lab-proto` và `make lab-py-proto`.
3. Go client: thêm `Email: "an@example.com"`.
4. Python client: thêm `email="an@example.com"`.
5. Server Go: thêm `profile.GetEmail()` vào `fmt.Printf`.
6. Server Python: thêm `profile.email` vào `print`.
7. Chạy lại cả 4 tổ hợp; không cần viết parser/framing mới.

#### Bài tập 3 — Giới hạn số lượng trong server stream

**Yêu cầu:** `ListProfiles` với `Limit = 2` chỉ trả tối đa 2 hồ sơ.

**Đáp án (Go):**

```go
limit := int(req.GetLimit())
for i, profile := range snapshot {
    if limit > 0 && i >= limit {
        break
    }
    if err := stream.Send(profile); err != nil {
        return err
    }
}
```

**Đáp án (Python):**

```python
for index, profile in enumerate(snapshot):
    if request.limit > 0 and index >= request.limit:
        break
    yield profile
```

**Kiểm tra:** chạy client 3 lần, sau đó gọi `ListProfiles` với `limit=2` và
đếm số message nhận được.

### 5. Mục "Lỗi thường gặp"

Các lỗi nhóm đã thực sự gặp khi làm lab:

| Lỗi | Nguyên nhân | Cách khắc phục |
|---|---|---|
| `protoc-gen-go: program not found` | Chưa cài plugin hoặc chưa thêm `$(go env GOPATH)/bin` vào `PATH` | `go install ...` rồi `export PATH="$PATH:$(go env GOPATH)/bin"` |
| `ModuleNotFoundError: No module named 'grpc'` | Chưa activate venv hoặc chưa cài requirements | `source .venv/bin/activate` rồi `pip install -r doc/handonlab/grpc-python/requirements.txt` |
| `ModuleNotFoundError: No module named 'profile_pb2'` | Chạy script sai thư mục hoặc thiếu stub | Chạy đúng đường dẫn trong README; sinh lại stub bằng `make lab-py-proto` |
| `connection refused` | Server chưa chạy hoặc client trỏ sai `--addr` | Chạy server trước; kiểm tra cổng/địa chỉ |
| `Address already in use` | Còn server cũ trên cổng `50051` | Dừng tiến trình cũ rồi chạy lại |
| RPC trả `Unimplemented` | Quên đăng ký service | Go: `RegisterProfileServiceServer`; Python: `add_ProfileServiceServicer_to_server` |
| Stream không kết thúc | Client chỉ đọc một message | Go đọc đến `io.EOF`; Python lặp hết iterator |
| Version mismatch Python | `grpcio`/`protobuf` cũ hơn stub | Cài đúng `requirements.txt` hoặc sinh lại stub |
| Go client không gọi được Python server | Python server bind sai địa chỉ | Dùng `--addr 0.0.0.0:50051` khi chạy local |
| Sửa `.proto` nhưng lỗi schema | Chưa sinh lại một trong hai bên | Chạy cả `make lab-proto` và `make lab-py-proto` |

---

## Phụ lục: đối chiếu Rubric D0–D3

| Mã | Yêu cầu | Phần đáp ứng |
|---|---|---|
| **D0** | Chọn chủ đề lab, giải thích lý do và giá trị học tập | Mục D0 |
| **D1** | Mục tiêu rõ ràng, điều kiện môi trường, các bước chạy được | Mục 1, 2, 3 |
| **D2** | 2–3 bài tập nhỏ kèm đáp án | Mục 4 |
| **D3** | Lỗi thường gặp thực tế + cách khắc phục | Mục 5 |

## Phụ lục: file tham chiếu

- Đề bài đầy đủ và hướng dẫn từng bước: `doc/handonlab/README.md`
- Ghi chú triển khai, cấu trúc thư mục, quy ước: `doc/handonlab/NOTES.md`
- Hướng dẫn Python + interop: `doc/handonlab/grpc-python/README.md`
- Hợp đồng API: `doc/handonlab/grpctypes/api/profile.proto`
