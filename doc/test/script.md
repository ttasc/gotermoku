## GIAI ĐOẠN 0: XÁC MINH TOÀN VẸN & BIÊN DỊCH

Mở **Terminal 1**, tại thư mục gốc của dự án:

```sh
# 1. Dọn dẹp bản build cũ và sinh mã Protobuf
make clean
make proto

# 2. Chạy toàn bộ Integration Tests với Race Detector
make test
```
*Kết quả mong đợi:* Toàn bộ 4 bài test (`TestLAN_MultiplayerIntegration`, `TestOnline_MatchmakingAndEloUpdate`, `TestOnline_PrivateRoom`, `TestOnline_DisconnectAndRejoin`) đều báo **`PASS`** mà không có bất kỳ cảnh báo Data Race nào.

Tiếp theo, biên dịch 2 file nhị phân:
```sh
make all
```
*Kết quả:* Xuất hiện 2 file thực thi độc lập: `./gotermoku` (Client) và `./gotermoku-server` (Server).

---

## GIAI ĐOẠN 1: OFFLINE & MẠNG NỘI BỘ LAN P2P

### Kịch bản 1: Chế độ Hotseat (Chơi 2 người trên 1 máy) & TUI Controls
* **Terminal 1:**
  ```sh
  ./gotermoku -s 10x10
  ```
* **Các bước kiểm tra trên giao diện bàn cờ:**
  1. **Vim Keys:** Bấm `h, j, k, l` hoặc các phím mũi tên `↑, ↓, ←, →` để di chuyển con trỏ.
  2. **Chuột:** Click chuột trái lần 1 để định vị con trỏ, click lần 2 tại ô đó để đặt quân.
  3. **Đặt quân:** Bấm `Space` hoặc `Enter`. Lượt đi trên thanh trạng thái luân phiên đổi màu giữa `WHITE` và `BLACK`.
  4. **Thắng cuộc:** Đánh đủ 5 quân liên tiếp (ngang, dọc hoặc chéo). Banner màu xanh lá `* WHITE WINS! *` xuất hiện, các ô thắng sáng màu.
  5. **Chơi lại:** Bấm phím `r` hoặc `R` để reset bàn cờ chơi ván mới.
  6. **Thoát:** Bấm `Esc` hoặc `Ctrl+C`. Terminal được dọn dẹp sạch sẽ, phục hồi cooked mode chuẩn POSIX.

---

### Kịch bản 2: Đấu với AI Bot (Single-Player)
* **Terminal 1:**
  ```sh
  ./gotermoku --bot -s 12x12
  ```
* **Kiểm tra:**
  1. Bạn cầm quân Trắng đi trước. Đặt 1 quân bất kỳ.
  2. Sau khoảng 400ms suy nghĩ, Bot (quân Đen) tự động phản hồi nước đi tối ưu nhất.
  3. Thử tạo chuỗi 3 hoặc 4 quân mở để kiểm tra Bot có chủ động chặn hai đầu hay không. Bấm `Esc` để thoát.

---

### Kịch bản 3: Đấu mạng ngang hàng LAN P2P (Host & Client kết nối trực tiếp)
* **Terminal 1 (Host - Quân Trắng):**
  ```sh
  ./gotermoku --host -s 15x15
  ```
  *(Host tự động lắng nghe cổng mặc định `3333`: `Starting gRPC Host... Waiting for client to connect on port 3333...`)*
* **Terminal 2 (Client - Quân Đen):**
  ```sh
  ./gotermoku --join 127.0.0.1
  ```
  *(Client tự động kết nối vào cổng mặc định `3333` mà không cần gõ `--port`)*
* **Kiểm tra:**
  1. Cả 2 màn hình đồng loạt mở bàn cờ $15 \times 15$.
  2. Host đánh $\rightarrow$ Client cập nhật tức thì. Client đánh $\rightarrow$ Host cập nhật tức thì.
  3. Tại Terminal 2, bấm `Ctrl+C` thoát game $\rightarrow$ Terminal 1 lập tức hiện banner đỏ: `Opponent disconnected.`

---

## GIAI ĐOẠN 2: CENTRAL SERVER - ELO MATCHMAKING & BBOLT STORAGE

Mở **Terminal 0** (dùng riêng để chạy Server trung tâm):
```sh
# Xóa database cũ để bắt đầu từ đầu
rm -f gotermoku.db

# Khởi chạy Central Server (Tự động nhận cổng 3333, bàn cờ 15x15)
./gotermoku-server --rows 15 --cols 15
```
*(Server log: `[SERVER] Listening on port 3333 (board: 15x15, db: gotermoku.db)`)*

