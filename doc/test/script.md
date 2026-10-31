## GIAI ĐOẠN 0: BIÊN DỊCH DỰ ÁN

Mở **Terminal 1**, chuyển về thư mục gốc của dự án và chạy:
```sh
# 1. Dọn dẹp bản build cũ và sinh mã Protobuf
make clean
make proto

# 2. Biên dịch đồng thời cả client và server nhị phân
make all
```
*Kết quả:* Thư mục gốc xuất hiện 2 file nhị phân thực thi: `gotermoku` (Client) và `gotermoku-server` (Server).

---

## GIAI ĐOẠN 1: KIỂM THỬ OFFLINE & MẠNG NỘI BỘ LAN

### Kịch bản 1: Chế độ Hotseat (2 người chơi chung 1 máy) & Điều khiển
* **Mục tiêu:** Kiểm tra bàn cờ TUI, phím tắt Vim, chuột và luật cờ cơ bản.
* **Thực hiện (Terminal 1):**
  ```sh
  ./gotermoku -s 10x10
  ```
* **Các bước test trên TUI:**
  1. **Điều khiển con trỏ:** Bấm các phím `h, j, k, l` hoặc các phím mũi tên `↑, ↓, ←, →` để di chuyển con trỏ.
  2. **Thao tác chuột:** Click chuột trái 1 lần để chuyển con trỏ tới ô đó, click lần 2 để đặt quân.
  3. **Đặt quân:** Bấm `Space` hoặc `Enter` để đặt quân. Quan sát thanh trạng thái chuyển lượt luân phiên giữa `WHITE` và `BLACK`.
  4. **Kiểm tra luật thắng:** Đánh 5 quân liên tiếp (ngang, dọc hoặc chéo).
     * *Kết quả mong đợi:* Xuất hiện banner màu xanh lá `* WHITE WINS! *` (hoặc `BLACK WINS!`), các ô thắng sáng màu nổi bật.
  5. **Chơi lại:** Bấm phím `r` hoặc `R` để reset bàn cờ chơi ván mới.
  6. **Thoát:** Bấm `Esc` hoặc `Ctrl+C` để thoát game. Terminal tự động phục hồi sạch sẽ về chế độ cooked mode.

---

### Kịch bản 2: Chơi đơn với AI Bot
* **Mục tiêu:** Kiểm tra thuật toán Heuristic AI Bot.
* **Thực hiện (Terminal 1):**
  ```sh
  ./gotermoku --bot -s 12x12
  ```
* **Các bước test:**
  1. Người chơi cầm quân Trắng đi trước. Đặt 1 quân bất kỳ.
  2. *Kết quả mong đợi:* Sau khoảng 400ms suy nghĩ, Bot (quân Đen) tự động đánh nước cờ đáp trả.
  3. Thử tạo chuỗi 3 hoặc 4 quân liên tiếp để kiểm tra xem Bot có chặn đầu phòng thủ không.

---

### Kịch bản 3: Mạng ngang hàng LAN P2P (Host & Client)
* **Mục tiêu:** Kiểm tra chế độ gRPC Streaming trực tiếp P2P không qua Server trung tâm.
* **Thực hiện:**
  * **Terminal 1 (Đóng vai Host - Quân Trắng):**
    ```sh
    ./gotermoku --host --port 3333 -s 15x15
    ```
    *(Terminal 1 sẽ in thông báo: `Starting gRPC Host... Waiting for client to connect on port 3333...`)*
  * **Terminal 2 (Đóng vai Client - Quân Đen):**
    ```sh
    ./gotermoku --join 127.0.0.1 --port 3333
    ```
* **Kết quả mong đợi:**
  * Cả hai terminal đồng loạt mở bàn cờ kích thước $15 \times 15$ do Host quy định.
  * Terminal 1 đánh quân Trắng $\rightarrow$ Xuất hiện tức thì trên Terminal 2.
  * Terminal 2 đánh quân Đen $\rightarrow$ Cập nhật đồng bộ trên Terminal 1.
  * Trên Terminal 2, bấm `Ctrl+C` thoát game $\rightarrow$ Terminal 1 lập tức hiện banner đỏ: `Opponent disconnected.`

---

