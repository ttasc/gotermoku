# Hands-on Lab: gRPC truyền nhiều kiểu dữ liệu

**Chủ đề (D0):** Dùng gRPC + Protocol Buffers để truyền một hồ sơ gồm nhiều
kiểu dữ liệu khác nhau, kèm một server-streaming RPC. So sánh với việc tự đóng
gói/parse bằng raw TCP.

**Thời lượng:** 30–45 phút.

> Ghi chú triển khai, cấu trúc thư mục và quy ước: xem [`NOTES.md`](./NOTES.md).
> Bản nộp theo mẫu Rubric D0–D3: xem [`docs.md`](./docs.md).

## 1. Mục tiêu

Sau lab, người học:

- Đọc và sửa được service/message trong file `.proto`.
- Sinh mã Go từ `.proto` bằng `protoc` + plugin.
- Chạy gRPC server/client và gọi được unary RPC.
- Truyền đúng các kiểu: `string`, `int32`, `int64`, `bool`, `double`,
  `bytes`, `enum`, `repeated`, nested message, `map`.
- Phân biệt unary RPC và server-streaming RPC.
- Giải thích vì sao gRPC giảm công việc so với tự thiết kế giao thức trên raw
  TCP: có hợp đồng API, sinh code, đóng khung message, mã lỗi chuẩn, deadline,
  streaming và metadata.

## 2. Chuẩn bị

- Go 1.26+ (repo đang dùng `go 1.26.2`).
- `protoc`.
- Hai plugin Go:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
export PATH="$PATH:$(go env GOPATH)/bin"
```

Mã Go đã sinh sẵn trong `api/` để người học không mất thời gian setup. Chỉ cần
sinh lại khi sửa `.proto`.

## 3. Cấu trúc

```text
doc/handonlab/grpctypes/
├── api/
│   ├── profile.proto        # hợp đồng API
│   ├── profile.pb.go        # message được sinh (không sửa tay)
│   └── profile_grpc.pb.go   # client/server stub được sinh (không sửa tay)
├── server/main.go           # cài đặt SaveProfile + ListProfiles
└── client/main.go           # gọi unary + đọc server stream
```

## 4. Các bước thực hiện

### Bước 1 — Đọc hợp đồng API

Mở `api/profile.proto`. Chú ý:

- `SaveProfile` là **unary**: 1 request → 1 response.
- `ListProfiles` là **server-streaming**: 1 request → nhiều response.
- Message `Profile` cố ý chứa nhiều kiểu dữ liệu.

Bảng ánh xạ chính:

| `.proto` | Go | Ghi chú |
|---|---|---|
| `string` | `string` | UTF-8 |
| `int32` | `int32` | |
| `int64` | `int64` | timestamp Unix |
| `bool` | `bool` | |
| `double` | `float64` | |
| `bytes` | `[]byte` | nhị phân, ví dụ avatar |
| `repeated string` | `[]string` | danh sách |
| `Address` | `*api.Address` | nested message |
| `Role` | `api.Role` | enum |
| `map<string, string>` | `map[string]string` | |

### Bước 2 — Sinh mã Go (chỉ khi sửa `.proto`)

Từ thư mục gốc repo:

```sh
make lab-proto
```

Hoặc chạy trực tiếp:

```sh
PATH="$PATH:$(go env GOPATH)/bin" protoc \
  --go_out=. --go_opt=module=github.com/ttasc/gotermoku \
  --go-grpc_out=. --go-grpc_opt=module=github.com/ttasc/gotermoku \
  doc/handonlab/grpctypes/api/profile.proto