---

### Kịch bản 4: Đăng ký & Ghép trận tự động theo dải Elo
* **Terminal 1 (Alice):**
  ```sh
  ./gotermoku --server localhost --register --user alice --pass 123456
  ```
  *(In ra: `[AUTH] Registered successfully! Elo: 1200` và vào hàng đợi tìm trận)*
* **Terminal 2 (Bob):**
  ```sh
  ./gotermoku --server localhost --register --user bob --pass 123456
  ```
  *(In ra: `[AUTH] Registered successfully! Elo: 1200`)*
* **Kết quả mong đợi:**
  * Server nhận diện dải điểm ngang bằng ($\Delta Elo = 0 \le 100$), lập tức ghép cặp Alice và Bob.
  * Cả 2 terminal in thông tin trận đấu (kèm **Match ID**) và chuyển vào TUI:
    ```text
    [MATCH READY] Match ID: 8a3f91cb...
    Opponent: bob (1200 Elo) | You: White (1200 Elo)
    ```
  * Thanh trạng thái hiển thị chuẩn: `WHITE (You - 1200)` vs `BLACK (bob - 1200)`.

---

### Kịch bản 5: Đăng nhập lại & Kiểm tra Cập nhật Điểm Elo trong Database bbolt
* **Tiếp tục từ Kịch bản 4:**
  1. Người cầm quân Trắng đánh 5 nước liên tiếp để giành chiến thắng.
  2. Sau khi thắng, cả 2 bên bấm `Esc` để thoát game.
* **Đăng nhập lại để kiểm tra số điểm được lưu trên đĩa:**
  * Tại **Terminal 1 (Alice thắng):**
    ```sh
    ./gotermoku --server localhost --user alice --pass 123456
    ```
    *Kết quả:* `[AUTH] Login successful! Elo: 1216` (Được cộng $+16$ điểm). Bấm `Ctrl+C` hủy tìm trận.
  * Tại **Terminal 2 (Bob thua):**
    ```sh
    ./gotermoku --server localhost --user bob --pass 123456
    ```
    *Kết quả:* `[AUTH] Login successful! Elo: 1184` (Bị trừ $-16$ điểm). Bấm `Ctrl+C` hủy tìm trận.

---

## GIAI ĐOẠN 3: PHÒNG RIÊNG & PHỤC HỒI KẾT NỐI (REJOIN)

Server tại **Terminal 0** vẫn tiếp tục chạy.

---

### Kịch bản 6: Tạo phòng riêng (`--create-room`) & Vào phòng bằng mã Code (`--join-room`)
* **Terminal 1 (Alice tạo phòng):**
  ```sh
  ./gotermoku --server localhost --user alice --pass 123456 --create-room
  ```
  *Terminal 1 in mã phòng ngẫu nhiên 6 ký tự:*
  ```text
  ==================================================
   [PRIVATE ROOM CREATED] ROOM CODE: 7K9PX2
   Share this code with your friend to start playing!
  ==================================================
  [WAITING] Waiting for opponent in room 7K9PX2...
  ```
* **Terminal 2 (Bob nhập mã phòng để vào chơi cùng):**
  *(Thay `7K9PX2` bằng mã code thực tế được in trên Terminal 1)*
  ```sh
  ./gotermoku --server localhost --user bob --pass 123456 --join-room 7K9PX2
  ```
* **Kết quả mong đợi:**
  * Server nhận diện mã phòng, hủy phòng chờ và kích hoạt trận đấu ngay lập tức.
  * Tiêu đề TUI trên cả hai terminal hiển thị mã phòng: `G O T E R M O K U  [ROOM: 7K9PX2]`.

---