## GIAI ĐOẠN 2: CENTRAL SERVER - ELO, PHÒNG RIÊNG, REJOIN & TLS

### Khởi động Server trung tâm
Mở **Terminal 0** (dùng riêng để chạy và theo dõi Server log):
```sh
# Xóa database test cũ nếu muốn bắt đầu mới hoàn toàn
rm -f gotermoku.db

# Khởi chạy Central Server (port 4444, bàn cờ mặc định 15x15)
./gotermoku-server --port 4444 --rows 15 --cols 15
```
*(Server thông báo: `[SERVER] Listening on port 4444 (board: 15x15, db: gotermoku.db)`)*

---

### Kịch bản 4: Đăng ký, Đăng nhập & Ghép trận Elo tự động
* **Terminal 1 (Người chơi 1 - Alice):**
  ```sh
  ./gotermoku --server localhost:4444 --register --user alice --pass 123456
  ```
  *(Terminal 1 in: `[AUTH] Registered successfully! Elo: 1200` và chờ ghép đối thủ)*
* **Terminal 2 (Người chơi 2 - Bob):**
  ```sh
  ./gotermoku --server localhost:4444 --register --user bob --pass 123456
  ```
* **Kết quả mong đợi:**
  * Matchmaker tự động ghép cặp 2 người chơi có cùng Elo 1200.
  * Cả 2 terminal in thông tin trận đấu (kèm **Match ID**) và chuyển vào bàn cờ TUI.
  * Trên thanh trạng thái TUI, hiển thị chính xác tên và điểm Elo: ví dụ `WHITE (You - 1200)` vs `BLACK (bob - 1200)`.

---

### Kịch bản 5: Kiểm tra Cập nhật Điểm Elo sau Ván Đấu (bbolt Database)
* **Tiếp tục từ Kịch bản 4:**
  1. Người chơi cầm quân Trắng đánh 5 quân liên tiếp để chiến thắng.
  2. Sau khi thắng, cả 2 bên bấm `Esc` để thoát game.
* **Kiểm tra đăng nhập lại để xem biến thiên Elo:**
  * Trên Terminal 1 (Alice thắng):
    ```sh
    ./gotermoku --server localhost:4444 --user alice --pass 123456
    ```
    *Kết quả mong đợi:* Điểm Elo tăng lên: `[AUTH] Login successful! Elo: 1216` (hoặc cao hơn).
  * Trên Terminal 2 (Bob thua):
    ```sh
    ./gotermoku --server localhost:4444 --user bob --pass 123456
    ```
    *Kết quả mong đợi:* Điểm Elo giảm xuống: `[AUTH] Login successful! Elo: 1184`.
  * Bấm `Ctrl+C` hủy hàng đợi trên cả hai terminal.

---

### Kịch bản 6: Tạo và Vào Phòng Riêng bằng mã Code (`--create-room` & `--join-room`)
* **Mục tiêu:** Hai người bạn muốn chơi trực tiếp với nhau, không qua hàng đợi ngẫu nhiên.
* **Terminal 1 (Alice tạo phòng):**
  ```sh
  ./gotermoku --server localhost:4444 --user alice --pass 123456 --create-room
  ```
  *Kết quả in ra:*
  ```text
  ==================================================
   [PRIVATE ROOM CREATED] ROOM CODE: 8F2K9X
   Share this code with your friend to start playing!
  ==================================================
  [WAITING] Waiting for opponent in room 8F2K9X...
  ```
* **Terminal 2 (Bob nhập mã phòng để vào):**
  *(Thay `8F2K9X` bằng mã code thực tế in trên Terminal 1)*
  ```sh
  ./gotermoku --server localhost:4444 --user bob --pass 123456 --join-room 8F2K9X
  ```
* **Kết quả mong đợi:**
  * Hai người chơi lập tức được ghép vào ván đấu riêng.
  * Tiêu đề TUI trên cả 2 màn hình hiển thị mã phòng: `G O T E R M O K U  [ROOM: 8F2K9X]`.

---