```

Không sửa tay `profile.pb.go` / `profile_grpc.pb.go`.

### Bước 3 — Chạy server

Terminal 1:

```sh
go run ./doc/handonlab/grpctypes/server
```

Server lắng nghe `:50051` và in từng trường nhận được.

### Bước 4 — Chạy client

Terminal 2:

```sh
go run ./doc/handonlab/grpctypes/client
go run ./doc/handonlab/grpctypes/client -name "Bình"
```

Client gửi `Profile` đầy đủ kiểu, in response của `SaveProfile`, rồi đọc
`ListProfiles` bằng vòng lặp `stream.Recv()` đến `io.EOF`.

### Bước 5 — Quan sát và sửa

1. Thêm `tags` hoặc `attributes` mới trong client, xem server in ra.
2. Đổi `Role` sang `ROLE_READER`, xem `summary` thay đổi.
3. Chạy client nhiều lần, xem `ListProfiles` trả về nhiều hồ sơ trên một stream.

## 5. Bài tập và đáp án

### Bài 1 — Validate dữ liệu

Yêu cầu: nếu `name` rỗng, trả lỗi `InvalidArgument` thay vì lưu hồ sơ.

Đáp án (đã có trong `server/main.go`):

```go
if profile == nil || profile.GetName() == "" {
    return nil, status.Error(codes.InvalidArgument, "name is required")
}
```

Thử bằng cách sửa client thành `Name: ""`; client sẽ nhận lỗi và `log.Fatalf`.

### Bài 2 — Thêm trường và sinh lại code

Yêu cầu: thêm `string email = 11;` vào message `Profile`, in email ở server và
client.

Đáp án:

1. Thêm dòng sau vào `Profile` trong `api/profile.proto`:

   ```protobuf
   string email = 11;
   ```

2. Chạy `make lab-proto`.
3. Trong client, thêm `Email: "an@example.com"`.
4. Trong server, thêm `profile.GetEmail()` vào `fmt.Printf`.
5. Chạy lại; không cần viết parser hay sửa framing.

### Bài 3 — Giới hạn server stream

Yêu cầu: `Limit = 2` chỉ trả tối đa 2 hồ sơ.

Đáp án (đã có trong `server/main.go`):

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

Thử: chạy client 3 lần rồi gọi `ListProfiles` với `Limit: 2`.

## 6. Lỗi thường gặp

- **`protoc-gen-go: program not found`**: chưa thêm `$(go env GOPATH)/bin` vào
  `PATH`, hoặc chưa `go install` plugin.
- **`undefined: api.ProfileServiceServer`**: chưa sinh lại code sau khi sửa
  `.proto`, hoặc `go_package` sai.
- **`connection refused`**: server chưa chạy, sai `-addr`, hoặc client trỏ sai
  cổng.
- **Quên `api.RegisterProfileServiceServer`**: server chạy nhưng RPC trả
  `Unimplemented`.
- **Sửa file `.pb.go` bằng tay**: thay đổi sẽ mất sau lần sinh lại; luôn sửa
  `.proto`.
- **Dùng `insecure.NewCredentials()` ngoài local**: lab dùng cho đơn giản;
  production cần TLS.
- **Stream không kết thúc**: client phải `Recv()` đến `io.EOF`, không chỉ đọc
  một message.

## 7. Vì sao gRPC hơn raw TCP cho bài này?

Raw TCP chỉ cho một luồng byte có thứ tự. Nếu tự làm, nhóm phải tự thiết kế:

- ranh giới message (length-prefix/framing),
- định dạng dữ liệu và versioning,
- mã lỗi,
- timeout/deadline,
- multiplexing và streaming.

gRPC cung cấp sẵn hợp đồng `.proto`, sinh client/server, Protocol Buffers,
HTTP/2 multiplexing, deadline, metadata, mã lỗi chuẩn và 4 kiểu RPC. Đổi lại,
cần thêm toolchain `protoc` và học khái niệm mới.

## 8. Bài tập mở rộng (tùy chọn)

- Thêm `rpc DeleteProfile(DeleteProfileRequest) returns (DeleteProfileResponse)`
  để luyện unary RPC thứ hai.
- Thêm client-streaming `UploadProfiles(stream Profile) returns (UploadSummary)`.
- Thêm bidirectional streaming và so sánh với server-streaming.

## 9. Bản Python và giao tiếp Go ↔ Python

Cùng `api/profile.proto` có thêm bản Python tại `grpc-python/`. Vì protobuf
độc lập ngôn ngữ, 4 tổ hợp server/client đều chạy được:

| Server | Client |
|---|---|
| Go | Go |
| Python | Python |
| Python | Go |
| Go | Python |

Chạy nhanh:

```sh
python3 -m venv .venv
source .venv/bin/activate
pip install -r doc/handonlab/grpc-python/requirements.txt

# Python server + Go client
python3 doc/handonlab/grpc-python/server.py
go run ./doc/handonlab/grpctypes/client -name "Go gọi Python"

# Go server + Python client
go run ./doc/handonlab/grpctypes/server
python3 doc/handonlab/grpc-python/client.py --name "Python gọi Go"
```

Dừng server đang chạy trước khi đổi server Go ↔ Python (cùng cổng `50051`).

Sinh lại stub Python sau khi sửa proto:

```sh
source .venv/bin/activate
make lab-py-proto
```

Chi tiết và lỗi thường gặp: `grpc-python/README.md`.