### Kịch bản 7: Rớt mạng đột ngột $\rightarrow$ Ân hạn 45 giây $\rightarrow$ Rejoin kết nối lại
* **Các bước thực hiện:**
  1. Trong trận đấu ở Kịch bản 6, lưu ý dòng **Match ID** đã in trên console trước khi vào TUI (ví dụ: `4b2a9d81e0f37c1a`).
  2. Cho quân Trắng đánh 1 nước bất kỳ trên bàn cờ.
  3. **Mô phỏng rớt mạng:** Tại **Terminal 2 (Bob)**, bấm `Ctrl+C` để tắt terminal đột ngột.
  4. **Quan sát Terminal 1 (Alice):**
     * Trận đấu lập tức bị đóng băng (Alice bấm đánh cờ không có tác dụng).
     * Xuất hiện banner màu đỏ/cam đếm ngược thời gian ân hạn:
       `* OPPONENT DISCONNECTED - WAITING FOR RECONNECT (45s) *`
       *(Số giây tự động đếm lùi: `(40s)`, `(35s)`,...)*
  5. **Mở lại kết nối (Bob Rejoin):**
     Tại **Terminal 2**, chạy lệnh với cờ `--rejoin`:
     ```sh
     ./gotermoku --server localhost --user bob --pass 123456 --rejoin 4b2a9d81e0f37c1a
     ```
  6. **Kết quả mong đợi sau khi Rejoin:**
     * Terminal 2 nhận thông báo: `[REJOIN] Checking match '4b2a9d81e0f37c1a' status... Match is active and ready for rejoin`.
     * Terminal 2 tự động mở lại TUI với toàn bộ các nước cờ cũ được khôi phục nguyên vẹn.
     * Banner cảnh báo trên Terminal 1 biến mất, thanh trạng thái hiển thị `reconnected. Match resumed!`.
     * Cả hai bên tiếp tục đánh cờ bình thường cho đến hết trận.

---

## GIAI ĐOẠN 4: Bảo mật Đường truyền với Mã hóa TLS

* **Mục tiêu:** Kiểm tra khả năng bảo vệ đường truyền, mã hóa mật khẩu và nước cờ mà **không cần chạy bất kỳ lệnh `openssl` nào và không sinh file rác trên đĩa**.

### 1. Khởi động Server với mã hóa TLS tự sinh trong RAM
Tại **Terminal 0**, tắt server cũ (`Ctrl+C`) và chạy:
```sh
./gotermoku-server --tls
```
*Server lập tức thông báo:*
```text
[SECURITY] TLS enabled (Zero-Config: Generated ephemeral ECDSA certificate in-memory).
[SERVER] Listening on port 3333 (board: 20x30, db: gotermoku.db)
```
*(Cặp khóa và chứng chỉ ECDSA P-256 được sinh trực tiếp trong bộ nhớ RAM trong vòng chưa đầy 2ms, không hề có file `.pem` nào được ghi xuống ổ cứng).*

### 2. Client kết nối an toàn qua cờ `--tls-insecure`
* **Terminal 1 (Alice):**
  ```sh
  ./gotermoku --server localhost --tls-insecure --user alice --pass 123456
  ```
* **Terminal 2 (Bob):**
  ```sh
  ./gotermoku --server localhost --tls-insecure --user bob --pass 123456
  ```
* **Kết quả mong đợi:**
  * Cả hai Client hoàn tất bắt tay TLS bảo mật với Server mà không cần cung cấp bất kỳ file chứng chỉ CA nào.
  * Trận đấu diễn ra trơn tru, toàn bộ dữ liệu truyền qua socket đều được mã hóa theo tiêu chuẩn mật mã học hiện đại.

---

## BẢNG TỔNG KẾT THAM CHIẾU LỆNH NHANH (CHEATSHEET)

| Chế độ / Tính năng | Lệnh Server | Lệnh Client |
| :--- | :--- | :--- |
| **Hotseat (2 người chung máy)** | *Không cần* | `./gotermoku -s 15x15` |
| **Đấu với AI Bot** | *Không cần* | `./gotermoku --bot -s 15x15` |
| **LAN P2P (Host)** | *Không cần* | `./gotermoku --host -s 15x15` |
| **LAN P2P (Client)** | *Không cần* | `./gotermoku --join <IP_HOST>` |
| **Online Ghép trận Elo** | `./gotermoku-server` | `./gotermoku --server <IP> --user <U> --pass <P>` |
| **Tạo phòng riêng** | `./gotermoku-server` | `./gotermoku --server <IP> --user <U> --pass <P> --create-room` |
| **Vào phòng riêng** | `./gotermoku-server` | `./gotermoku --server <IP> --user <U> --pass <P> --join-room <CODE>` |
| **Nối lại ván cờ (Rejoin)** | `./gotermoku-server` | `./gotermoku --server <IP> --user <U> --pass <P> --rejoin <MATCH_ID>` |
| **Zero-Config TLS** | `./gotermoku-server --tls` | `./gotermoku --server <IP> --tls-insecure --user <U> --pass <P>` |