### Kịch bản 7: Rớt mạng, Ân hạn 45 giây & Phục hồi kết nối (`--rejoin`)
* **Mục tiêu:** Kiểm tra khả năng chịu lỗi và nối lại ván cờ bị ngắt quãng.
* **Các bước thực hiện:**
  1. Trong ván cờ đang chơi ở Kịch bản 6, chú ý dòng **Match ID** đã in ra lúc bắt đầu (ví dụ: `5c6a76aec1b394de`).
  2. Cho quân Trắng đánh 1-2 nước cờ trên bàn.
  3. **Mô phỏng rớt mạng:** Tại **Terminal 2 (Bob)**, bấm tổ hợp phím `Ctrl+C` để tắt terminal.
  4. **Quan sát Terminal 1 (Alice):**
     * *Kết quả mong đợi:* Bàn cờ lập tức bị khóa, xuất hiện banner cảnh báo màu đỏ/cam đếm ngược:
       `* OPPONENT DISCONNECTED - WAITING FOR RECONNECT (45s) *`
       Số giây tự động đếm lùi: `(40s)`, `(35s)`,... Alice thử bấm phím đánh cờ nhưng không thể đặt quân vì game đang tạm dừng.
  5. **Mở lại kết nối (Bob Rejoin):**
     Tại **Terminal 2**, chạy lệnh rejoin với Match ID vừa lưu:
     ```sh
     ./gotermoku --server localhost:4444 --user bob --pass 123456 --rejoin 5c6a76aec1b394de
     ```
  6. **Kết quả mong đợi sau khi Rejoin:**
     * Terminal 2 nhận thông báo: `[REJOIN] Checking match '5c6a76aec1b394de' status... Match is active and ready for rejoin`.
     * Terminal 2 mở lại bàn cờ TUI với toàn bộ các nước đi cũ còn nguyên vẹn.
     * Banner cảnh báo trên Terminal 1 biến mất, thanh trạng thái báo `reconnected. Match resumed!`.
     * Cả hai bên tiếp tục đánh cờ bình thường.

---

### Kịch bản 8: Bảo mật Đường truyền với Mã hóa TLS
* **Mục tiêu:** Đảm bảo toàn bộ lưu lượng game và mật khẩu được mã hóa an toàn qua TLS.
* **1. Tạo chứng chỉ TLS tự ký (Self-Signed Certificate) chuẩn POSIX:**
  *(Chạy tại thư mục gốc của dự án)* [3, 4]
  ```sh
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout key.pem -out cert.pem -days 365 \
    -subj "/CN=localhost"
  ```
* **2. Khởi động Server với cờ TLS (Terminal 0):**
  Tắt server cũ (`Ctrl+C`), rồi khởi chạy lại:
  ```sh
  ./gotermoku-server --port 4444 --tls --cert cert.pem --key key.pem
  ```
  *(Server thông báo: `[SECURITY] TLS encryption enabled.`)*
* **3. Client kết nối qua kênh bảo mật TLS:**
  * Terminal 1:
    ```sh
    ./gotermoku --server localhost:4444 --tls --ca-cert cert.pem --register --user charlie --pass 123456
    ```
  * Terminal 2:
    ```sh
    ./gotermoku --server localhost:4444 --tls --ca-cert cert.pem --register --user david --pass 123456
    ```
* **Kết quả mong đợi:**
  * Hai client bắt tay TLS thành công với Server, xác thực qua file `cert.pem` và vào trận an toàn. Mọi dữ liệu truyền qua socket đều được mã hóa bằng thuật toán TLS hiện đại.

---

## TỔNG KẾT BẢNG PHÍM TẮT & ĐIỀU KHIỂN TRONG TRẬN ĐẤU

| Phím / Thao tác | Chức năng |
| :--- | :--- |
| `h`, `j`, `k`, `l` | Di chuyển con trỏ (Trái, Xuống, Lên, Phải) theo chuẩn Vim |
| `↑`, `↓`, `←`, `→` | Di chuyển con trỏ theo phím mũi tên thông thường |
| `Space` / `Enter` | Đặt quân cờ tại vị trí đang chọn |
| `Chuột trái (Click 1)` | Đặt tiêu điểm con trỏ vào ô cờ |
| `Chuột trái (Click 2)` | Đặt quân cờ vào ô cờ vừa chọn |
| `r` / `R` | Yêu cầu chơi lại (khi ván cờ đã kết thúc) |
| `Esc` / `Ctrl+C` | Thoát khỏi trận đấu / Hủy tìm trận |
