# BÀI HỌC 1: TẦNG ĐỊNH NGHĨA GIAO THỨC (`api/proto/gomoku.proto`)

### 1. Khái niệm cho người mới bắt đầu
* **gRPC là gì?** Bình thường khi viết web/API, bạn gửi chuỗi văn bản JSON qua HTTP/1.1 (`{"name": "alice"}`). Cách này dễ đọc nhưng tốn băng thông và CPU để parse chuỗi văn bản. **gRPC** (chạy trên nền **HTTP/2**) nén toàn bộ dữ liệu thành các **byte nhị phân (binary)** siêu nhỏ và truyền đi với tốc độ cực cao.
* **Protobuf (`.proto`) là gì?** Là một bản "hợp đồng" (Interface Definition Language - IDL). Bạn viết file này trước, sau đó dùng trình biên dịch `protoc` để tự động sinh ra code Go (trong thư mục `internal/common/pb/`).
* **Con số `= 1; = 2;` là gì?** Đây **KHÔNG PHẢI** là giá trị của biến! Đó là **Field Tag (Mã định danh trường)** dưới dạng nhị phân. Khi nén dữ liệu qua mạng, gRPC không gửi tên biến `username` mà chỉ gửi số `1`. Vì vậy, không bao giờ được đổi số tag của một trường đã có.

---

### 2. Mổ xẻ từng dòng code `api/proto/gomoku.proto`

#### Phần 1: Khởi tạo cú pháp & Package
```protobuf
syntax = "proto3";
package pb;
option go_package = "github.com/ttasc/gotermoku/internal/common/pb";
```
* **Dòng 1 (`syntax = "proto3";`):** Khai báo phiên bản cú pháp Protobuf thế hệ thứ 3 (phiên bản chuẩn mực hiện đại nhất).
* **Dòng 2 (`package pb;`):** Đặt tên namespace của Protobuf là `pb` để tránh xung đột với các file proto khác.
* **Dòng 3 (`option go_package = "...";`):** Hướng dẫn trình biên dịch `protoc`: khi sinh code Go, hãy đặt code đó vào thư mục `internal/common/pb` và thuộc package Go có tên là `pb`.

---

#### Phần 2: Mô hình Dữ liệu Game Bàn Cờ
```protobuf
message BoardRow {
    bytes cells = 1;
}

message Position {
    int32 x = 1;
    int32 y = 2;
}

message ClientAction {
    string type = 1;
    int32 x = 2;
    int32 y = 3;
}
```
* **Dòng `message BoardRow`:**
  * Định nghĩa một hàng của bàn cờ.
  * Trường `bytes cells = 1;`: Trong Protobuf, kiểu `bytes` ánh xạ trực tiếp $1:1$ với mảng byte `[]uint8` trong Go. Thay vì tạo một mảng số nguyên cồng kềnh, một hàng 30 ô cờ chỉ tốn đúng 30 bytes bộ nhớ.
* **Dòng `message Position`:** Lưu tọa độ $(X, Y)$ trên bàn cờ. `int32` là số nguyên có dấu 32-bit.
* **Dòng `message ClientAction`:** Gói tin từ Client gửi lên Server khi người chơi thao tác:
  * `type = 1`: Là hành động gì? (ví dụ: `"move"` là đánh cờ, `"restart"` là yêu cầu chơi lại).
  * `x = 2; y = 3;`: Tọa độ ô muốn đánh.

---

#### Phần 3: Gói tin Đồng bộ Trạng thái (`GameSync`)
```protobuf
message GameSync {
    string type = 1;
    int32 current_turn = 2;
    int32 winner = 3;
    repeated BoardRow board = 4;
    repeated Position winning_positions = 5;
    string message = 6;
    bool is_paused = 7;
    int32 grace_seconds_remaining = 8;
    int32 white_elo = 9;
    int32 black_elo = 10;
}
```
* **`message GameSync`:** Đây là gói tin Server phát sóng (broadcast) cho Client để vẽ lại toàn bộ bàn cờ.
* **`repeated BoardRow board = 4;`:** Từ khóa `repeated` tương đương với mảng/lát cắt (`slice` trong Go). Nghĩa là một danh sách nhiều `BoardRow`, tạo thành ma trận 2D của bàn cờ.
* **`repeated Position winning_positions = 5;`:** Danh sách 5 tọa độ quân cờ tạo nên đường thắng (dùng để tô sáng màu xanh lá trên TUI).
* **`bool is_paused = 7;`:** Cờ trạng thái: `true` nếu trận đấu đang bị tạm dừng do có người rớt mạng.
* **`int32 grace_seconds_remaining = 8;`:** Số giây đếm lùi của thời gian ân hạn 45 giây.
* **`int32 white_elo = 9; int32 black_elo = 10;`:** Điểm Elo hiện tại của 2 đấu thủ để hiển thị trên thanh trạng thái.

---

#### Phần 4: Xác thực & Hàng đợi Ghép trận
```protobuf
message AuthRequest {
    string username = 1;
    string password = 2;
}

message AuthResponse {
    bool success = 1;
    string message = 2;
    string token = 3;
    int32 elo = 4;
}

message QueueRequest {}

enum QueueStatus {
    QUEUE_WAITING = 0;
    QUEUE_MATCHED = 1;
    QUEUE_CANCELLED = 2;
    QUEUE_ERROR = 3;
}

message QueueResponse {
    QueueStatus status = 1;
    string message = 2;
    string match_id = 3;
    string opponent_name = 4;
    int32 assigned_color = 5;
    int32 rows = 6;
    int32 cols = 7;
    int32 opponent_elo = 8;
    int32 your_elo = 9;
    string room_code = 10;
}
```
* **`message QueueRequest {}`:** Gói tin gửi yêu cầu vào hàng đợi tìm trận. Bạn thấy nó rỗng vì `token` đã được đưa lên HTTP/2 Metadata header (bảo mật chuẩn REST/gRPC), không nhồi vào body.
* **`enum QueueStatus`:** Định nghĩa tập hợp các trạng thái tìm trận. Giá trị đầu tiên của enum trong Protobuf bắt buộc phải là `0` (trạng thái mặc định).
* **`message QueueResponse`:** Server trả về cho Client khi xếp hàng:
  * `assigned_color = 5;`: Màu quân cờ do Server gán (1: Trắng đi trước, 2: Đen đi sau).
  * `match_id = 3;`: Mã định danh duy nhất của phòng cờ.
  * `room_code = 10;`: Mã phòng riêng 6 ký tự (nếu là phòng đấu kín).

---

#### Phần 5: Phòng riêng, Phục hồi kết nối & Định nghĩa Service
```protobuf
message CreateRoomRequest {
    int32 rows = 1;
    int32 cols = 2;
}

message CreateRoomResponse {
    bool success = 1;
    string message = 2;
    string room_code = 3;
}

message JoinRoomRequest {
    string room_code = 1;
}

message RejoinRequest {
    string match_id = 1;
}

message RejoinResponse {
    bool success = 1;
    string message = 2;
    string match_id = 3;
    string opponent_name = 4;
    int32 assigned_color = 5;
    int32 rows = 6;
    int32 cols = 7;
    int32 opponent_elo = 8;
    int32 your_elo = 9;
}

message OnlineAction {
    string match_id = 1;
    ClientAction action = 2;
}

service GomokuService {
    rpc PlayStream(stream ClientAction) returns (stream GameSync);
    rpc Register(AuthRequest) returns (AuthResponse);
    rpc Login(AuthRequest) returns (AuthResponse);
    rpc QueueMatch(QueueRequest) returns (stream QueueResponse);
    rpc CreatePrivateRoom(CreateRoomRequest) returns (CreateRoomResponse);
    rpc JoinPrivateRoom(JoinRoomRequest) returns (stream QueueResponse);
    rpc CheckRejoin(RejoinRequest) returns (RejoinResponse);
    rpc PlayOnline(stream OnlineAction) returns (stream GameSync);
}
```
* **Khái niệm các loại RPC trong `service GomokuService`:**
  1. **Unary RPC (1 gửi - 1 nhận):** Ví dụ `rpc Register(AuthRequest) returns (AuthResponse);`. Client gửi 1 request, Server xử lý và trả về 1 response duy nhất (y hệt API HTTP GET/POST cổ điển).
  2. **Server-Streaming RPC (1 gửi - nhận liên tục):** Ví dụ `rpc QueueMatch(QueueRequest) returns (stream QueueResponse);`. Từ khóa `stream` ở vế sau nghĩa là Client chỉ gửi 1 yêu cầu xếp hàng, nhưng Server có thể liên tục "bắn" thông điệp về: *"Đang tìm đối thủ..."*, *"Vẫn đang tìm..."*, *"Đã ghép được trận!"*.
  3. **Bi-directional Streaming RPC (Truyền nhận 2 chiều đồng thời):** Ví dụ `rpc PlayOnline(stream OnlineAction) returns (stream GameSync);`. Từ khóa `stream` ở cả 2 vế nghĩa là giữa Client và Server mở một đường ống dữ liệu thời gian thực: Client thích gửi nước đi lúc nào thì gửi, Server thích gửi cập nhật bàn cờ lúc nào thì gửi, không bên nào phải chờ bên nào.

> **Cách sửa code:** Nếu muốn thêm một trường mới vào gói tin (ví dụ: thêm avatar URL vào `AuthResponse`), bạn chỉ cần thêm dòng `string avatar_url = 5;` vào cuối message `AuthResponse`, sau đó chạy lệnh `make proto`. **Tuyệt đối không sửa số tag của các trường cũ** để tránh làm hỏng tương thích ngược.

---

# BÀI HỌC 2: MÔ HÌNH DỮ LIỆU BÀN CỜ (`internal/common/core/state.go`)

### 1. Khái niệm Go cần biết
* **`package core`:** Trong Go, mọi file đều thuộc về một package. Các file trong cùng một package có thể sử dụng biến và hàm của nhau mà không cần import.
* **`iota`:** Là một công cụ tự động tăng số nguyên của Go. Khi bạn khai báo một khối `const (...)`, `iota` bắt đầu từ `0`, dòng tiếp theo tự động là `1`, `2`, `3`...
* **Struct:** Tương đương với `class` trong C++/Java hoặc Object trong Python, nhưng chỉ chứa dữ liệu thuần túy (không chứa code thực thi bên trong).
* **Pointer Receiver `(s *GameState)`:** Ký tự dấu hoa thị `*` biểu thị con trỏ (Pointer). Khi hàm khai báo `func (s *GameState) Reset()`, Go sẽ truyền **địa chỉ ô nhớ** của biến `GameState` vào hàm. Bất kỳ thay đổi nào bên trong hàm này sẽ tác động trực tiếp lên dữ liệu gốc. Nếu bỏ dấu `*`, Go sẽ tạo một bản sao (copy) và dữ liệu gốc sẽ không đổi!

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/common/core/state.go`

```go
package core

import "time"

const (
	Empty uint8 = iota
	White
	Black
	Draw
)
```
* **`Empty uint8 = iota`:**
  * `Empty` nhận giá trị `0` (ô cờ trống). Kiểu `uint8` là số nguyên không dấu 8-bit (chiếm đúng 1 byte, giá trị $0 \rightarrow 255$).
  * Dòng tiếp theo: `White` tự động nhận giá trị `1` (quân Trắng).
  * `Black` tự động nhận giá trị `2` (quân Đen).
  * `Draw` tự động nhận giá trị `3` (kết quả Hòa cờ).

```go
type GameState struct {
	Board            [][]uint8
	Rows             int
	Cols             int
	CurrentTurn      uint8
	MoveCount        map[uint8]int
	Winner           uint8
	WinningPositions [][2]int
	SelectedX        int
	SelectedY        int
	StartTime        time.Time
}
```
* **`type GameState struct`:** Trái tim lưu trữ toàn bộ trạng thái của một ván cờ (Single Source of Truth).
* **`Board [][]uint8`:** Lát cắt 2 chiều (Slice of Slices). `Board[y][x]` đại diện cho ô cờ tại hàng `y`, cột `x`. Giá trị của nó chỉ có thể là `Empty (0)`, `White (1)` hoặc `Black (2)`.
* **`MoveCount map[uint8]int`:** Kiểu `map` (từ điển Key-Value) lưu số nước cờ đã đánh của từng phe: `MoveCount[White]` và `MoveCount[Black]`.
* **`WinningPositions [][2]int`:** Slice chứa danh sách các tọa độ `[x, y]` tạo nên đường thắng 5 quân.
* **`SelectedX, SelectedY int`:** Tọa độ ô cờ mà con trỏ chuột/phím đang nhắm tới trên màn hình terminal.
* **`StartTime time.Time`:** Thời điểm bắt đầu ván cờ (dùng để tính đồng hồ thời gian trôi qua).

```go
func NewGameState(rows, cols int) *GameState {
	board := make([][]uint8, rows)
	for i := range board {
		board[i] = make([]uint8, cols)
	}

	return &GameState{
		Board:            board,
		Rows:             rows,
		Cols:             cols,
		CurrentTurn:      White,
		MoveCount:        map[uint8]int{White: 0, Black: 0},
		WinningPositions: make([][2]int, 0),
		SelectedX:        -1,
		SelectedY:        -1,
		StartTime:        time.Now(),
	}
}
```
* **`func NewGameState(...) *GameState`:** Đây là hàm khởi tạo (Constructor Pattern) kinh điển trong Go. Do Go không có từ khóa `class` hay `new GameState()`, ta tự viết hàm `New...` để trả về một con trỏ tới struct đã được khởi tạo sẵn sàng trong bộ nhớ Heap.
* **`board := make([][]uint8, rows)`:** Hàm `make` dùng để cấp phát động slice trong RAM. Dòng này cấp phát mảng chứa `rows` phần tử hàng.
* Vòng lặp `for i := range board`: Cấp phát từng hàng có độ dài `cols` ô cờ. Mặc định toàn bộ mảng byte này sẽ nhận giá trị `0` (tức là `Empty`).
* `CurrentTurn: White`: Theo luật Gomoku chuẩn quốc tế, quân Trắng luôn là người đi trước.
* `SelectedX: -1, SelectedY: -1`: Giá trị `-1` biểu thị con trỏ hiện chưa được đặt vào ô nào.

```go
func (s *GameState) Reset() {
	for y := 0; y < s.Rows; y++ {
		for x := 0; x < s.Cols; x++ {
			s.Board[y][x] = Empty
		}
	}
	s.CurrentTurn = White
	s.MoveCount = map[uint8]int{White: 0, Black: 0}
	s.Winner = Empty
	s.WinningPositions = make([][2]int, 0)
	s.SelectedX = -1
	s.SelectedY = -1
	s.StartTime = time.Now()
}
```
* **`func (s *GameState) Reset()`:** Tái sử dụng lại vùng nhớ của bàn cờ hiện có, đưa tất cả các ô về `Empty (0)` mà không cần cấp phát lại bộ nhớ mới (giảm tải cho Garbage Collector của Go).

```go
func (s *GameState) IsWinPos(x, y int) bool {
	if s.Winner == Empty || s.Winner == Draw {
		return false
	}
	for _, pos := range s.WinningPositions {
		if x == pos[0] && y == pos[1] {
			return true
		}
	}
	return false
}
```
* **`IsWinPos`:** Trả về `true` nếu tọa độ `(x, y)` nằm trong danh sách 5 ô cờ chiến thắng. Renderer TUI sẽ dùng hàm này để biết ô nào cần tô màu xanh lá đặc biệt.

---

# BÀI HỌC 3: THUẬT TOÁN LUẬT CHƠI GOMOKU (`internal/common/core/logic.go`)

### 1. Khái niệm cốt lõi: Thuật toán Quét tia Cục bộ $O(1)$
Nhiều người mới lập trình game thường mắc sai lầm: sau mỗi nước đi, họ dùng 2 vòng lặp lồng nhau duyệt qua toàn bộ $20 \times 30 = 600$ ô cờ để xem có ai thắng không. Việc này có độ phức tạp $O(Rows \times Cols)$.
Trong `gotermoku`, chúng ta áp dụng **Thuật toán Quét tia (Ray Casting)**: Chỉ đứng tại **đúng ô cờ vừa được đánh**, phóng tầm nhìn ra 4 hướng đối xứng với cự ly tối đa 4 bước. Độ phức tạp là $O(1)$ — chạy trong thời gian nano giây bất kể bàn cờ lớn đến đâu.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/common/core/logic.go`

```go
package core

var WinDirections = [4][2]int{
	{1, 0},  // Ngang (Horizontal)
	{0, 1},  // Dọc (Vertical)
	{1, 1},  // Đường chéo chính (Main diagonal: từ trên-trái xuống dưới-phải)
	{1, -1}, // Đường chéo phụ (Anti-diagonal: từ dưới-trái lên trên-phải)
}
```
* **`WinDirections`:** Ma trận 4 vector chỉ hướng dịch chuyển tọa độ $(\Delta x, \Delta y)$ trên mặt phẳng bàn cờ 2D.

```go
func ScanRay(board [][]uint8, cols, rows, x, y, dx, dy int, color uint8) (count int, openEnd int, pos [][2]int) {
	for i := 1; i <= 4; i++ {
		nx, ny := x+(dx*i), y+(dy*i)
		if nx < 0 || nx >= cols || ny < 0 || ny >= rows {
			break
		}
		if board[ny][nx] == color {
			count++
			pos = append(pos, [2]int{nx, ny})
		} else if board[ny][nx] == Empty {
			openEnd = 1
			break
		} else {
			break
		}
	}
	return
}
```
* **Cú pháp Go: Named Return Values `(count int, openEnd int, pos [][2]int)`:** Go cho phép đặt tên cho các giá trị trả về ngay tại chữ ký của hàm. Các biến này được tự động khởi tạo bằng giá trị zero, khi gọi lệnh `return` trần (naked return), Go sẽ tự động trả về giá trị của 3 biến này.
* **`for i := 1; i <= 4; i++`:** Một đường cờ 5 quân chỉ cần kiểm tra tối đa 4 bước về một phía tính từ ô gốc.
* **`nx, ny := x+(dx*i), y+(dy*i)`:** Tính tọa độ của ô tiếp theo theo hướng tia phóng đi.
* **Kiểm tra biên:** `if nx < 0 || nx >= cols || ny < 0 || ny >= rows`: Nếu tia đi vượt ra mép ngoài của bàn cờ $\rightarrow$ dừng tia (`break`).
* **`if board[ny][nx] == color`:** Nếu gặp quân cờ cùng màu $\rightarrow$ tăng biến đếm `count++` và ghi nhận tọa độ vào `pos`.
* **`else if board[ny][nx] == Empty`:** Nếu gặp ô trống $\rightarrow$ tia này có một đầu mở (`openEnd = 1`). Thông tin này cực kỳ quan trọng đối với thuật toán chấm điểm của AI Bot!

```go
func CanPlacePiece(state *GameState, x, y int) bool {
	if state.Winner != Empty {
		return false
	}
	if x < 0 || x >= state.Cols || y < 0 || y >= state.Rows {
		return false
	}
	return state.Board[y][x] == Empty
}
```
* **`CanPlacePiece`:** Kiểm tra nước đi có hợp lệ không:
  1. Trận đấu phải chưa kết thúc (`state.Winner == Empty`).
  2. Tọa độ phải nằm trong khung bàn cờ.
  3. Ô định đánh phải đang trống (`Board[y][x] == Empty`).

```go
func PlacePiece(state *GameState, x, y int, color uint8) {
	state.Board[y][x] = color
	state.MoveCount[color]++
	updateWinState(state, x, y)

	if state.Winner == Empty {
		// Kiểm tra hòa cờ O(1)
		if state.MoveCount[White]+state.MoveCount[Black] >= state.Rows*state.Cols {
			state.Winner = Draw
			return
		}

		if state.CurrentTurn == White {
			state.CurrentTurn = Black
		} else {
			state.CurrentTurn = White
		}
	}
}
```
* **`PlacePiece`:** Đặt quân cờ và biến đổi trạng thái:
  1. `state.Board[y][x] = color`: Gán màu cờ cho ô đó.
  2. `state.MoveCount[color]++`: Tăng tổng số nước đi của phe đó lên 1.
  3. `updateWinState(state, x, y)`: Kiểm tra xem nước cờ này có tạo thành chuỗi 5 quân không.
  4. **Kiểm tra Hòa cờ $O(1)$:** Nếu chưa ai thắng mà tổng số nước đi của cả hai phe $\ge Rows \times Cols$ (toàn bộ ô cờ đã bị đánh kín) $\rightarrow$ Gán `state.Winner = Draw` và dừng ván đấu.
  5. Nếu vẫn chưa kết thúc ván: Đổi lượt luân phiên (Trắng $\rightarrow$ Đen, Đen $\rightarrow$ Trắng).

```go
func updateWinState(state *GameState, lastX, lastY int) {
	color := state.Board[lastY][lastX]
	if color == Empty {
		return
	}

	for _, dir := range WinDirections {
		c1, _, p1 := ScanRay(state.Board, state.Cols, state.Rows, lastX, lastY, dir[0], dir[1], color)
		c2, _, p2 := ScanRay(state.Board, state.Cols, state.Rows, lastX, lastY, -dir[0], -dir[1], color)

		if c1+c2+1 >= 5 {
			state.Winner = color
			state.WinningPositions = make([][2]int, 0, c1+c2+1)
			state.WinningPositions = append(state.WinningPositions, [2]int{lastX, lastY})
			state.WinningPositions = append(state.WinningPositions, p1...)
			state.WinningPositions = append(state.WinningPositions, p2...)
			return
		}
	}
}
```
* **`c1` và `c2`:** Quét theo hướng thuận (`dir[0], dir[1]`) và hướng nghịch đảo (`-dir[0], -dir[1]`).
* **`c1 + c2 + 1 >= 5`:** Tổng số quân cùng màu ở 2 phía cộng với chính quân vừa đánh $\ge 5 \rightarrow$ Đã tìm ra người chiến thắng!
* **`state.WinningPositions`:** Thu thập chính xác tất cả các tọa độ tạo nên chuỗi thắng để gửi cho giao diện TUI vẽ hiệu ứng.

---

# BÀI HỌC 4: THUẬT TOÁN ĐIỂM ELO CHUẨN FIDE (`internal/common/core/elo.go`)

### 1. Khái niệm Toán học Elo
Hệ thống tính điểm Elo (do Giáo sư vật lý Arpad Elo phát minh) dựa trên hàm phân phối Logistic để tính toán xác suất chiến thắng kỳ vọng giữa 2 đối thủ:
* Nếu người $1600$ điểm đấu với người $1200$ điểm: Người $1600$ điểm có xác suất thắng cực cao. Nếu họ thắng, họ chỉ được cộng rất ít điểm. Nhưng nếu người $1200$ lật kèo thắng được, họ sẽ được cộng rất nhiều điểm!
* Công thức kỳ vọng xác suất của người A ($E_A$):
  $$E_A = \frac{1}{1 + 10^{(R_B - R_A)/400}}$$
* Điểm Elo mới sau trận:
  $$R'_A = R_A + K \times (S_A - E_A)$$
  *(Trong đó $K=32$ là hệ số điều chỉnh, $S_A$ là kết quả thực tế: $1.0$ khi Thắng, $0.5$ khi Hòa, $0.0$ khi Thua).*

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/common/core/elo.go`

```go
package core

import "math"

const (
	DefaultElo = 1200
	KFactor    = 32
	MinElo     = 100
)
```
* `DefaultElo = 1200`: Điểm khởi đầu mặc định của một tài khoản mới đăng ký.
* `KFactor = 32`: Hệ số co giãn tiêu chuẩn của Liên đoàn Cờ vua Quốc tế (FIDE).
* `MinElo = 100`: Điểm sàn tối thiểu (không bao giờ để người chơi bị âm điểm Elo).

```go
func CalculateElo(ratingA, ratingB int, scoreA float64) (newA, newB int) {
	expectedA := 1.0 / (1.0 + math.Pow(10.0, float64(ratingB-ratingA)/400.0))
	expectedB := 1.0 - expectedA

	scoreB := 1.0 - scoreA

	deltaA := float64(KFactor) * (scoreA - expectedA)
	deltaB := float64(KFactor) * (scoreB - expectedB)

	newA = ratingA + int(math.Round(deltaA))
	newB = ratingB + int(math.Round(deltaB))

	if newA < MinElo {
		newA = MinElo
	}
	if newB < MinElo {
		newB = MinElo
	}

	return newA, newB
}
```
* **`math.Pow(10.0, float64(ratingB-ratingA)/400.0)`:** Hàm tính lũy thừa $10^x$ của thư viện chuẩn `math`. Do hàm này yêu cầu kiểu số thực `float64`, ta phải ép kiểu `float64(...)`.
* **`expectedB := 1.0 - expectedA`:** Tổng xác suất kỳ vọng của cả hai người luôn luôn bằng $1.0$ ($100\%$).
* **`deltaA`:** Chênh lệch điểm số thực tế so với kỳ vọng ban đầu.
* **`math.Round(deltaA)`:** Làm tròn số nguyên gần nhất trước khi cộng/trừ vào điểm Elo hiện tại.
* **Chặn sàn an toàn:** `if newA < MinElo { newA = MinElo }`: Đảm bảo nếu thua liên tục thì điểm số vẫn giữ ở mức sàn $100$.

---

### TỔNG KẾT MODULE 1
Bạn vừa nắm vững:
1. File hợp đồng gRPC `gomoku.proto` và bản chất truyền nhận nhị phân HTTP/2.
2. Mô hình bộ nhớ tối giản của bàn cờ trong `state.go`.
3. Thuật toán quét tia $O(1)$ và phát hiện hòa cờ trong `logic.go`.
4. Thuật toán cân bằng trình độ Elo trong `elo.go`.

---

# BÀI HỌC 1: ĐIỂM KHỞI CHẠY SERVER (`cmd/gotermoku-server/main.go`)

### 1. Kiến trúc phân tầng tối giản
Trong các dự án Go chuẩn mực (đặc biệt theo triết lý UNIX/Suckless), thư mục `cmd/` chỉ đóng vai trò là "vỏ bọc mỏng" (thin wrapper). File `main.go` không chứa logic nghiệp vụ, mà chỉ làm nhiệm vụ ủy quyền: tiếp nhận tham số dòng lệnh từ hệ điều hành (`os.Args[1:]`) và chuyển giao quyền điều khiển cho package chức năng bên dưới (`internal/server`).

### 2. Mổ xẻ từng dòng code `cmd/gotermoku-server/main.go`

```go
package main

import (
	"os"

	"github.com/ttasc/gotermoku/internal/server"
)

func main() {
	server.Run(os.Args[1:])
}
```
* **`package main`:** Bắt buộc trong Go đối với bất kỳ chương trình nào muốn biên dịch thành một file thực thi (binary). Khi bạn gõ lệnh `./gotermoku-server`, hàm `func main()` trong `package main` là điểm đầu tiên được hệ điều hành kích hoạt.
* **`os.Args`:** Là một slice chứa toàn bộ các tham số bạn gõ từ terminal:
  * `os.Args[0]` luôn là tên file thực thi (ví dụ: `./gotermoku-server`).
  * `os.Args[1:]` (cú pháp Slice Truncate của Go) lấy toàn bộ các cờ phía sau (ví dụ: `--port 3333 --tls`).
* **`server.Run(...)`:** Gọi hàm `Run` của package `server`. Toàn bộ quá trình nạp database, dựng socket và mở gRPC server sẽ diễn ra tại đây.

---

# BÀI HỌC 2: CƠ SỞ DỮ LIỆU NHÚNG ACID `bbolt` (`internal/server/storage.go`)

### 1. Khái niệm cho người mới bắt đầu
* **Embedded Database (Cơ sở dữ liệu nhúng):** MySQL hay PostgreSQL yêu cầu bạn phải cài đặt một phần mềm máy chủ chạy ngầm, cấu hình cổng mạng và tài khoản kết nối. `bbolt` thì ngược lại: toàn bộ mã nguồn quản lý dữ liệu được biên dịch tĩnh (statically linked) **thẳng vào trong file thực thi `gotermoku-server`**. Nó lưu toàn bộ dữ liệu vào duy nhất một file trên đĩa (mặc định là `gotermoku.db`).
* **Kỹ thuật POSIX `mmap` (Memory-Mapped File):** `bbolt` không dùng các lệnh đọc/ghi file thông thường (`read`/`write`). Nó gọi system call `mmap` của hệ điều hành để ánh xạ trực tiếp file trên đĩa vào không gian bộ nhớ ảo của CPU. Nhờ đó, tốc độ đọc dữ liệu đạt mức tương đương đọc RAM trực tiếp mà không tốn công copy bộ nhớ.
* **POSIX File Lock (`flock`):** Khi `bbolt` mở file, nó yêu cầu hệ điều hành khóa file bằng `flock`. Nếu có một tiến trình khác cố tình mở cùng file này, nó sẽ bị chặn lại để chống hỏng cấu trúc dữ liệu.
* **Struct Tag (Thẻ cấu trúc `json:"..."`):** Trong Go, các thẻ đặt trong dấu huyền `` `json:"username"` `` hướng dẫn hàm `json.Marshal` biến trường đó thành tên tương ứng khi chuyển đổi struct thành chuỗi byte JSON.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/server/storage.go`

#### Khởi tạo Cấu trúc & Hằng số
```go
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/ttasc/gotermoku/internal/common/core"
)

var usersBucket = []byte("users")

type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	CreatedAt    time.Time `json:"created_at"`
	Elo          int       `json:"elo"`
	Wins         int       `json:"wins"`
	Losses       int       `json:"losses"`
	Draws        int       `json:"draws"`
}

type UserStore struct {
	db *bbolt.DB
}
```
* **`var usersBucket = []byte("users")`:** Trong `bbolt`, dữ liệu được tổ chức theo các **Bucket** (tương tự như một "Bảng" trong SQL). `bbolt` chỉ làm việc với mảng byte `[]byte`, nên ta ép kiểu chuỗi `"users"` sang byte.
* **`type User struct`:** Khuôn mẫu lưu trữ thông tin một tài khoản. Ngoài thông tin đăng nhập, nó lưu `Elo`, `Wins` (thắng), `Losses` (thua), `Draws` (hòa).
* **`type UserStore struct`:** Struct bọc lấy con trỏ `*bbolt.DB` để quản lý các thao tác truy vấn.

---

#### Hàm khởi tạo: Mở Database và Tạo Bucket
```go
func NewUserStore(filePath string) (*UserStore, error) {
	db, err := bbolt.Open(filePath, 0600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(usersBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize users bucket: %w", err)
	}

	return &UserStore{db: db}, nil
}
```
* **`bbolt.Open(filePath, 0600, ...)`:**
  * `0600`: **Quyền hạn file POSIX (File Permission)**. Số `0600` nghĩa là chỉ có người dùng (Owner) hiện tại mới có quyền Đọc và Ghi file `gotermoku.db`, những tài khoản khác trên máy tính không thể đọc trộm. Đây là tiêu chuẩn bảo mật bắt buộc của UNIX.
  * `Timeout: 2 * time.Second`: Nếu có tiến trình khác đang giữ khóa file này, chờ tối đa 2 giây rồi báo lỗi thay vì treo máy vĩnh viễn.
* **`db.Update(func(tx *bbolt.Tx) error { ... })`:**
  * Mở một **Giao dịch Ghi (Read-Write Transaction)**. `bbolt` đảm bảo tính toán tử ACID: tất cả thay đổi trong hàm này hoặc là thành công trọn vẹn, hoặc nếu có lỗi thì hoàn tác sạch sẽ (Rollback).
  * `tx.CreateBucketIfNotExists(usersBucket)`: Tự động tạo bucket `"users"` nếu file mới tạo, nếu có rồi thì bỏ qua.

```go
func (s *UserStore) Close() error {
	return s.db.Close()
}
```
* **`Close()`:** Giải phóng bộ nhớ ảo `mmap` và mở khóa POSIX `flock` khi tắt server.

---

#### Thao tác Đọc: `Get(username)`
```go
func (s *UserStore) Get(username string) (*User, bool) {
	var user *User
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(usersBucket)
		if b == nil {
			return nil
		}
		data := b.Get([]byte(username))
		if data == nil {
			return nil
		}
		var u User
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		user = &u
		return nil
	})

	if err != nil || user == nil {
		return nil, false
	}
	return user, true
}
```
* **`s.db.View(...)`:** Mở một **Giao dịch Đọc (Read-Only Transaction)**.
  * Điểm đặc biệt của `bbolt`: Hàng ngàn tiến trình có thể cùng đọc đồng thời (`concurrent read`) thông qua `db.View` với tốc độ cực đại mà không bao giờ chặn nhau (Lock-free Read).
* **`b.Get([]byte(username))`:** Tìm kiếm nhị phân theo Key trong cấu trúc cây B+ Tree. Nếu không tìm thấy, trả về `nil`.
* **`json.Unmarshal(data, &u)`:** Giải nén mảng byte JSON thành struct `User`.

---

#### Thao tác Tạo mới: `Create(user)`
```go
func (s *UserStore) Create(user *User) error {
	if user.Elo <= 0 {
		user.Elo = core.DefaultElo
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(usersBucket)
		k := []byte(user.Username)
		if b.Get(k) != nil {
			return errors.New("username already exists")
		}

		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return b.Put(k, data)
	})
}
```
* **Chống trùng tài khoản:** Kiểm tra `if b.Get(k) != nil` ngay bên trong giao dịch `db.Update`. Do giao dịch ghi được khóa tuần tự, điều này triệt tiêu hoàn toàn lỗi Race Condition (hai người cùng đăng ký 1 tên cùng một mili-giây).
* **`b.Put(k, data)`:** Ghi cặp Key-Value (`[username] -> [dữ liệu JSON của user]`) xuống đĩa.

---

#### Cập nhật Thống kê Nguyên tử: `UpdateStats`
```go
func (s *UserStore) UpdateStats(username string, newElo int, won, lost, drew bool) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(usersBucket)
		k := []byte(username)
		data := b.Get(k)
		if data == nil {
			return errors.New("user not found")
		}

		var u User
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}

		u.Elo = newElo
		if won {
			u.Wins++
		} else if lost {
			u.Losses++
		} else if drew {
			u.Draws++
		}

		updatedData, err := json.Marshal(&u)
		if err != nil {
			return err
		}
		return b.Put(k, updatedData)
	})
}
```
* **Bản chất nguyên tử:** Toàn bộ chu trình: Đọc dữ liệu cũ $\rightarrow$ Đổi điểm Elo mới $\rightarrow$ Tăng số trận thắng/thua/hòa $\rightarrow$ Ghi đè lại dữ liệu được đóng gói trong **duy nhất 1 transaction `db.Update`**. Không bao giờ xảy ra tình trạng đang tính điểm thì bị xung đột luồng.

---

# BÀI HỌC 3: XÁC THỰC, MẬT MÃ HỌC & RATE LIMITING (`internal/server/auth.go`)

### 1. Khái niệm Mật mã học quan trọng
* **Salt (Muối) là gì?** Nếu Alice và Bob cùng đặt mật khẩu `"123456"`, khi băm bằng SHA-256 thông thường, mã hash của cả 2 sẽ giống hệt nhau. Kẻ tấn công có thể dùng bảng tra cứu sẵn (**Rainbow Table**) để dịch ngược mật khẩu ngay lập tức. Bằng cách sinh ra 16 bytes ngẫu nhiên làm "muối" gộp vào mật khẩu trước khi băm, chuỗi hash sinh ra sẽ hoàn toàn khác nhau.
* **Key Stretching (50.000 vòng SHA-256):** Máy tính hiện đại có thể thử hàng trăm triệu phép tính SHA-256 mỗi giây để mò mật khẩu (Brute-force). Bằng cách bắt máy tính phải lặp lại phép băm 50.000 lần, ta làm chậm tốc độ mò mật khẩu của hacker xuống hàng chục ngàn lần, trong khi người dùng bình thường đăng nhập chỉ tốn khoảng 10 miligiây.
* **Timing Attack & `subtle.ConstantTimeCompare`:**
  * Thông thường, toán tử so sánh chuỗi `str1 == str2` trong các ngôn ngữ lập trình sẽ so sánh từng ký tự từ trái qua phải. Nếu ký tự đầu tiên sai, nó dừng lại ngay lập tức.
  * Hacker có thể đo thời gian phản hồi của server bằng đơn vị nano giây để đoán xem ký tự nào đúng!
  * Hàm `subtle.ConstantTimeCompare` luôn luôn so sánh toàn bộ độ dài của chuỗi với thời gian cố định, miễn nhiễm hoàn toàn với tấn công rò rỉ thời gian.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/server/auth.go`

#### Phần 1: Bộ lọc Chống Spam (Rate Limiter theo IP)
```go
package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
)

const (
	hashIterations = 50000
	tokenBytesLen  = 32
	sessionTTL     = 24 * time.Hour
)

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	window   time.Duration
	maxHits  int
}
```
* **`sync.Mutex`:** Khóa loại trừ tương hỗ (Mutual Exclusion). Dùng để ngăn hai Goroutine sửa đổi cùng lúc vào map `attempts` (vốn sẽ gây panic `concurrent map writes` trong Go).
* **`attempts map[string][]time.Time`:** Lưu lịch sử các mốc thời gian gọi request của từng IP.

```go
func NewRateLimiter(window time.Duration, maxHits int) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		maxHits:  maxHits,
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}
```
* **`go func() { ... }()`:** Cú pháp chạy một **Goroutine** ngầm (tiến trình cực nhẹ của Go, chỉ tốn ~2KB RAM).
* **`ticker := time.NewTicker(5 * time.Minute)`:** Cứ mỗi 5 phút, goroutine này thức dậy gọi hàm `rl.cleanup()` để dọn sạch các IP cũ ra khỏi RAM, chống rò rỉ bộ nhớ.

```go
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-rl.window)

	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if t.After(threshold) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.maxHits {
		rl.attempts[ip] = valid
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}
```
* **`defer rl.mu.Unlock()`:** Từ khóa `defer` trong Go đảm bảo lệnh mở khóa Mutex chắc chắn được gọi khi hàm kết thúc, kể cả khi hàm bị lỗi hoặc return giữa chừng.
* **Thuật toán Cửa sổ trượt (Sliding Window):** Loại bỏ các mốc thời gian đã nằm ngoài khoảng `window` (ví dụ 10 giây). Nếu số lần gọi còn lại $\ge maxHits$ (ví dụ 5 lần) $\rightarrow$ Từ chối (`return false`).

---

#### Phần 2: Quản lý Phiên Đăng Nhập (Session Store)
```go
type Session struct {
	Username  string
	ExpiresAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]Session),
	}
}

func (ss *SessionStore) Create(username string) (string, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	buf := make([]byte, tokenBytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	ss.sessions[token] = Session{
		Username:  username,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	return token, nil
}
```
* **`crypto/rand.Read(buf)`:** Sử dụng hàm sinh ngẫu nhiên cấp độ mật mã học của nhân hệ điều hành (`/dev/urandom` trên POSIX Linux/macOS) để sinh ra chuỗi 32 bytes ngẫu nhiên.
* **`hex.EncodeToString(buf)`:** Chuyển 32 bytes nhị phân thành chuỗi Hex dài 64 ký tự an toàn để làm Token phiên.
* **`sessionTTL = 24 * time.Hour`:** Token có hạn dùng trong 24 giờ.

```go
func (ss *SessionStore) Validate(token string) (string, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	sess, ok := ss.sessions[token]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return "", false
	}
	return sess.Username, true
}
```
* **`sync.RWMutex`:** Khóa cho phép **nhiều luồng cùng đọc** (`RLock`), nhưng **chỉ một luồng được ghi** (`Lock`). Tối ưu hiệu năng cực cao khi kiểm tra token cho hàng loạt request gửi lên cùng lúc.

---

#### Phần 3: Đăng ký & Đăng nhập (`AuthManager`)
```go
type AuthManager struct {
	store   *UserStore
	limiter *RateLimiter
	session *SessionStore
}

func NewAuthManager(store *UserStore) *AuthManager {
	return &AuthManager{
		store:   store,
		limiter: NewRateLimiter(10*time.Second, 5), // Tối đa 5 request / 10s trên 1 IP
		session: NewSessionStore(),
	}
}

func (a *AuthManager) Register(ip, username, password string) (string, error) {
	if !a.limiter.Allow(ip) {
		return "", errors.New("too many requests, please slow down")
	}

	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 20 {
		return "", errors.New("username must be between 3 and 20 characters")
	}
	if len(password) < 6 {
		return "", errors.New("password must be at least 6 characters")
	}

	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes)
	hash := computeHash(password, salt)

	user := &User{
		Username:     username,
		PasswordHash: hash,
		Salt:         salt,
		CreatedAt:    time.Now(),
		Elo:          core.DefaultElo,
		Wins:         0,
		Losses:       0,
		Draws:        0,
	}

	if err := a.store.Create(user); err != nil {
		return "", err
	}

	return a.session.Create(username)
}
```
* **Kiểm tra đầu vào:** Tên tài khoản từ $3 \rightarrow 20$ ký tự, mật khẩu tối thiểu 6 ký tự.
* **Tạo Salt:** Sinh 16 bytes ngẫu nhiên bằng `crypto/rand`.
* **Lưu trữ:** Gán mặc định điểm Elo ban đầu là `1200`, lưu vào `bbolt` qua `a.store.Create(user)`, và cấp ngay token phiên đăng nhập.

```go
func (a *AuthManager) Login(ip, username, password string) (string, error) {
	if !a.limiter.Allow(ip) {
		return "", errors.New("too many requests, please slow down")
	}

	user, ok := a.store.Get(username)
	if !ok {
		return "", errors.New("invalid username or password")
	}

	expectedHash := computeHash(password, user.Salt)
	if subtle.ConstantTimeCompare([]byte(expectedHash), []byte(user.PasswordHash)) != 1 {
		return "", errors.New("invalid username or password")
	}

	return a.session.Create(username)
}
```
* **Đăng nhập an toàn:** Lấy muối `user.Salt` đã lưu trong database của người đó, băm lại mật khẩu vừa nhập bằng `computeHash`.
* **`subtle.ConstantTimeCompare`:** So sánh hash an toàn, chống lộ mật khẩu qua Timing Attack.

```go
func computeHash(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(password))
	digest := h.Sum(nil)

	for i := 0; i < hashIterations; i++ {
		h.Reset()
		h.Write(digest)
		digest = h.Sum(nil)
	}

	return hex.EncodeToString(digest)
}
```
* **Hàm băm 50.000 vòng:**
  1. Gộp `salt` và `password`, băm lần thứ nhất để ra mảng `digest`.
  2. Dùng vòng lặp `for i := 0; i < 50000; i++` liên tục băm lại kết quả của lần trước.
  3. Trả về chuỗi Hex kết quả cuối cùng.

---

# BÀI HỌC 1: INTERCEPTOR & XÁC THỰC TẬP TRUNG (`internal/server/interceptor.go`)

### 1. Khái niệm cho người mới bắt đầu
* **Interceptor (Middleware) là gì?** Thay vì ở mỗi hàm xử lý game bạn đều phải viết lại 5–10 dòng code: lấy token $\rightarrow$ kiểm tra token $\rightarrow$ nếu sai thì ngắt kết nối... ta đặt một người "gác cổng" ở ngoài cùng. Mọi request muốn vào server đều phải đi qua Interceptor này. Nếu hợp lệ, nó sẽ gắn thông tin người dùng vào `Context` rồi cho đi tiếp.
* **`context.Context` là gì?** Là một đối tượng đặc thù của Go, đi kèm với mỗi request từ lúc sinh ra đến lúc kết thúc. Nó dùng để mang theo các thông tin định danh (Metadata) và truyền tín hiệu hủy (Cancel/Timeout) xuống các hàm con.
* **Tại sao phải dùng `type contextKey string`?** Go khuyên bạn **không bao giờ** dùng chuỗi thuần túy (`"user"`) làm Key trong Context, vì package khác cũng có thể đặt trùng key `"user"`. Bằng cách tự định nghĩa một kiểu dữ liệu riêng (`type contextKey string`), Go đảm bảo tính duy nhất, tránh bị ghi đè dữ liệu ngoài ý muốn.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/server/interceptor.go`

#### Khởi tạo Context Key & Helper trích xuất
```go
package server

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	userContextKey  contextKey = "auth_user"
	tokenContextKey contextKey = "auth_token"
)

func UserFromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(userContextKey).(string)
	return u, ok
}

func TokenFromContext(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(tokenContextKey).(string)
	return t, ok
}
```
* **`type contextKey string`:** Định nghĩa kiểu Key riêng cho Context.
* **`UserFromContext` & `TokenFromContext`:** Các hàm tiện ích (Helper). Cú pháp `.(string)` trong Go gọi là **Type Assertion (Ép kiểu an toàn)**: nếu giá trị lấy ra đúng là kiểu chuỗi thì `ok = true`, nếu không có hoặc sai kiểu thì `ok = false` (không bao giờ gây crash chương trình).

---

#### Danh sách Trắng (Whitelist) & Bóc tách Token từ HTTP/2 Headers
```go
func isPublicMethod(fullMethod string) bool {
	switch fullMethod {
	case "/pb.GomokuService/Register",
		"/pb.GomokuService/Login",
		"/pb.GomokuService/PlayStream": // Dành riêng cho LAN P2P
		return true
	default:
		return false
	}
}

func extractAndValidateToken(ctx context.Context, auth *AuthManager) (string, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "missing metadata headers")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	rawToken := authHeaders[0]
	token := strings.TrimPrefix(rawToken, "Bearer ")
	token = strings.TrimSpace(token)

	username, valid := auth.ValidateToken(token)
	if !valid {
		return "", "", status.Error(codes.Unauthenticated, "invalid or expired session token")
	}

	return username, token, nil
}
```
* **`isPublicMethod`:** Whitelist các RPC không cần token. Đăng ký và Đăng nhập rõ ràng chưa có token; còn `PlayStream` là chế độ P2P nội bộ LAN nên không cần xác thực tài khoản.
* **`metadata.FromIncomingContext(ctx)`:** Bóc tách các header HTTP/2 mà Client gửi kèm trong request.
* **`md.Get("authorization")`:** Đọc header mang tên `authorization`.
* **`strings.TrimPrefix(rawToken, "Bearer ")`:** Chuẩn OAuth2/JWT thường gửi header dạng `Bearer <chuỗi_token>`. Hàm này cắt bỏ tiền tố `"Bearer "` để lấy token thô.
* **`auth.ValidateToken(token)`:** Kiểm tra token trong `SessionStore` (đã học ở Module 2). Nếu sai hoặc hết hạn 24h $\rightarrow$ trả về lỗi chuẩn `codes.Unauthenticated` của gRPC.

---

#### Unary Interceptor (Dành cho RPC 1-Gửi 1-Nhận)
```go
func UnaryAuthInterceptor(auth *AuthManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		username, token, err := extractAndValidateToken(ctx, auth)
		if err != nil {
			return nil, err
		}

		newCtx := context.WithValue(ctx, userContextKey, username)
		newCtx = context.WithValue(newCtx, tokenContextKey, token)
		return handler(newCtx, req)
	}
}
```
* **Cơ chế hoạt động:**
  1. Kiểm tra nếu method nằm trong Whitelist $\rightarrow$ Cho đi tiếp ngay (`handler(ctx, req)`).
  2. Bóc tách và xác thực token bằng `extractAndValidateToken`.
  3. **`context.WithValue(ctx, key, val)`:** Tạo một Context mới, nhét thêm biến `username` và `token` vào bên trong.
  4. Chuyển `newCtx` này cho `handler` tiếp theo. Các hàm phía sau (như `CreatePrivateRoom`) chỉ cần gọi `UserFromContext(ctx)` là biết ngay ai đang gọi lệnh.

---

#### Stream Interceptor (Dành cho RPC Truyền nhận Liên tục)
```go
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

func StreamAuthInterceptor(auth *AuthManager) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		username, token, err := extractAndValidateToken(ss.Context(), auth)
		if err != nil {
			return err
		}

		newCtx := context.WithValue(ss.Context(), userContextKey, username)
		newCtx = context.WithValue(newCtx, tokenContextKey, token)
		wrapped := &wrappedServerStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}
```
* **Kỹ thuật Interface Wrapping (Decorator Pattern):** Trong gRPC Stream, `grpc.ServerStream` là một Interface có sẵn hàm `Context()`. Nhưng interface gốc không cho phép bạn đổi Context.
* Ta tạo struct `wrappedServerStream` "bọc" lấy stream cũ, và ghi đè hàm `Context()` để trả về `newCtx` đã chứa thông tin xác thực. Nhờ đó, luồng stream `PlayOnline` có thể biết danh tính người chơi trong suốt cả ván đấu.

---

# BÀI HỌC 2: HẠ TẦNG SERVER, KEEPALIVE & TLS IN-MEMORY (`internal/server/server.go`)

### 1. Khái niệm nền tảng
* **Tại sao dùng ECDSA P-256 thay vì RSA?** Khóa RSA 2048-bit cổ điển đòi hỏi thuật toán tìm số nguyên tố lớn rất tốn CPU (mất khoảng $50 - 200\text{ms}$). Thuật toán đường cong elip **ECDSA P-256** sinh khóa cực nhanh ($< 2\text{ms}$), kích thước khóa nhỏ hơn $8$ lần nhưng có độ bảo mật tương đương, cực kỳ phù hợp với triết lý tối giản của Suckless.
* **Keepalive Ping là gì?** Khi chơi game qua Internet, nếu một bên bị mất mạng đột ngột (rút dây mạng, mất 4G), hệ điều hành không nhận được gói tin đóng kết nối (FIN/RST), socket sẽ rơi vào trạng thái "chết lâm sàng" (Half-open). gRPC Keepalive tự động gửi khung tin HTTP/2 PING định kỳ. Nếu quá thời gian quy định mà không có phản hồi PONG, Server lập tức ngắt socket để giải phóng tài nguyên.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/server/server.go`

#### Thuật toán Sinh Chứng chỉ Tự ký Trong RAM
```go
func generateEphemeralCertificate() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate ecdsa key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate serial: %w", err)
	}
```
* **`ecdsa.GenerateKey(elliptic.P256(), rand.Reader)`:** Sinh một cặp khóa bí mật và công khai (Private/Public Key) trên đường cong chuẩn P-256 bằng nguồn ngẫu nhiên an toàn của OS.
* **`serialNumber`:** Mỗi chứng chỉ X.509 chuẩn quốc tế bắt buộc phải có một số sê-ri duy nhất 128-bit. Dòng này sinh ngẫu nhiên số sê-ri đó.

```go
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Gotermoku In-Memory Server"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}
```
* **`template := x509.Certificate`:** Bản thiết kế chứng chỉ số:
  * `NotBefore` & `NotAfter`: Chứng chỉ có hiệu lực từ cách đây 1 giờ đến 1 năm sau (lùi lại 1 giờ để phòng trường hợp đồng hồ giữa Client và Server bị lệch nhau vài phút).
  * `ExtKeyUsage: x509.ExtKeyUsageServerAuth`: Xác định chứng chỉ này dùng để làm máy chủ SSL/TLS.
  * `DNSNames` & `IPAddresses`: Chứng chỉ hợp lệ cho tên miền `localhost`, địa chỉ IPv4 `127.0.0.1` và IPv6 `::1`.
* **`x509.CreateCertificate(...)`:** Ký số chứng chỉ bằng chính private key vừa tạo (Self-signed), xuất ra định dạng nhị phân DER.
* **`tls.Certificate`:** Đóng gói chứng chỉ nhị phân và private key thành đối tượng hoàn chỉnh sẵn sàng cho gRPC nạp vào RAM mà **không cần ghi một byte nào xuống đĩa cứng**.

---

#### Hàm Điều phối Vòng đời: `server.Run(args)`
```go
func Run(args []string) {
	fs := flag.NewFlagSet("gotermoku-server", flag.ExitOnError)
	port := fs.String("port", "3333", "TCP port for central server")
	dbPath := fs.String("db", "gotermoku.db", "Path to bbolt database file")
	rows := fs.Int("rows", 20, "Default board rows for online matches")
	cols := fs.Int("cols", 30, "Default board cols for online matches")

	enableTLS := fs.Bool("tls", false, "Enable TLS encryption")
	certFile := fs.String("cert", "", "Path to TLS certificate file (optional)")
	keyFile := fs.String("key", "", "Path to TLS private key file (optional)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing server flags: %v\n", err)
		os.Exit(1)
	}
```
* **`flag.NewFlagSet`:** Phân tích các cờ dòng lệnh độc lập cho Server. Giá trị mặc định của `port` được đồng bộ là `"3333"`.

```go
	userStore, err := NewUserStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}

	authMgr := NewAuthManager(userStore)
	roomMgr := NewRoomManager()
	matchmaker := NewMatchmaker(roomMgr, authMgr, *rows, *cols)
	centralSrv := NewCentralServer(authMgr, matchmaker, roomMgr)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error binding port %s: %v\n", *port, err)
		_ = userStore.Close()
		os.Exit(1)
	}
```
* **Mô hình Khởi tạo Ráp nối (Dependency Injection):** `userStore` $\rightarrow$ `authMgr` $\rightarrow$ `roomMgr` $\rightarrow$ `matchmaker` $\rightarrow$ `centralSrv`. Các thành phần được truyền rõ ràng vào nhau qua hàm khởi tạo, không dùng biến toàn cục (Global state) gây khó kiểm soát.
* **`net.Listen("tcp", ":"+*port)`:** Lời gọi hệ thống mở cổng socket TCP trên card mạng.

---

#### Cấu hình gRPC Keepalive & Nạp TLS
```go
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              10 * time.Second,
		Timeout:           3 * time.Second,
	}
	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             1 * time.Second,
		PermitWithoutStream: true,
	}

	serverOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(kaPolicy),
		grpc.UnaryInterceptor(UnaryAuthInterceptor(authMgr)),
		grpc.StreamInterceptor(StreamAuthInterceptor(authMgr)),
	}
```
* **`Time: 10s, Timeout: 3s`:** Sau 10 giây không có nước đi nào, Server tự bắn 1 gói tin PING xuống Client. Nếu sau 3 giây Client không trả lời $\rightarrow$ Server coi như Client đã rớt mạng và ngắt stream.
* **`MinTime: 1s, PermitWithoutStream: true`:** (Như đã giải thích trước đó) Cho phép Client ping cách nhau tối thiểu 1s để không bị kích nhầm trong các bài kiểm tra hoặc khi mạng LAN có độ trễ cực thấp.
* **`serverOpts`:** Đăng ký các Middleware Interceptor ta vừa viết ở Bài học 1 vào Server.

```go
	if *enableTLS {
		if *certFile != "" && *keyFile != "" {
			creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
			// ...
			serverOpts = append(serverOpts, grpc.Creds(creds))
		} else {
			tlsCert, err := generateEphemeralCertificate()
			// ...
			serverOpts = append(serverOpts, grpc.Creds(credentials.NewServerTLSFromCert(&tlsCert)))
		}
	} else {
		fmt.Println("[SECURITY] Running in plaintext mode (insecure).")
	}

	grpcServer := grpc.NewServer(serverOpts...)
	pb.RegisterGomokuServiceServer(grpcServer, centralSrv)
```
* **Phân nhánh TLS linh hoạt:** Có file thì nạp file (Let's Encrypt), không có file thì tự tạo cert ECDSA trong RAM, đảm bảo Server luôn sẵn sàng mã hóa an toàn mà không đòi hỏi thao tác cấu hình từ người dùng.

---

#### Bắt Tín hiệu POSIX (Graceful Shutdown)
```go
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n[SERVER] Received signal %v. Gracefully shutting down...\n", sig)
		grpcServer.GracefulStop()
		_ = userStore.Close()
		fmt.Println("[SERVER] Database and server closed cleanly.")
		os.Exit(0)
	}()

	fmt.Printf("[SERVER] Listening on port %s (board: %dx%d, db: %s)\n", *port, *rows, *cols, *dbPath)
	if err := grpcServer.Serve(listener); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		_ = userStore.Close()
		os.Exit(1)
	}
}
```
* **`signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)`:** Lắng nghe 2 tín hiệu kết thúc chuẩn của hệ điều hành POSIX:
  * `os.Interrupt`: Tín hiệu `SIGINT` (khi người quản trị nhấn `Ctrl+C`).
  * `syscall.SIGTERM`: Tín hiệu yêu cầu dừng tiến trình từ hệ điều hành (lệnh `kill` hoặc khi dịch vụ `systemd`/Docker dừng tiến trình).
* **`sig := <-sigChan`:** Lệnh nhận từ channel này sẽ **đứng chờ (block)** cho đến khi có tín hiệu bấm `Ctrl+C`.
* **`grpcServer.GracefulStop()`:** Không ngắt kết nối đột ngột mà cho phép các request dở dang hoàn tất, thông báo cho các Client đóng stream êm đẹp.
* **`userStore.Close()`:** Đóng cơ sở dữ liệu `bbolt`, xả hết cache xuống đĩa và mở khóa file `flock` chuẩn POSIX trước khi tiến trình chính thức thoát (`os.Exit(0)`).

---

# BÀI HỌC 1: STATE MACHINE PHÒNG CỜ & XỬ THUA BỎ CUỘC (`internal/server/room.go`)

### 1. Khái niệm kỹ thuật cho người mới bắt đầu
* **Non-blocking Channel Send:** Nếu bạn viết `player.SendChan <- msg`, nếu mạng của người chơi đó bị lag và buffer bị đầy, câu lệnh này sẽ **treo vô hạn (block)**! Và vì câu lệnh này đang nằm trong hàm giữ khóa `r.mu.Lock()`, toàn bộ phòng đấu và server sẽ bị **Deadlock** (treo cứng). Để giải quyết triệt để, ta dùng cấu trúc:
  ```go
  select {
  case player.SendChan <- msg:
  default: // Nếu buffer đầy, bỏ qua ngay lập tức, không được phép treo server!
  }
  ```
* **Kênh Hủy (Cancellation Channel `chan struct{}`):** Để yêu cầu một Goroutine đếm lùi đang chạy ngầm dừng lại (khi đối thủ kết nối lại kịp thời), cách chuẩn mực nhất của Go là đóng một channel rỗng: `close(r.graceStopChan)`. Tất cả các lệnh `select` đang lắng nghe channel này sẽ lập tức thoát vòng lặp.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/server/room.go`

#### Khởi tạo Cấu trúc Phòng & Người chơi
```go
package server

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/gotermoku/internal/common/pb"
)

type RoomStatus int

const (
	RoomActive RoomStatus = iota // 0: Đang thi đấu bình thường
	RoomPaused                   // 1: Tạm dừng (đối thủ rớt mạng, đang đếm lùi ân hạn)
	RoomClosed                   // 2: Đã kết thúc hoặc đã hủy
)

const defaultGraceSeconds = 45

type RoomPlayer struct {
	Username  string
	Token     string
	Color     uint8
	Elo       int
	Connected bool
	SendChan  chan *pb.GameSync
}
```
* **`RoomStatus`:** Enum quản lý trạng thái phòng bằng `iota`.
* **`RoomPlayer`:** Đại diện cho 1 người trong phòng:
  * `Connected`: Trạng thái kết nối socket mạng (`true`/`false`).
  * `SendChan chan *pb.GameSync`: Kênh đệm 10 phần tử để đẩy các gói tin bàn cờ xuống goroutine mạng của người chơi đó.

```go
type Room struct {
	ID             string
	mu             sync.Mutex
	State          *core.GameState
	Status         RoomStatus
	White          *RoomPlayer
	Black          *RoomPlayer
	onRemove       func(roomID string)
	authMgr        *AuthManager
	eloUpdated     bool
	graceStopChan  chan struct{}
	graceRemaining int
}
```
* **`State *core.GameState`:** Server nắm giữ bản gốc duy nhất của bàn cờ (Single Source of Truth), Client chỉ là màn hình hiển thị.
* **`onRemove func(roomID string)`:** Hàm Callback. Khi phòng kết thúc, phòng tự gọi hàm này để báo cho `RoomManager` xóa mình ra khỏi RAM.
* **`eloUpdated bool`:** Cờ đánh dấu để đảm bảo điểm Elo chỉ được cộng/trừ duy nhất 1 lần khi ván cờ kết thúc, không bị tính trùng lặp.

---

#### Hàm Xử lý Nước đi & Kiểm duyệt Luật (`HandleAction`)
```go
func (r *Room) HandleAction(token string, action *pb.ClientAction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status != RoomActive {
		return errors.New("game is currently paused or closed")
	}

	player := r.getPlayerByToken(token)
	if player == nil {
		return errors.New("unauthorized action: player not in this room")
	}

	switch action.Type {
	case "move":
		if r.State.CurrentTurn != player.Color {
			return errors.New("not your turn")
		}
		x, y := int(action.X), int(action.Y)
		if !core.CanPlacePiece(r.State, x, y) {
			return errors.New("invalid move coordinate")
		}
		core.PlacePiece(r.State, x, y, player.Color)

		if r.State.Winner != core.Empty {
			r.updateEloOnGameEndLocked()
		}

		r.broadcastLocked("sync", "")
```
* **Server-Authoritative:** Client gửi tọa độ `(X, Y)` lên, Server không tin tưởng Client mù quáng mà kiểm tra từng bước:
  1. Phòng có đang bị tạm dừng không? (`r.Status != RoomActive`).
  2. Người gửi có đúng là người trong phòng này không?
  3. Có đúng lượt đi của người này không? (`r.State.CurrentTurn != player.Color`).
  4. Ô đó có hợp lệ và đang trống không? (`core.CanPlacePiece`).
* **Cập nhật Elo tức thì:** Nếu nước cờ tạo ra chiến thắng (`Winner != Empty`), gọi `updateEloOnGameEndLocked()` để tính lại điểm Elo FIDE ngay lập tức.
* **`r.broadcastLocked("sync", "")`:** Phát sóng bàn cờ mới cho cả hai bên cùng thấy.

---

#### Cơ chế Xử lý Rớt mạng & Đếm lùi Ân hạn 45 giây
```go
func (r *Room) HandleDisconnect(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status == RoomClosed {
		return
	}

	leaver := r.getPlayerByToken(token)
	if leaver == nil {
		return
	}
	leaver.Connected = false

	if r.State.Winner != core.Empty {
		r.closeRoomLocked()
		return
	}

	other := r.getOtherPlayer(token)
	if other == nil || !other.Connected {
		r.closeRoomLocked()
		return
	}

	r.Status = RoomPaused
	r.graceRemaining = defaultGraceSeconds
	r.graceStopChan = make(chan struct{})

	msg := fmt.Sprintf("%s disconnected. Waiting for reconnect (%ds)...", leaver.Username, r.graceRemaining)
	r.broadcastLocked("pause", msg)

	go r.runGracePeriodCountdown(r.graceStopChan, token)
}
```
* **Bảo vệ ván đấu:** Khi 1 người ngắt kết nối:
  1. Nếu ván cờ đã xong rồi $\rightarrow$ Đóng phòng (`closeRoomLocked`).
  2. Nếu cả 2 đều đã ngắt kết nối $\rightarrow$ Đóng phòng.
  3. Nếu đối thủ vẫn còn trong phòng: Chuyển sang `RoomPaused`, tạo `graceStopChan` và phát sóng gói tin `pause` kèm số giây đếm lùi `45s`.
  4. Bật một Goroutine chạy ngầm `runGracePeriodCountdown` để đếm lùi từng giây.

```go
func (r *Room) runGracePeriodCountdown(stopChan chan struct{}, leaverToken string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return // Đối thủ đã kết nối lại -> Thoát goroutine đếm lùi ngay lập tức!
		case <-ticker.C:
			r.mu.Lock()
			if r.Status != RoomPaused {
				r.mu.Unlock()
				return
			}

			r.graceRemaining--

			if r.graceRemaining <= 0 {
				// HẾT 45 GIÂY -> XỬ THUA DO BỎ CUỘC (FORFEIT)
				other := r.getOtherPlayer(leaverToken)
				leaver := r.getPlayerByToken(leaverToken)

				if other != nil && leaver != nil {
					r.State.Winner = other.Color // Người ở lại chiến thắng!
					r.updateEloOnGameEndLocked() // Trừ điểm người bỏ cuộc, cộng điểm người ở lại
					forfeitMsg := fmt.Sprintf("%s forfeited due to disconnection timeout. %s wins!", leaver.Username, other.Username)
					r.broadcastLocked("disconnect", forfeitMsg)
				}

				r.closeRoomLocked()
				r.mu.Unlock()
				return
			}

			if r.graceRemaining%5 == 0 || r.graceRemaining <= 5 {
				msg := fmt.Sprintf("Waiting for reconnect (%ds remaining)...", r.graceRemaining)
				r.broadcastLocked("pause", msg)
			}
			r.mu.Unlock()
		}
	}
}
```
* **Xử thua bỏ cuộc (Forfeit):** Nếu hết 45 giây mà `stopChan` không được kích hoạt:
  * Gán người chiến thắng là người ở lại: `r.State.Winner = other.Color`.
  * Tính toán và trừ điểm Elo của người bỏ cuộc trong cơ sở dữ liệu `bbolt`.
  * Đóng phòng đấu an toàn.

---

#### Khôi phục Kết nối Ván cờ (`Rejoin`)
```go
func (r *Room) Rejoin(token string, newSendChan chan *pb.GameSync) (*RoomPlayer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status == RoomClosed {
		return nil, errors.New("match has already ended or closed")
	}

	player := r.getPlayerByToken(token)
	if player == nil {
		return nil, errors.New("unauthorized: player not in this match")
	}

	if r.Status == RoomPaused {
		if r.graceStopChan != nil {
			close(r.graceStopChan) // Phát tín hiệu dập tắt Goroutine đếm lùi 45s
			r.graceStopChan = nil
		}
		r.Status = RoomActive
	}

	player.Connected = true
	player.SendChan = newSendChan // Nối lại đường ống mạng mới của Client

	resumeMsg := fmt.Sprintf("%s reconnected. Match resumed!", player.Username)
	r.broadcastLocked("resume", resumeMsg)

	return player, nil
}
```
* **Đấu nối lại đường ống mạng:** Khi người chơi mở lại terminal với cờ `--rejoin`:
  * Đóng kênh `graceStopChan` để hủy đếm lùi xử thua.
  * Đưa phòng về `RoomActive`.
  * Gán kênh gửi tin mới `player.SendChan = newSendChan`.
  * Bắn gói tin `resume` mang đầy đủ trạng thái bàn cờ hiện tại để 2 bên tiếp tục thi đấu.

---

# BÀI HỌC 2: HÀNG ĐỢI ELO & PHÒNG RIÊNG (`internal/server/matchmaker.go`)

### 1. Khái niệm Thuật toán Ghép trận
* **Vấn đề nghẽn hàng đợi (Matchmaking Starvation):** Nếu bạn quy định cứng: *"Chỉ ghép 2 người có chênh lệch $\le 50$ điểm Elo"*, thì vào ban đêm khi ít người chơi, người $1300$ điểm sẽ phải chờ mãi mãi nếu người duy nhất đang online có $1400$ điểm.
* **Giải pháp Dynamic Elo Window:** Khởi điểm dải điểm là $\pm 100$. Cứ sau mỗi $5$ giây chờ đợi, dải điểm tự động nới rộng thêm $+50$:
  $$\Delta Elo = 100 + 50 \times \lfloor \text{Thời gian chờ (giây)} / 5 \rfloor$$
  Nhờ đó, ban đầu hệ thống luôn ưu tiên người cùng trình độ, nhưng nếu phải chờ lâu, dải điểm sẽ mở rộng để đảm bảo người chơi luôn tìm được trận đấu.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/server/matchmaker.go`

#### Đăng ký Hàng đợi & Vòng lặp Ghép trận
```go
func (m *Matchmaker) Enqueue(username, token string) (<-chan MatchResult, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	elo := core.DefaultElo
	if user, ok := m.authMgr.GetUser(username); ok && user.Elo > 0 {
		elo = user.Elo
	}

	resultCh := make(chan MatchResult, 1)
	entry := &QueueEntry{
		Username:   username,
		Token:      token,
		Elo:        elo,
		EnqueuedAt: time.Now(),
		ResultCh:   resultCh,
	}

	m.queue = append(m.queue, entry)
	m.tryMatchQueueLocked() // Thử ghép ngay lập tức nếu có người phù hợp sẵn

	return resultCh, m.makeCancelFunc(entry)
}
```
* **Cú pháp Go: Channel nhận dữ liệu một chiều `<-chan MatchResult`:** Hàm trả về channel dạng read-only, Client chỉ có thể đọc kết quả ghép trận từ channel này mà không thể gửi bậy vào channel.
* **Closure Hủy hàng đợi `m.makeCancelFunc(entry)`:** Trả về một hàm ẩn danh (closure). Nếu người dùng bấm `Ctrl+C` trước khi có trận, client gọi hàm này để tự động xóa mình ra khỏi hàng đợi.

```go
func (m *Matchmaker) tryMatchQueueLocked() {
	if len(m.queue) < 2 {
		return
	}

	now := time.Now()
	matched := make(map[int]bool)

	for i := 0; i < len(m.queue); i++ {
		if matched[i] {
			continue
		}
		e1 := m.queue[i]
		waitSec1 := int(now.Sub(e1.EnqueuedAt).Seconds())
		allowedDelta1 := 100 + (waitSec1/5)*50 // Mở rộng dải Elo của người 1

		bestJ := -1
		bestDiff := 1000000

		for j := i + 1; j < len(m.queue); j++ {
			if matched[j] || e1.Username == e2.Username {
				continue
			}
			e2 := m.queue[j]
			diff := abs(e1.Elo - e2.Elo)
			waitSec2 := int(now.Sub(e2.EnqueuedAt).Seconds())
			allowedDelta2 := 100 + (waitSec2/5)*50

			maxAllowed := max(allowedDelta1, allowedDelta2)
			if diff <= maxAllowed && diff < bestDiff {
				bestDiff = diff
				bestJ = j
			}
		}

		if bestJ != -1 {
			matched[i] = true
			matched[bestJ] = true
			m.createMatchLocked(e1, m.queue[bestJ], "", m.defaultRows, m.defaultCols)
		}
	}
    // ... Dọn dẹp những người đã được ghép ra khỏi hàng đợi
}
```
* **Thuật toán quét chọn đối thủ tối ưu:** Trong tất cả những người nằm trong dải dung sai cho phép (`diff <= maxAllowed`), chọn người có khoảng cách điểm nhỏ nhất (`diff < bestDiff`) để trận đấu có chất lượng chuyên môn cao nhất.

---

#### Quản lý Phòng Riêng (Private Room Codes)
```go
func (m *Matchmaker) CreatePrivateRoom(rows, cols int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := 0; i < 10; i++ {
		code := generateRoomCode() // Sinh mã 6 ký tự như "7K9PX2"
		if _, exists := m.privateRooms[code]; !exists {
			m.privateRooms[code] = &PrivateRoom{
				Code:      code,
				Rows:      rows,
				Cols:      cols,
				CreatedAt: time.Now(),
			}
			return code, nil
		}
	}
	return "", errors.New("failed to generate unique room code")
}
```
* **Mã phòng thân thiện:** Chuỗi `roomCodeCharset` loại bỏ các ký tự `0`, `O`, `1`, `I` để người dùng đọc và gõ trên terminal không bị nhầm lẫn.

```go
func (m *Matchmaker) JoinPrivateRoom(code, username, token string) (<-chan MatchResult, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, exists := m.privateRooms[code]
	if !exists {
		return nil, nil, errors.New("private room not found or expired")
	}
    // ...
	if room.HostEntry == nil {
		room.HostEntry = entry // Người tạo phòng vào trước giữ chỗ
		return resultCh, cancel, nil
	}

	// Người thứ hai nhập đúng mã phòng -> Kích hoạt trận đấu ngay lập tức
	hostEntry := room.HostEntry
	delete(m.privateRooms, code) // Xóa mã phòng khỏi danh bạ

	m.createMatchLocked(hostEntry, entry, code, room.Rows, room.Cols)
	return resultCh, func() {}, nil
}
```
* **Ghép phòng tức thì:** Người thứ nhất gọi lệnh thì giữ chỗ `HostEntry`. Người thứ hai nhập đúng mã phòng thì hệ thống lấy `HostEntry` ghép cặp với người thứ hai, khởi tạo bàn cờ và xóa mã phòng để không ai khác vào được nữa.

---

# BÀI HỌC 3: HIỆN THỰC HÓA GIAO TIẾP GOMOKU SERVICE (`internal/server/grpc_server.go`)

### 1. Vai trò của `CentralServer`
File này chính là nơi nối kết hợp đồng Protobuf (`pb.GomokuServiceServer`) với toàn bộ logic nghiệp vụ bên dưới (`authMgr`, `matchmaker`, `roomMgr`).

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/server/grpc_server.go`

#### Đăng ký & Đăng nhập
```go
func (s *CentralServer) Register(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	ip := getClientIP(ctx)
	token, err := s.auth.Register(ip, req.Username, req.Password)
	if err != nil {
		return &pb.AuthResponse{Success: false, Message: err.Error()}, nil
	}

	elo := core.DefaultElo
	if u, ok := s.auth.GetUser(req.Username); ok && u.Elo > 0 {
		elo = u.Elo
	}

	return &pb.AuthResponse{
		Success: true,
		Message: "Registration successful",
		Token:   token,
		Elo:     int32(elo),
	}, nil
}
```
* **Lấy IP từ gRPC Peer:** `getClientIP(ctx)` bóc tách địa chỉ IP thật của socket kết nối để phục vụ bộ lọc chống spam Rate Limiter.
* Trả về `Token` phiên và điểm `Elo` ban đầu của tài khoản.

---

#### Kiểm tra Tái kết nối (`CheckRejoin`)
```go
func (s *CentralServer) CheckRejoin(ctx context.Context, req *pb.RejoinRequest) (*pb.RejoinResponse, error) {
	username, ok := UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	room, exists := s.roomMgr.Get(req.MatchId)
	if !exists || room.Status == RoomClosed {
		return &pb.RejoinResponse{Success: false, Message: "Match not found or already closed"}, nil
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	// Xác định vai vế của người chơi (Trắng hay Đen)
	var opponentName string
	var assignedColor, opponentElo, yourElo int32

	if room.White.Username == username {
		assignedColor = int32(core.White)
		opponentName = room.Black.Username
		yourElo = int32(room.White.Elo)
		opponentElo = int32(room.Black.Elo)
	} else if room.Black.Username == username {
		assignedColor = int32(core.Black)
		opponentName = room.White.Username
		yourElo = int32(room.Black.Elo)
		opponentElo = int32(room.White.Elo)
	} else {
		return &pb.RejoinResponse{Success: false, Message: "You are not a participant in this match"}, nil
	}

	return &pb.RejoinResponse{
		Success:       true,
		Message:       "Match is active and ready for rejoin",
		MatchId:       req.MatchId,
		OpponentName:  opponentName,
		AssignedColor: assignedColor,
		Rows:          int32(room.State.Rows),
		Cols:          int32(room.State.Cols),
		OpponentElo:   opponentElo,
		YourElo:       yourElo,
	}, nil
}
```
* **Kiểm tra an toàn:** Trước khi mở giao diện TUI bàn cờ trên Client, hàm này xác minh xem:
  1. Token có hợp lệ không? (`UserFromContext(ctx)`).
  2. Ván cờ có còn tồn tại không?
  3. Người này có đúng là 1 trong 2 đấu thủ của ván cờ đó không?
  * Nếu hợp lệ, trả về đầy đủ màu quân, kích thước bàn cờ và điểm số để Client sẵn sàng vào lại trận.

---

#### Luồng Stream Thời gian thực 2 chiều (`PlayOnline`)
```go
func (s *CentralServer) PlayOnline(stream pb.GomokuService_PlayOnlineServer) error {
	ctx := stream.Context()
	username, okUser := UserFromContext(ctx)
	token, okToken := TokenFromContext(ctx)
	if !okUser || !okToken {
		return status.Error(codes.Unauthenticated, "unauthorized: missing session in metadata")
	}

	firstMsg, err := stream.Recv()
	if err != nil || firstMsg.MatchId == "" {
		return status.Error(codes.InvalidArgument, "match_id required in initial handshake")
	}

	room, exists := s.roomMgr.Get(firstMsg.MatchId)
	if !exists {
		return status.Error(codes.NotFound, "game room not found or closed")
	}

	sendChan := make(chan *pb.GameSync, 10)
	var player *RoomPlayer

	room.mu.Lock()
	if room.White.Username == username {
		room.White.Token = token
		player = room.White
	} else if room.Black.Username == username {
		room.Black.Token = token
		player = room.Black
	}
	room.mu.Unlock()

	if player == nil {
		return status.Error(codes.PermissionDenied, "player not permitted in this room")
	}

	if room.Status == RoomPaused {
		_, _ = room.Rejoin(token, sendChan)
	} else {
		room.mu.Lock()
		player.Connected = true
		player.SendChan = sendChan
		room.mu.Unlock()
		room.SyncToPlayer(token)
	}
```
* **Bắt tay kết nối (Handshake):**
  1. Nhận gói tin đầu tiên chứa `MatchId`.
  2. Tìm phòng đấu tương ứng trong `roomMgr`.
  3. **Tự động nhận diện Reconnect:** Nếu phòng đang ở trạng thái `RoomPaused` $\rightarrow$ Kích hoạt `room.Rejoin` để hủy đếm lùi ân hạn và phát sóng tiếp tục trận đấu. Nếu là kết nối mới $\rightarrow$ Gọi `SyncToPlayer`.

```go
	errChan := make(chan error, 2)

	// Luồng 1 (Goroutine gửi): Đẩy dữ liệu từ phòng đấu xuống Client
	go func() {
		for syncMsg := range sendChan {
			if err := stream.Send(syncMsg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Luồng 2 (Goroutine nhận): Lắng nghe nước đi từ Client gửi lên Server
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				room.HandleDisconnect(token) // Bắt lỗi socket đứt -> Kích hoạt đếm lùi 45s
				errChan <- err
				return
			}
			if req.Action != nil {
				_ = room.HandleAction(token, req.Action)
			}
		}
	}()

	err = <-errChan
	return err
}
```
* **Mô hình 2 Goroutine song song:**
  * **Luồng gửi:** Đọc liên tục từ kênh `sendChan` và gọi `stream.Send(syncMsg)`.
  * **Luồng nhận:** Gọi `stream.Recv()` lặp vô tận. Khi đối thủ tắt terminal hoặc mất mạng, `stream.Recv()` sẽ trả về lỗi $\rightarrow$ lập tức kích hoạt `room.HandleDisconnect(token)`.
* **`err = <-errChan`:** Lệnh nhận này đứng chờ ở cuối hàm. Khi 1 trong 2 luồng gặp sự cố ngắt mạng, nó nhận mã lỗi và giải phóng toàn bộ tài nguyên stream.

---

# BÀI HỌC 1: TRÍ TUỆ NHÂN TẠO HEURISTIC BOT (`internal/client/bot.go`)

### 1. Khái niệm Thuật toán Heuristic (Hàm lượng giá)
* Đối với cờ Gomoku trên bàn cờ lớn ($20 \times 30 = 600$ ô), nếu dùng thuật toán đệ quy Minimax/Alpha-Beta duyệt sâu 4-5 bước, số nhánh tính toán sẽ bùng nổ lên tới hàng tỷ trường hợp ($600!$), gây đơ máy ngay lập tức.
* Triết lý **Suckless** giải quyết bài toán này bằng **Hàm lượng giá Heuristic cục bộ**:
  * Duyệt qua tất cả các ô còn trống.
  * Giả lập: Nếu Bot (quân Đen) đánh vào ô này thì được bao nhiêu điểm tấn công (**Attack Score**)? Nếu đối thủ (quân Trắng) đánh vào ô này thì đối thủ được bao nhiêu điểm phòng ngự (**Defense Score**)?
  * Cộng hai điểm này lại: $\text{Total Score} = \text{Attack} + \text{Defense}$. Ô cờ nào có tổng điểm cao nhất chính là tử huyệt (vừa giúp ta tấn công, vừa chặn đường đối thủ).
  * Độ phức tạp chỉ là $O(N \times M)$ — tính toán xong trong chưa đầy 5 miligiây.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/bot.go`

#### Hàm Chọn Nước Đi Tối Ưu (`GetBotMove`)
```go
package client

import (
	"math/rand"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
)

func GetBotMove(state *core.GameState) [2]int {
	bestScore := -1
	var bestMoves [][2]int

	for y := 0; y < state.Rows; y++ {
		for x := 0; x < state.Cols; x++ {
			if state.Board[y][x] == core.Empty {
				attackScore := evaluateCell(state, x, y, core.Black)
				defenseScore := evaluateCell(state, x, y, core.White)

				totalScore := attackScore + defenseScore
				if attackScore >= 100000 {
					totalScore += 50000 // Ưu tiên tuyệt đối: Nếu có cơ hội thắng ngay, đánh luôn không cần thủ!
				}

				if totalScore > bestScore {
					bestScore = totalScore
					bestMoves = [][2]int{{x, y}}
				} else if totalScore == bestScore {
					bestMoves = append(bestMoves, [2]int{x, y})
				}
			}
		}
	}

	if bestScore == -1 || len(bestMoves) == 0 {
		return [2]int{state.Cols / 2, state.Rows / 2} // Bàn cờ trống hoàn toàn -> Đánh vào chính giữa
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return bestMoves[rng.Intn(len(bestMoves))]
}
```
* **`bestMoves [][2]int`:** Nếu có nhiều ô cờ cùng đạt điểm số cao nhất bằng nhau, Bot gom tất cả vào slice này.
* **`attackScore >= 100000`:** Điểm $100.000$ tương đương chuỗi 5 quân (thắng cuộc). Nếu nước này đánh vào là thắng luôn, Bot sẽ cộng thêm $50.000$ điểm ưu tiên để kết liễu trận đấu thay vì mải mê đi phòng thủ nước khác.
* **`rng := rand.New(...)`:** Chọn ngẫu nhiên một trong các nước cờ đồng điểm cao nhất. Việc này giúp nước đi của Bot tự nhiên, không bị rập khuôn máy móc qua từng ván đấu.

---

#### Hàm Đánh giá Ô cờ & Bảng điểm Heuristic
```go
func evaluateCell(state *core.GameState, x, y int, color uint8) int {
	score := 0
	for _, dir := range core.WinDirections {
		c1, o1, _ := core.ScanRay(state.Board, state.Cols, state.Rows, x, y, dir[0], dir[1], color)
		c2, o2, _ := core.ScanRay(state.Board, state.Cols, state.Rows, x, y, -dir[0], -dir[1], color)
		consecutive := c1 + c2 + 1
		openEnds := o1 + o2
		score += calculateScore(consecutive, openEnds)
	}
	return score
}
```
* **Tận dụng `ScanRay` (đã học ở Module 1):** Phóng 2 tia ngược chiều nhau theo 4 hướng (`Ngang, Dọc, 2 Chéo`).
  * `consecutive = c1 + c2 + 1`: Số quân liên tiếp nếu đặt quân vào ô này.
  * `openEnds = o1 + o2`: Số đầu mở (bị chặn 1 đầu, không bị chặn đầu nào, hay bị chặn cả 2 đầu).

```go
func calculateScore(consecutive, openEnds int) int {
	if consecutive >= 5 {
		return 100000 // Thắng ngay lập tức
	}
	if consecutive == 4 {
		if openEnds == 2 {
			return 10000 // 4 quân mở 2 đầu: Nước cờ tử thần (chắc chắn thắng)
		}
		if openEnds == 1 {
			return 1000  // 4 quân bị chặn 1 đầu: Nguy hiểm, cần chặn gấp
		}
	}
	if consecutive == 3 {
		if openEnds == 2 {
			return 1000  // 3 quân mở 2 đầu: Rất nguy hiểm (sắp thành 4 quân mở)
		}
		if openEnds == 1 {
			return 100
		}
	}
	if consecutive == 2 {
		if openEnds == 2 {
			return 100
		}
		if openEnds == 1 {
			return 10
		}
	}
	return 0
}
```
* **Bảng điểm Heuristic phân cấp:**
  * $\ge 5$ quân: $100.000$ điểm (ưu tiên tối cao).
  * 4 quân mở 2 đầu: $10.000$ điểm (đối thủ không thể chặn được cả 2 đầu cùng lúc).
  * 3 quân mở 2 đầu hoặc 4 quân bị chặn 1 đầu: $1.000$ điểm.

---

# BÀI HỌC 2: BẮT PHÍM VIM & CHUYỂN ĐỔI TỌA ĐỘ CHUỘT (`internal/client/input.go`)

### 1. Khái niệm Xử lý Input
* **Tách rời Input và Logic (Decoupling):** Hàm bắt phím `HandleInput` **không trực tiếp thay đổi bàn cờ hay gọi mạng**. Nó chỉ phân tích xem người dùng vừa bấm gì và xuất ra một cấu trúc ý định: `InputIntent { Action: ActionPlace, X: 5, Y: 7 }`. Tầng Engine bên ngoài sẽ quyết định xem ý định đó có được phép thực thi hay không.
* **Chuột trong Terminal (X11 / SGR Mouse Tracking):** Thư viện `ttbox` bật chế độ theo dõi chuột của terminal bằng mã ANSI escape sequence (`\033[?1000h`). Khi bạn click chuột, terminal gửi về mã tọa độ pixel màn hình $(X_{screen}, Y_{screen})$.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/input.go`

```go
package client

import (
	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/ttbox"
)

const (
	ActionNone = iota
	ActionQuit
	ActionPlace
	ActionRestart
)

type InputIntent struct {
	Action int
	X, Y   int
}

func HandleInput(evt ttbox.Event, state *core.GameState) InputIntent {
	if evt.Type == ttbox.EventKey {
		return handleKeyboard(evt, state)
	} else if evt.Type == ttbox.EventMouse {
		return handleMouse(evt, state)
	}
	return InputIntent{Action: ActionNone}
}
```
* Phân luồng sự kiện: Sự kiện bàn phím (`EventKey`) chuyển sang `handleKeyboard`, sự kiện chuột (`EventMouse`) chuyển sang `handleMouse`.

---

#### Xử lý Bàn phím & Phím Vim
```go
func handleKeyboard(evt ttbox.Event, state *core.GameState) InputIntent {
	if evt.Key == ttbox.KeyEscape || evt.Key == ttbox.KeyCtrlC || evt.Ch == 'q' || evt.Ch == 'Q' {
		return InputIntent{Action: ActionQuit}
	}
	if state.Winner != core.Empty && (evt.Ch == 'r' || evt.Ch == 'R') {
		return InputIntent{Action: ActionRestart}
	}

	if state.SelectedX < 0 || state.SelectedY < 0 {
		state.SelectedX, state.SelectedY = state.Cols/2, state.Rows/2
	}

	if processMovement(evt, state) {
		return InputIntent{Action: ActionNone}
	}

	if evt.Key == ttbox.KeyEnter || evt.Ch == ' ' {
		return InputIntent{Action: ActionPlace, X: state.SelectedX, Y: state.SelectedY}
	}
	return InputIntent{Action: ActionNone}
}
```
* **Phím thoát chuẩn POSIX:** `Esc`, `Ctrl+C` hoặc phím `q`/`Q` đều xuất ra `ActionQuit`.
* **Phím chơi lại:** Chỉ khi ván cờ đã kết thúc (`Winner != Empty`) thì phím `r`/`R` mới kích hoạt `ActionRestart`.
* **Khởi tạo con trỏ:** Nếu con trỏ chưa được đặt vào ô nào (`< 0`), phím bấm đầu tiên sẽ tự đưa con trỏ vào chính giữa bàn cờ (`Cols/2, Rows/2`).
* **Phím đặt quân:** Phím `Enter` hoặc phím cách `Space`.

```go
func processMovement(evt ttbox.Event, state *core.GameState) bool {
	moved := false
	switch evt.Key {
	case ttbox.KeyArrowUp: state.SelectedY--; moved = true
	case ttbox.KeyArrowDown: state.SelectedY++; moved = true
	case ttbox.KeyArrowLeft: state.SelectedX--; moved = true
	case ttbox.KeyArrowRight: state.SelectedX++; moved = true
	default:
		switch evt.Ch {
		case 'k', 'K': state.SelectedY--; moved = true // Vim: k = lên
		case 'j', 'J': state.SelectedY++; moved = true // Vim: j = xuống
		case 'h', 'H': state.SelectedX--; moved = true // Vim: h = trái
		case 'l', 'L': state.SelectedX++; moved = true // Vim: l = phải
		}
	}

	if moved {
		state.SelectedX = max(0, min(state.SelectedX, state.Cols-1))
		state.SelectedY = max(0, min(state.SelectedY, state.Rows-1))
	}
	return moved
}
```
* **Hỗ trợ song song Phím Mũi tên và Phím Vim (`h, j, k, l`):** Chuẩn mực bắt buộc cho người dùng terminal.
* **Chặn biên (Clamping):** `max(0, min(pos, max-1))` đảm bảo con trỏ không bao giờ chạy văng ra ngoài khung bàn cờ.

---

#### Thuật toán Chuyển đổi Tọa độ Chuột (`handleMouse`)
```go
func handleMouse(evt ttbox.Event, state *core.GameState) InputIntent {
	if !evt.Press || evt.Button != ttbox.MouseLeft {
		return InputIntent{Action: ActionNone}
	}

	w, h := ttbox.Size()
	offsetX := (w - (state.Cols * CellWidth)) / 2
	offsetY := (h - state.Rows) / 2

	relX, relY := evt.X-offsetX, evt.Y-offsetY
	if relX < 0 || relX >= state.Cols*CellWidth || relY < 0 || relY >= state.Rows {
		state.SelectedX, state.SelectedY = -1, -1
		return InputIntent{Action: ActionNone}
	}

	boardX, boardY := relX/CellWidth, relY
	if state.SelectedX == boardX && state.SelectedY == boardY {
		state.SelectedX, state.SelectedY = -1, -1
		return InputIntent{Action: ActionPlace, X: boardX, Y: boardY}
	}

	state.SelectedX, state.SelectedY = boardX, boardY
	return InputIntent{Action: ActionNone}
}
```
* **Công thức toán học biến đổi tọa độ:**
  1. `offsetX` và `offsetY` là khoảng đệm từ mép cửa sổ terminal đến mép ngoài của bàn cờ.
  2. `relX = evt.X - offsetX`: Tọa độ click chuột tính từ góc trên-trái của bàn cờ.
  3. `boardX = relX / CellWidth`: Do mỗi ô cờ chiếm 3 ký tự ngang (`CellWidth = 3`: dấu cách - quân cờ - dấu cách), ta chia cho 3 để ra chỉ số cột thực tế trên ma trận bàn cờ!
* **Cơ chế 2-Click an toàn:**
  * Click lần 1: Đưa con trỏ nhắm vào ô cờ (`SelectedX = boardX`).
  * Click lần 2 vào đúng ô đang nhắm: Kích hoạt đặt quân (`ActionPlace`). Cơ chế này giúp người chơi không bao giờ bị click nhầm ô cờ do rung tay.

---

# BÀI HỌC 3: GIAO DIỆN TUI BÀN CỜ xterm-256 (`internal/client/renderer.go`)

### 1. Khái niệm Đồ họa Terminal
* **Alternate Screen Buffer (`\033[?1049h`):** Khi `ttbox.Init()` được gọi, nó chuyển terminal sang màn hình phụ độc lập. Khi bạn thoát game (`ttbox.Close()`), nó chuyển lại màn hình chính của Shell, bảo toàn $100\%$ các lệnh bạn đã gõ trước đó mà không để lại bất kỳ vệt vẽ nào làm bẩn terminal.
* **Double Buffering:** Toàn bộ quá trình vẽ được thực hiện trên một vùng đệm nhị phân trong RAM. Chỉ khi hàm `ttbox.Present()` được gọi, những ô thực sự thay đổi mới được xuất ra màn hình qua 1 lệnh `syscall.Write` duy nhất, triệt tiêu hoàn toàn hiện tượng giật màn hình.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/renderer.go`

#### Hằng số Hiển thị & Bảng màu 256
```go
package client

import (
	"fmt"
	"os"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/ttbox"
)

const CellWidth = 3

const (
	CharDot          = '·' // Ô trống
	CharWhite        = 'O' // Quân Trắng
	CharBlack        = 'X' // Quân Đen
	CharLeftBracket  = '[' // Viền trái con trỏ
	CharRightBracket = ']' // Viền phải con trỏ
)

const (
	ColorDefault    int = ttbox.ColorDefault
	ColorBoardGrid  int = 239 // Màu xám đậm cho dấu chấm lưới
	ColorWhitePiece int = 255 // Màu trắng sáng
	ColorBlackPiece int = 245 // Màu xám bạc
	ColorSelValid   int = 39  // Màu xanh dương (con trỏ tại ô trống hợp lệ)
	ColorSelInvalid int = 196 // Màu đỏ (con trỏ tại ô đã có quân)
	ColorWin        int = 114 // Màu xanh lá dạ quang (5 ô chiến thắng)
	ColorText       int = 250
	ColorTextDim    int = 240
	ColorBgActive   int = 236
	ColorBgModal    int = 235
)
```
* Bảng màu sử dụng các mã màu chuẩn xterm-256 (từ `0` đến `255`).

---

#### Vòng đời & Kiểm tra Kích thước Terminal
```go
func InitUI(rows, cols int) {
	if err := ttbox.Init(); err != nil {
		fmt.Printf("initializing TUI: %v\n", err)
		os.Exit(1)
	}

	termW, termH := ttbox.Size()
	maxCols := termW / CellWidth
	maxRows := termH - 6 // Dành 6 hàng cho tiêu đề, thanh Elo, đồng hồ và hướng dẫn phím

	if cols > maxCols || rows > maxRows {
		ttbox.Close()
		fmt.Printf("terminal size too small. Max capacity: %dx%d\n", maxRows, maxCols)
		os.Exit(1)
	}
	ttbox.EnableMouse()
}

func CloseUI() {
	ttbox.DisableMouse()
	ttbox.Close()
}
```
* **Bảo vệ hiển thị (Fail-safe):** Trước khi vào game, kiểm tra xem cửa sổ terminal hiện tại có đủ diện tích vẽ bàn cờ không. Nếu cửa sổ quá nhỏ, thoát ngay và in thông báo rõ ràng thay vì để giao diện bị vỡ hình.
* **`ttbox.EnableMouse()` & `DisableMouse()`:** Bật và tắt chế độ bắt sự kiện chuột an toàn.

---

#### Đường ống Render Chính (`Render`)
```go
func Render(state *core.GameState, opts RenderOpts) {
	ttbox.Clear()

	drawStatusline(state, opts)
	drawBoard(state)

	if state.Winner != core.Empty {
		drawEndgameBanner(state)
	} else if opts.IsPaused {
		drawPauseBanner(opts)
	} else if opts.DisconnectMsg != "" {
		drawDisconnectBanner(opts.DisconnectMsg)
	} else {
		drawControlsGuide()
	}

	ttbox.Present()
}
```
* **Pipeline tuần tự:**
  1. `ttbox.Clear()`: Xóa sạch bộ đệm đệm trong RAM.
  2. `drawStatusline`: Vẽ tiêu đề, mã phòng, điểm Elo, đồng hồ trận đấu và lượt đi.
  3. `drawBoard`: Vẽ toàn bộ các ô cờ và con trỏ.
  4. Hiển thị Banner theo thứ tự ưu tiên: Kết thúc ván $\rightarrow$ Đang tạm dừng (Grace Period) $\rightarrow$ Ngắt kết nối $\rightarrow$ Hướng dẫn phím bấm.
  5. `ttbox.Present()`: Đẩy dữ liệu ra terminal.

---

#### Vẽ Ô Cờ & Con Trỏ Nhắm
```go
func drawCell(state *core.GameState, x, y, offsetX, offsetY int) {
	ch := CharDot
	fg, bg := ColorBoardGrid, ColorDefault
	isWinPos := state.IsWinPos(x, y)

	switch state.Board[y][x] {
	case core.White:
		ch, fg = CharWhite, ColorWhitePiece
	case core.Black:
		ch, fg = CharBlack, ColorBlackPiece
	}

	if isWinPos {
		fg, bg = ColorBgModal, ColorWin // Ô thắng: chữ đen, nền xanh lá nổi bật
	}

	screenX, screenY := offsetX+(x*CellWidth), offsetY+y
	drawCursor(state, x, y, screenX, screenY, isWinPos, bg)

	ttbox.SetAttr(true, false, false, false) // Bật in đậm (Bold)
	ttbox.SetCell(screenX, screenY, ch, fg, bg)
	ttbox.ResetAttr()
}
```
* **`drawCursor`:**
  * Nếu ô `(x, y)` đang được chọn: Vẽ ký tự `[` ở bên trái (`screenX - 1`) và `]` ở bên phải (`screenX + 1`).
  * Nếu ô đó đang trống: Dấu ngoặc vuông có màu xanh dương (`ColorSelValid = 39`).
  * Nếu ô đó đã có quân cờ rồi: Dấu ngoặc vuông đổi sang màu đỏ cảnh báo (`ColorSelInvalid = 196`), người chơi lập tức nhận biết ô này không thể đặt cờ.

---

#### Vẽ Thanh Trạng thái, Điểm Elo & Đồng hồ Trận đấu
```go
func getPlayerLabels(opts RenderOpts) (string, string) {
	wLabel, bLabel := " WHITE ", " BLACK "
	if opts.OppName != "" {
		if opts.LocalColor == core.White {
			wStr := "You"
			if opts.YourElo > 0 {
				wStr = fmt.Sprintf("You - %d", opts.YourElo)
			}
			bStr := opts.OppName
			if opts.OppElo > 0 {
				bStr = fmt.Sprintf("%s - %d", opts.OppName, opts.OppElo)
			}
			wLabel = fmt.Sprintf(" WHITE (%s) ", wStr)
			bLabel = fmt.Sprintf(" BLACK (%s) ", bStr)
		} else if opts.LocalColor == core.Black {
			// Tương tự cho trường hợp người chơi cầm quân Đen
		}
	}
	return fmt.Sprintf(" %c -%s", CharWhite, wLabel), fmt.Sprintf(" %c -%s", CharBlack, bLabel)
}
```
* Tự động nhận diện vai vế của người chơi (Bạn cầm Trắng hay Đen) để nhúng điểm Elo chính xác: `WHITE (You - 1216)` vs `BLACK (bob - 1184)`.

---

#### Banner Đếm lùi Ân hạn (Grace Period Countdown)
```go
func drawPauseBanner(opts RenderOpts) {
	_, h := ttbox.Size()
	msg := fmt.Sprintf(" * OPPONENT DISCONNECTED - WAITING FOR RECONNECT (%ds) * ", opts.GraceSeconds)
	if opts.PauseMsg != "" {
		msg = fmt.Sprintf(" * %s * ", opts.PauseMsg)
	}

	ttbox.SetAttr(true, false, false, false)
	ttbox.DrawTextCenter(h-2, msg, ColorSelInvalid, ColorDefault)
	ttbox.ResetAttr()
	ttbox.DrawTextCenter(h-1, " Game is paused. Match will automatically resume once reconnected. ", ColorTextDim, ColorDefault)
}
```
* Khi đối thủ rớt mạng, banner này xuất hiện ở cuối màn hình với màu đỏ nổi bật, hiển thị chính xác số giây đếm lùi `(45s)`, `(40s)`... được đồng bộ từ Server qua gRPC stream.

---

# BÀI HỌC 1: TRỪU TƯỢNG HÓA MẠNG (`transport.go` & `transport_lan.go`)

### 1. Khái niệm Interface trong Go
* Trong các ngôn ngữ hướng đối tượng (OOP) như C++ hay Java, bạn phải dùng từ khóa `implements Transport`. Trong Go, tính đa hình là **Implicit (Ngầm định)**:
  * Bạn chỉ cần định nghĩa một `interface` gồm các hàm cần có.
  * Bất kỳ struct nào có đủ các hàm đó thì Go tự động coi nó là một `Transport` mà **không cần khai báo kế thừa**.
  * Nhờ vậy, `engine.RunEngine` chỉ cần gọi `t.Send()` và `t.Receive()`, hoàn toàn không cần biết bên dưới là mạng LAN hay Server trung tâm.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/transport.go`

```go
package client

import (
	"github.com/ttasc/gotermoku/internal/common/core"
)

type Transport interface {
	Send(msg NetMessage) error
	Receive() <-chan NetMessage
	Close()
}
```
* **`Send(msg NetMessage) error`:** Gửi một thông điệp đi (nước đi hoặc yêu cầu restart).
* **`Receive() <-chan NetMessage`:** Trả về một kênh chỉ đọc (`read-only channel`). Engine chỉ việc lắng nghe kênh này để nhận gói tin mới từ đối thủ.
* **`Close()`:** Đóng socket và dọn dẹp luồng mạng khi thoát game.

```go
type NetMessage struct {
	Type                  string // "sync", "move", "restart", "disconnect", "pause", "resume"
	X                     int
	Y                     int
	Board                 [][]uint8
	CurrentTurn           uint8
	Winner                uint8
	WinningPositions      [][2]int
	Message               string
	IsPaused              bool
	GraceSecondsRemaining int
	WhiteElo              int
	BlackElo              int
}

func BroadcastSync(state *core.GameState, t Transport) {
	if t == nil {
		return
	}
	t.Send(NetMessage{
		Type:             "sync",
		Board:            state.Board,
		CurrentTurn:      state.CurrentTurn,
		Winner:           state.Winner,
		WinningPositions: state.WinningPositions,
	})
}
```
* **`NetMessage`:** Cấu trúc dữ liệu trung gian thuần túy của Go, chuyển đổi từ các message gRPC/Protobuf để Engine dễ dàng xử lý mà không bị phụ thuộc vào package `pb`.
* **`BroadcastSync`:** Hàm tiện ích dùng cho bên giữ vai trò Host để đóng gói trạng thái bàn cờ hiện tại và gửi cho đối thủ.

---

### 3. Mạng ngang hàng LAN P2P (`internal/client/transport_lan.go`)

File này cung cấp hàm `InitLANTransport(cfg *Config) Transport`:
* **Nếu là Host (`cfg.IsHost == true`):**
  1. Mở socket lắng nghe cổng TCP (`net.Listen("tcp", ":"+cfg.Port)`).
  2. Dựng một `grpc.NewServer()` ngay trên máy mình.
  3. Dùng một channel `streamOk chan pb.GomokuService_PlayStreamServer` để tạm dừng chương trình cho tới khi Client kết nối tới.
* **Nếu là Client (`cfg.IsHost == false`):**
  1. Gọi `grpc.NewClient(addr, insecure.NewCredentials())` kết nối thẳng vào IP của Host.
  2. Mở stream 2 chiều `client.PlayStream(context.Background())`.
  3. Bật một Goroutine chạy ngầm `readLoop()`: Đọc liên tục từ stream gRPC và đẩy dữ liệu vào channel `incoming` để Engine lấy ra vẽ bàn cờ.

---

# BÀI HỌC 2: CẤU HÌNH CLI CHUẨN POSIX & TỰ GẮN CỔNG (`internal/client/config.go`)

### 1. Kỹ thuật Chuẩn hóa Địa chỉ Mạng
Khi người dùng gõ `--server localhost`, nếu ta đưa thẳng chuỗi này vào `grpc.NewClient`, gRPC sẽ không biết kết nối vào cổng nào. Hàm `formatServerAddr` tự động giải quyết bài toán này.

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/config.go`

```go
func formatServerAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}

	// 1. Kiểm tra xem chuỗi đã có cổng chưa (VD: "localhost:5555" hoặc "[::1]:5555")
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr // Đã có cổng -> Giữ nguyên
	}

	// 2. Nếu là địa chỉ IPv6 thuần chưa có ngoặc vuông (VD: "::1")
	if strings.Contains(addr, ":") && !strings.HasPrefix(addr, "[") {
		return "[" + addr + "]:3333"
	}

	// 3. Chuỗi là hostname (localhost) hoặc IPv4 (127.0.0.1) -> Tự động gắn :3333
	return addr + ":3333"
}
```
* **`net.SplitHostPort`:** Hàm của thư viện chuẩn Go dùng để tách chuỗi thành Host và Port. Nếu tách thành công, nghĩa là người dùng đã nhập port riêng $\rightarrow$ không can thiệp.
* **Chuẩn hóa IPv6 POSIX:** Địa chỉ IPv6 khi đi kèm cổng bắt buộc phải nằm trong ngoặc vuông (ví dụ `[::1]:3333`), hàm này tự động bao bọc ngoặc vuông nếu cần.

```go
	return &Config{
		Rows:          rows,
		Cols:          cols,
		IsOnline:      *hostFlag || *joinAddr != "" || isCentral,
		IsHost:        *hostFlag,
		IsBotMode:     *botFlag,
		JoinAddr:      *joinAddr,
		Port:          *port,
		ServerAddr:    formatServerAddr(*serverAddr),
		Username:      *username,
		Password:      *password,
		IsRegister:    *registerFlag,
		IsCentralMode: isCentral,
		IsCreateRoom:  *createRoom,
		JoinRoomCode:  strings.ToUpper(strings.TrimSpace(*joinRoom)),
		RejoinMatchID: strings.TrimSpace(*rejoin),
		EnableTLS:     *enableTLS || *tlsInsecure, // Bật --tls-insecure tự động bật TLS
		TLSInsecure:   *tlsInsecure,
		CACertFile:    *caCert,
	}
```
* **Logic loại trừ xung đột:** Đảm bảo người dùng không gõ nhầm các cờ trái ngược nhau (ví dụ: vừa `--bot` vừa `--host`, hoặc vừa `--create-room` vừa `--join-room`).

---

# BÀI HỌC 3: KẾT NỐI SERVER TRUNG TÂM, TLS & METADATA (`internal/client/transport_central.go`)

### 1. Khái niệm Mạng Client
* **Metadata Context Injection:** Ở Module 3, ta học Server bóc tách token từ header `authorization`. Ở file này, Client dùng hàm `metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)` để đóng gói token vào header của tất cả các gói tin HTTP/2 gửi đi.
* **Zero-Config TLS:** Khi người dùng bật `--tls-insecure`, Client truyền cấu hình `&tls.Config{InsecureSkipVerify: true}`. Toàn bộ lưu lượng mạng được mã hóa TLS, nhưng bỏ qua bước kiểm tra chữ ký CA (rất tiện lợi cho mạng nội bộ/tự host).

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/transport_central.go`

#### Khởi tạo Socket & Thiết lập Bảo mật
```go
func ConnectCentral(cfg *Config) (*CentralSession, error) {
	fmt.Printf("[NET] Connecting to Central Server at %s...\n", cfg.ServerAddr)

	var creds credentials.TransportCredentials
	if cfg.EnableTLS || cfg.TLSInsecure {
		if cfg.CACertFile != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(cfg.CACertFile, "")
			if err != nil {
				return nil, fmt.Errorf("failed to load CA certificate: %w", err)
			}
		} else {
			creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: cfg.TLSInsecure})
		}
	} else {
		creds = insecure.NewCredentials()
	}
```
* **Phân nhánh bảo mật:**
  * Có file CA cert $\rightarrow$ Dùng `NewClientTLSFromFile`.
  * Có cờ `--tls-insecure` $\rightarrow$ Mã hóa TLS với `InsecureSkipVerify: true`.
  * Có cờ `--tls` thuần túy $\rightarrow$ Dùng kho Root CA của hệ điều hành.
  * Không bật cờ nào $\rightarrow$ Chạy Plaintext thông thường (`insecure.NewCredentials()`).

```go
	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.NewClient(cfg.ServerAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kaParams),
	)
```
* Cấu hình Client tự động gửi HTTP/2 PING sau 10 giây nếu đường truyền không có dữ liệu để giữ kết nối không bị router ngắt.

---

#### Đăng nhập & Bơm Token vào Context
```go
	// ... Sau khi gọi RPC Register hoặc Login thành công và nhận về 'token':
	authCtx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
```
* Mọi lời gọi RPC sau thời điểm này (tạo phòng, xếp hàng, đánh cờ) đều truyền `authCtx` này vào để Server tự động nhận diện danh tính.

---

#### Phân nhánh Kịch bản Ghép trận & Rejoin
```go
	if cfg.RejoinMatchID != "" {
		// Kịch bản 1: Nối lại ván cờ cũ
		rejoinRes, err := client.CheckRejoin(authCtx, &pb.RejoinRequest{MatchId: cfg.RejoinMatchID})
		// ...
	} else if cfg.IsCreateRoom {
		// Kịch bản 2: Tạo phòng riêng và chia sẻ mã
		createRes, err := client.CreatePrivateRoom(authCtx, &pb.CreateRoomRequest{...})
		// Gọi callback thông báo mã phòng ra ngoài an toàn
		if cfg.OnRoomCreated != nil {
			cfg.OnRoomCreated(createRes.RoomCode)
		}
		qStream, err := client.JoinPrivateRoom(authCtx, &pb.JoinRoomRequest{RoomCode: cfg.RoomCode})
		session, err = pollQueueStream(qStream, conn)
	} else if cfg.JoinRoomCode != "" {
		// Kịch bản 3: Nhập mã phòng để vào chơi
		qStream, err := client.JoinPrivateRoom(authCtx, &pb.JoinRoomRequest{RoomCode: cfg.JoinRoomCode})
		session, err = pollQueueStream(qStream, conn)
	} else {
		// Kịch bản 4: Ghép trận tự động theo điểm Elo
		qStream, err := client.QueueMatch(authCtx, &pb.QueueRequest{})
		session, err = pollQueueStream(qStream, conn)
	}
```
* **`pollQueueStream`:** Vòng lặp đọc trạng thái từ stream hàng đợi:
  * Nếu nhận `QUEUE_WAITING` $\rightarrow$ In thông điệp `[WAITING] ...` trên console.
  * Nếu nhận `QUEUE_MATCHED` $\rightarrow$ Thoát vòng lặp và trả về đối tượng `CentralSession`.

---

#### Mở Stream Đấu Cờ Thời Gian Thực (`PlayOnline`)
```go
	playStream, err := client.PlayOnline(authCtx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open play stream: %w", err)
	}

	// Gửi gói tin Handshake ban đầu mang MatchId
	if err := playStream.Send(&pb.OnlineAction{MatchId: session.MatchID}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to handshake game room: %w", err)
	}

	transport := &centralTransport{
		conn:     conn,
		stream:   playStream,
		incoming: make(chan NetMessage, 10),
		matchID:  session.MatchID,
	}
	go transport.readLoop()
```
* Mở stream `PlayOnline`, bắn gói tin đầu tiên chứa `MatchId` để Server liên kết socket này vào đúng phòng cờ.
* Kích hoạt Goroutine `readLoop()` chạy ngầm liên tục giải mã gói tin `pb.GameSync` từ Server thành `NetMessage` của Go.

---

# BÀI HỌC 4: GAME LOOP ĐIỀU PHỐI NON-BLOCKING (`internal/client/engine.go`)

### 1. Kiến trúc Game Loop
Game Loop là một vòng lặp vô tận `for { ... }`. Để đảm bảo game phản hồi mượt mà:
* Không được có bất kỳ hàm nào gây nghẽn (Blocking).
* Hàm đọc sự kiện bàn phím/chuột `ttbox.PollEventTimeout(50ms)` được giới hạn thời gian chờ tối đa 50 miligiây. Nếu không có ai bấm phím, nó trả về ngay để vòng lặp tiếp tục chạy sang bước vẽ bàn cờ (`Render`).

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/engine.go`

#### Vòng lặp Trung tâm (`RunEngine`)
```go
func RunEngine(cfg *Config, state *core.GameState, t Transport) {
	opts := buildRenderOpts(cfg)

	if cfg.IsHost {
		BroadcastSync(state, t)
	}

	botMoves := make(chan [2]int, 1)
	botThinking := false

	for {
		// 1. Kích hoạt suy nghĩ của Bot bất đồng bộ (chế độ chơi với máy)
		if cfg.IsBotMode && state.CurrentTurn == core.Black && state.Winner == core.Empty && !botThinking {
			botThinking = true
			go func(s *core.GameState) {
				time.Sleep(400 * time.Millisecond) // Giả lập độ trễ suy nghĩ
				botMoves <- GetBotMove(s)
			}(state)
		}

		// 2. Nhận nước đi của Bot (Non-blocking select)
		select {
		case move := <-botMoves:
			botThinking = false
			if core.CanPlacePiece(state, move[0], move[1]) {
				core.PlacePiece(state, move[0], move[1], core.Black)
			}
		default: // Nếu Bot chưa nghĩ xong, đi tiếp ngay lập tức, không được đứng chờ!
		}

		// 3. Tiếp nhận và xử lý các sự kiện mạng từ đối thủ
		if t != nil {
			processNetwork(cfg, state, t, &opts)
		}

		// 4. Bắt sự kiện phím bấm / chuột từ người chơi (Timeout 50ms)
		evt, err := ttbox.PollEventTimeout(50 * time.Millisecond)
		if err == nil {
			intent := HandleInput(evt, state)
			if intent.Action == ActionQuit {
				break // Người chơi bấm Esc / Ctrl+C -> Thoát vòng lặp game
			}
			processIntent(cfg, intent, state, t, opts)
		}

		// 5. Vẽ lại giao diện TUI lên terminal
		Render(state, opts)
	}
}
```
* **5 bước nhịp nhàng trong mỗi chu kỳ (Frame):**
  1. Kiểm tra lượt Bot $\rightarrow$ Chạy Goroutine tính toán ngầm.
  2. Đọc kênh `botMoves` bằng `select-default` (không khóa).
  3. Xử lý gói tin mạng từ `t.Receive()`.
  4. Đọc phím bấm/chuột trong tối đa $50\text{ms}$.
  5. Gọi hàm `Render(state, opts)` để cập nhật màn hình terminal.

---

#### Xử lý Sự kiện Mạng (`processNetwork`)
```go
func processNetwork(cfg *Config, state *core.GameState, t Transport, opts *RenderOpts) {
	select {
	case msg := <-t.Receive():
		if msg.Type == "disconnect" {
			opts.IsPaused = false
			opts.DisconnectMsg = msg.Message
		} else if msg.Type == "pause" {
			// Đối thủ rớt mạng -> Tạm dừng trận đấu và bật đếm lùi ân hạn
			opts.IsPaused = true
			opts.GraceSeconds = msg.GraceSecondsRemaining
			opts.PauseMsg = msg.Message
		} else if msg.Type == "resume" {
			// Đối thủ kết nối lại -> Hủy tạm dừng và cập nhật bàn cờ
			opts.IsPaused = false
			opts.PauseMsg = ""
			if len(msg.Board) > 0 && !cfg.IsHost {
				state.Board = msg.Board
				state.CurrentTurn = msg.CurrentTurn
				state.Winner = msg.Winner
			}
		} else if msg.Type == "sync" && !cfg.IsHost {
			// Đồng bộ trạng thái bàn cờ bình thường
			state.Board = msg.Board
			state.CurrentTurn = msg.CurrentTurn
			state.Winner = msg.Winner
			state.WinningPositions = msg.WinningPositions
		}
	default:
		// Không có gói tin mạng mới -> Bỏ qua ngay lập tức
	}
}
```
* Cú pháp `select { case msg := <-t.Receive(): ... default: }`: Đảm bảo nếu đối thủ chưa đánh cờ thì vòng lặp vẫn tiếp tục chạy để vẽ đồng hồ và bắt phím, không bao giờ bị đơ giao diện.

---

#### Xử lý Ý định Người chơi (`processIntent`)
```go
func processIntent(cfg *Config, intent InputIntent, state *core.GameState, t Transport, opts RenderOpts) {
	// KHÓA BÀN CỜ KHI ĐANG TẠM DỪNG (GRACE PERIOD)
	if opts.IsPaused && intent.Action == ActionPlace {
		return
	}

	if intent.Action == ActionPlace {
		if cfg.IsOnline && !cfg.IsHost {
			// CHẾ ĐỘ ONLINE: Gửi yêu cầu nước đi lên Server để kiểm duyệt
			allowedTurn := core.Black
			if cfg.IsCentralMode {
				allowedTurn = cfg.AssignedColor // Kiểm tra đúng màu cờ Trắng/Đen của mình
			}

			if state.CurrentTurn == allowedTurn && core.CanPlacePiece(state, intent.X, intent.Y) {
				t.Send(NetMessage{Type: "move", X: intent.X, Y: intent.Y})
			}
			return
		}

		// CHẾ ĐỘ OFFLINE / HOST: Tự đặt quân và kiểm tra thắng cuộc cục bộ
		turnAllowed := true
		colorToPlace := state.CurrentTurn

		if cfg.IsBotMode || cfg.IsHost {
			turnAllowed = (state.CurrentTurn == core.White)
			colorToPlace = core.White
		}

		if turnAllowed && core.CanPlacePiece(state, intent.X, intent.Y) {
			core.PlacePiece(state, intent.X, intent.Y, colorToPlace)
			if cfg.IsHost {
				BroadcastSync(state, t)
			}
		}
	}
}
```
* **Bảo vệ chống thao tác nhầm:** Khi trận đấu đang tạm dừng (`opts.IsPaused == true`), người chơi bấm `Enter`/`Space` sẽ bị bỏ qua.
* **Quyền hạn (Authority):**
  * Nếu là Client online: Chỉ đóng gói tọa độ gửi lên Server qua `t.Send(NetMessage{Type: "move", ...})`, không tự ý vẽ cờ.
  * Nếu là Host / Offline: Gọi thẳng hàm `core.PlacePiece` và phát sóng cho đối thủ.

---

# BÀI HỌC 5: ĐIỀU PHỐI CLIENT & CONSOLE INTERACTION (`client.go`)

File `internal/client/client.go` thực hiện đúng yêu cầu thiết kế của dự án: **Không dựng thêm màn hình TUI phụ**. Toàn bộ khâu nhập tài khoản, mật khẩu và chờ ghép trận diễn ra trên dòng lệnh POSIX (`stdin`/`stdout`).

```go
func runCentralMatch(cfg *Config) {
	reader := bufio.NewReader(os.Stdin)

	if cfg.Username == "" {
		fmt.Print("Enter username: ")
		cfg.Username = readLine(reader)
	}
	if cfg.Password == "" {
		fmt.Print("Enter password: ")
		cfg.Password = readLine(reader)
	}

	// Kết nối mạng, xác thực và ghép trận trên Console
	session, err := ConnectCentral(cfg)
	if err != nil {
		fmt.Printf("Online error: %v\n", err)
		os.Exit(1)
	}
	defer session.Transport.Close()

	// In thông tin trận đấu và Match ID để người dùng lưu lại Rejoin nếu cần
	fmt.Println("\n==================================================")
	fmt.Printf(" [MATCH READY] Match ID: %s\n", session.MatchID)
	fmt.Printf(" (Tip: Save this Match ID to rejoin if disconnected: --rejoin %s)\n", session.MatchID)
	fmt.Printf(" Opponent: %s (%d Elo) | You: %s (%d Elo)\n",
		cfg.OpponentName, cfg.OpponentElo, colorStr, cfg.YourElo)
	fmt.Println("==================================================")
	fmt.Println("Entering game in 1 second...")
	time.Sleep(1 * time.Second)

	// CHỈ KHỞI TẠO TUI KHI TRẬN ĐẤU THỰC SỰ BẮT ĐẦU!
	state := core.NewGameState(cfg.Rows, cfg.Cols)
	InitUI(cfg.Rows, cfg.Cols)
	defer CloseUI()

	RunEngine(cfg, state, session.Transport)
}
```
* **`defer CloseUI()` & `defer session.Transport.Close()`:** Đảm bảo dù game kết thúc bình thường hay người dùng nhấn `Ctrl+C`, cửa sổ terminal luôn được hoàn trả về chế độ ban đầu và kết nối mạng gRPC được đóng giải phóng an toàn.

---

# BÀI HỌC 1: TRỪU TƯỢNG HÓA MẠNG (`transport.go` & `transport_lan.go`)

### 1. Khái niệm Interface trong Go
* Trong các ngôn ngữ hướng đối tượng (OOP) như C++ hay Java, bạn phải dùng từ khóa `implements Transport`. Trong Go, tính đa hình là **Implicit (Ngầm định)**:
  * Bạn chỉ cần định nghĩa một `interface` gồm các hàm cần có.
  * Bất kỳ struct nào có đủ các hàm đó thì Go tự động coi nó là một `Transport` mà **không cần khai báo kế thừa**.
  * Nhờ vậy, `engine.RunEngine` chỉ cần gọi `t.Send()` và `t.Receive()`, hoàn toàn không cần biết bên dưới là mạng LAN hay Server trung tâm.

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/transport.go`

```go
package client

import (
	"github.com/ttasc/gotermoku/internal/common/core"
)

type Transport interface {
	Send(msg NetMessage) error
	Receive() <-chan NetMessage
	Close()
}
```
* **`Send(msg NetMessage) error`:** Gửi một thông điệp đi (nước đi hoặc yêu cầu restart).
* **`Receive() <-chan NetMessage`:** Trả về một kênh chỉ đọc (`read-only channel`). Engine chỉ việc lắng nghe kênh này để nhận gói tin mới từ đối thủ.
* **`Close()`:** Đóng socket và dọn dẹp luồng mạng khi thoát game.

```go
type NetMessage struct {
	Type                  string // "sync", "move", "restart", "disconnect", "pause", "resume"
	X                     int
	Y                     int
	Board                 [][]uint8
	CurrentTurn           uint8
	Winner                uint8
	WinningPositions      [][2]int
	Message               string
	IsPaused              bool
	GraceSecondsRemaining int
	WhiteElo              int
	BlackElo              int
}

func BroadcastSync(state *core.GameState, t Transport) {
	if t == nil {
		return
	}
	t.Send(NetMessage{
		Type:             "sync",
		Board:            state.Board,
		CurrentTurn:      state.CurrentTurn,
		Winner:           state.Winner,
		WinningPositions: state.WinningPositions,
	})
}
```
* **`NetMessage`:** Cấu trúc dữ liệu trung gian thuần túy của Go, chuyển đổi từ các message gRPC/Protobuf để Engine dễ dàng xử lý mà không bị phụ thuộc vào package `pb`.
* **`BroadcastSync`:** Hàm tiện ích dùng cho bên giữ vai trò Host để đóng gói trạng thái bàn cờ hiện tại và gửi cho đối thủ.

---

### 3. Mạng ngang hàng LAN P2P (`internal/client/transport_lan.go`)

File này cung cấp hàm `InitLANTransport(cfg *Config) Transport`:
* **Nếu là Host (`cfg.IsHost == true`):**
  1. Mở socket lắng nghe cổng TCP (`net.Listen("tcp", ":"+cfg.Port)`).
  2. Dựng một `grpc.NewServer()` ngay trên máy mình.
  3. Dùng một channel `streamOk chan pb.GomokuService_PlayStreamServer` để tạm dừng chương trình cho tới khi Client kết nối tới.
* **Nếu là Client (`cfg.IsHost == false`):**
  1. Gọi `grpc.NewClient(addr, insecure.NewCredentials())` kết nối thẳng vào IP của Host.
  2. Mở stream 2 chiều `client.PlayStream(context.Background())`.
  3. Bật một Goroutine chạy ngầm `readLoop()`: Đọc liên tục từ stream gRPC và đẩy dữ liệu vào channel `incoming` để Engine lấy ra vẽ bàn cờ.

---

# BÀI HỌC 2: CẤU HÌNH CLI CHUẨN POSIX & TỰ GẮN CỔNG (`internal/client/config.go`)

### 1. Kỹ thuật Chuẩn hóa Địa chỉ Mạng
Khi người dùng gõ `--server localhost`, nếu ta đưa thẳng chuỗi này vào `grpc.NewClient`, gRPC sẽ không biết kết nối vào cổng nào. Hàm `formatServerAddr` tự động giải quyết bài toán này.

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/config.go`

```go
func formatServerAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}

	// 1. Kiểm tra xem chuỗi đã có cổng chưa (VD: "localhost:5555" hoặc "[::1]:5555")
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr // Đã có cổng -> Giữ nguyên
	}

	// 2. Nếu là địa chỉ IPv6 thuần chưa có ngoặc vuông (VD: "::1")
	if strings.Contains(addr, ":") && !strings.HasPrefix(addr, "[") {
		return "[" + addr + "]:3333"
	}

	// 3. Chuỗi là hostname (localhost) hoặc IPv4 (127.0.0.1) -> Tự động gắn :3333
	return addr + ":3333"
}
```
* **`net.SplitHostPort`:** Hàm của thư viện chuẩn Go dùng để tách chuỗi thành Host và Port. Nếu tách thành công, nghĩa là người dùng đã nhập port riêng $\rightarrow$ không can thiệp.
* **Chuẩn hóa IPv6 POSIX:** Địa chỉ IPv6 khi đi kèm cổng bắt buộc phải nằm trong ngoặc vuông (ví dụ `[::1]:3333`), hàm này tự động bao bọc ngoặc vuông nếu cần.

```go
	return &Config{
		Rows:          rows,
		Cols:          cols,
		IsOnline:      *hostFlag || *joinAddr != "" || isCentral,
		IsHost:        *hostFlag,
		IsBotMode:     *botFlag,
		JoinAddr:      *joinAddr,
		Port:          *port,
		ServerAddr:    formatServerAddr(*serverAddr),
		Username:      *username,
		Password:      *password,
		IsRegister:    *registerFlag,
		IsCentralMode: isCentral,
		IsCreateRoom:  *createRoom,
		JoinRoomCode:  strings.ToUpper(strings.TrimSpace(*joinRoom)),
		RejoinMatchID: strings.TrimSpace(*rejoin),
		EnableTLS:     *enableTLS || *tlsInsecure, // Bật --tls-insecure tự động bật TLS
		TLSInsecure:   *tlsInsecure,
		CACertFile:    *caCert,
	}
```
* **Logic loại trừ xung đột:** Đảm bảo người dùng không gõ nhầm các cờ trái ngược nhau (ví dụ: vừa `--bot` vừa `--host`, hoặc vừa `--create-room` vừa `--join-room`).

---

# BÀI HỌC 3: KẾT NỐI SERVER TRUNG TÂM, TLS & METADATA (`internal/client/transport_central.go`)

### 1. Khái niệm Mạng Client
* **Metadata Context Injection:** Ở Module 3, ta học Server bóc tách token từ header `authorization`. Ở file này, Client dùng hàm `metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)` để đóng gói token vào header của tất cả các gói tin HTTP/2 gửi đi.
* **Zero-Config TLS:** Khi người dùng bật `--tls-insecure`, Client truyền cấu hình `&tls.Config{InsecureSkipVerify: true}`. Toàn bộ lưu lượng mạng được mã hóa TLS, nhưng bỏ qua bước kiểm tra chữ ký CA (rất tiện lợi cho mạng nội bộ/tự host).

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/transport_central.go`

#### Khởi tạo Socket & Thiết lập Bảo mật
```go
func ConnectCentral(cfg *Config) (*CentralSession, error) {
	fmt.Printf("[NET] Connecting to Central Server at %s...\n", cfg.ServerAddr)

	var creds credentials.TransportCredentials
	if cfg.EnableTLS || cfg.TLSInsecure {
		if cfg.CACertFile != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(cfg.CACertFile, "")
			if err != nil {
				return nil, fmt.Errorf("failed to load CA certificate: %w", err)
			}
		} else {
			creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: cfg.TLSInsecure})
		}
	} else {
		creds = insecure.NewCredentials()
	}
```
* **Phân nhánh bảo mật:**
  * Có file CA cert $\rightarrow$ Dùng `NewClientTLSFromFile`.
  * Có cờ `--tls-insecure` $\rightarrow$ Mã hóa TLS với `InsecureSkipVerify: true`.
  * Có cờ `--tls` thuần túy $\rightarrow$ Dùng kho Root CA của hệ điều hành.
  * Không bật cờ nào $\rightarrow$ Chạy Plaintext thông thường (`insecure.NewCredentials()`).

```go
	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.NewClient(cfg.ServerAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kaParams),
	)
```
* Cấu hình Client tự động gửi HTTP/2 PING sau 10 giây nếu đường truyền không có dữ liệu để giữ kết nối không bị router ngắt.

---

#### Đăng nhập & Bơm Token vào Context
```go
	// ... Sau khi gọi RPC Register hoặc Login thành công và nhận về 'token':
	authCtx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
```
* Mọi lời gọi RPC sau thời điểm này (tạo phòng, xếp hàng, đánh cờ) đều truyền `authCtx` này vào để Server tự động nhận diện danh tính.

---

#### Phân nhánh Kịch bản Ghép trận & Rejoin
```go
	if cfg.RejoinMatchID != "" {
		// Kịch bản 1: Nối lại ván cờ cũ
		rejoinRes, err := client.CheckRejoin(authCtx, &pb.RejoinRequest{MatchId: cfg.RejoinMatchID})
		// ...
	} else if cfg.IsCreateRoom {
		// Kịch bản 2: Tạo phòng riêng và chia sẻ mã
		createRes, err := client.CreatePrivateRoom(authCtx, &pb.CreateRoomRequest{...})
		// Gọi callback thông báo mã phòng ra ngoài an toàn
		if cfg.OnRoomCreated != nil {
			cfg.OnRoomCreated(createRes.RoomCode)
		}
		qStream, err := client.JoinPrivateRoom(authCtx, &pb.JoinRoomRequest{RoomCode: cfg.RoomCode})
		session, err = pollQueueStream(qStream, conn)
	} else if cfg.JoinRoomCode != "" {
		// Kịch bản 3: Nhập mã phòng để vào chơi
		qStream, err := client.JoinPrivateRoom(authCtx, &pb.JoinRoomRequest{RoomCode: cfg.JoinRoomCode})
		session, err = pollQueueStream(qStream, conn)
	} else {
		// Kịch bản 4: Ghép trận tự động theo điểm Elo
		qStream, err := client.QueueMatch(authCtx, &pb.QueueRequest{})
		session, err = pollQueueStream(qStream, conn)
	}
```
* **`pollQueueStream`:** Vòng lặp đọc trạng thái từ stream hàng đợi:
  * Nếu nhận `QUEUE_WAITING` $\rightarrow$ In thông điệp `[WAITING] ...` trên console.
  * Nếu nhận `QUEUE_MATCHED` $\rightarrow$ Thoát vòng lặp và trả về đối tượng `CentralSession`.

---

#### Mở Stream Đấu Cờ Thời Gian Thực (`PlayOnline`)
```go
	playStream, err := client.PlayOnline(authCtx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open play stream: %w", err)
	}

	// Gửi gói tin Handshake ban đầu mang MatchId
	if err := playStream.Send(&pb.OnlineAction{MatchId: session.MatchID}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to handshake game room: %w", err)
	}

	transport := &centralTransport{
		conn:     conn,
		stream:   playStream,
		incoming: make(chan NetMessage, 10),
		matchID:  session.MatchID,
	}
	go transport.readLoop()
```
* Mở stream `PlayOnline`, bắn gói tin đầu tiên chứa `MatchId` để Server liên kết socket này vào đúng phòng cờ.
* Kích hoạt Goroutine `readLoop()` chạy ngầm liên tục giải mã gói tin `pb.GameSync` từ Server thành `NetMessage` của Go.

---

# BÀI HỌC 4: GAME LOOP ĐIỀU PHỐI NON-BLOCKING (`internal/client/engine.go`)

### 1. Kiến trúc Game Loop
Game Loop là một vòng lặp vô tận `for { ... }`. Để đảm bảo game phản hồi mượt mà:
* Không được có bất kỳ hàm nào gây nghẽn (Blocking).
* Hàm đọc sự kiện bàn phím/chuột `ttbox.PollEventTimeout(50ms)` được giới hạn thời gian chờ tối đa 50 miligiây. Nếu không có ai bấm phím, nó trả về ngay để vòng lặp tiếp tục chạy sang bước vẽ bàn cờ (`Render`).

---

### 2. Mổ xẻ chi tiết từng dòng code `internal/client/engine.go`

#### Vòng lặp Trung tâm (`RunEngine`)
```go
func RunEngine(cfg *Config, state *core.GameState, t Transport) {
	opts := buildRenderOpts(cfg)

	if cfg.IsHost {
		BroadcastSync(state, t)
	}

	botMoves := make(chan [2]int, 1)
	botThinking := false

	for {
		// 1. Kích hoạt suy nghĩ của Bot bất đồng bộ (chế độ chơi với máy)
		if cfg.IsBotMode && state.CurrentTurn == core.Black && state.Winner == core.Empty && !botThinking {
			botThinking = true
			go func(s *core.GameState) {
				time.Sleep(400 * time.Millisecond) // Giả lập độ trễ suy nghĩ
				botMoves <- GetBotMove(s)
			}(state)
		}

		// 2. Nhận nước đi của Bot (Non-blocking select)
		select {
		case move := <-botMoves:
			botThinking = false
			if core.CanPlacePiece(state, move[0], move[1]) {
				core.PlacePiece(state, move[0], move[1], core.Black)
			}
		default: // Nếu Bot chưa nghĩ xong, đi tiếp ngay lập tức, không được đứng chờ!
		}

		// 3. Tiếp nhận và xử lý các sự kiện mạng từ đối thủ
		if t != nil {
			processNetwork(cfg, state, t, &opts)
		}

		// 4. Bắt sự kiện phím bấm / chuột từ người chơi (Timeout 50ms)
		evt, err := ttbox.PollEventTimeout(50 * time.Millisecond)
		if err == nil {
			intent := HandleInput(evt, state)
			if intent.Action == ActionQuit {
				break // Người chơi bấm Esc / Ctrl+C -> Thoát vòng lặp game
			}
			processIntent(cfg, intent, state, t, opts)
		}

		// 5. Vẽ lại giao diện TUI lên terminal
		Render(state, opts)
	}
}
```
* **5 bước nhịp nhàng trong mỗi chu kỳ (Frame):**
  1. Kiểm tra lượt Bot $\rightarrow$ Chạy Goroutine tính toán ngầm.
  2. Đọc kênh `botMoves` bằng `select-default` (không khóa).
  3. Xử lý gói tin mạng từ `t.Receive()`.
  4. Đọc phím bấm/chuột trong tối đa $50\text{ms}$.
  5. Gọi hàm `Render(state, opts)` để cập nhật màn hình terminal.

---

#### Xử lý Sự kiện Mạng (`processNetwork`)
```go
func processNetwork(cfg *Config, state *core.GameState, t Transport, opts *RenderOpts) {
	select {
	case msg := <-t.Receive():
		if msg.Type == "disconnect" {
			opts.IsPaused = false
			opts.DisconnectMsg = msg.Message
		} else if msg.Type == "pause" {
			// Đối thủ rớt mạng -> Tạm dừng trận đấu và bật đếm lùi ân hạn
			opts.IsPaused = true
			opts.GraceSeconds = msg.GraceSecondsRemaining
			opts.PauseMsg = msg.Message
		} else if msg.Type == "resume" {
			// Đối thủ kết nối lại -> Hủy tạm dừng và cập nhật bàn cờ
			opts.IsPaused = false
			opts.PauseMsg = ""
			if len(msg.Board) > 0 && !cfg.IsHost {
				state.Board = msg.Board
				state.CurrentTurn = msg.CurrentTurn
				state.Winner = msg.Winner
			}
		} else if msg.Type == "sync" && !cfg.IsHost {
			// Đồng bộ trạng thái bàn cờ bình thường
			state.Board = msg.Board
			state.CurrentTurn = msg.CurrentTurn
			state.Winner = msg.Winner
			state.WinningPositions = msg.WinningPositions
		}
	default:
		// Không có gói tin mạng mới -> Bỏ qua ngay lập tức
	}
}
```
* Cú pháp `select { case msg := <-t.Receive(): ... default: }`: Đảm bảo nếu đối thủ chưa đánh cờ thì vòng lặp vẫn tiếp tục chạy để vẽ đồng hồ và bắt phím, không bao giờ bị đơ giao diện.

---

#### Xử lý Ý định Người chơi (`processIntent`)
```go
func processIntent(cfg *Config, intent InputIntent, state *core.GameState, t Transport, opts RenderOpts) {
	// KHÓA BÀN CỜ KHI ĐANG TẠM DỪNG (GRACE PERIOD)
	if opts.IsPaused && intent.Action == ActionPlace {
		return
	}

	if intent.Action == ActionPlace {
		if cfg.IsOnline && !cfg.IsHost {
			// CHẾ ĐỘ ONLINE: Gửi yêu cầu nước đi lên Server để kiểm duyệt
			allowedTurn := core.Black
			if cfg.IsCentralMode {
				allowedTurn = cfg.AssignedColor // Kiểm tra đúng màu cờ Trắng/Đen của mình
			}

			if state.CurrentTurn == allowedTurn && core.CanPlacePiece(state, intent.X, intent.Y) {
				t.Send(NetMessage{Type: "move", X: intent.X, Y: intent.Y})
			}
			return
		}

		// CHẾ ĐỘ OFFLINE / HOST: Tự đặt quân và kiểm tra thắng cuộc cục bộ
		turnAllowed := true
		colorToPlace := state.CurrentTurn

		if cfg.IsBotMode || cfg.IsHost {
			turnAllowed = (state.CurrentTurn == core.White)
			colorToPlace = core.White
		}

		if turnAllowed && core.CanPlacePiece(state, intent.X, intent.Y) {
			core.PlacePiece(state, intent.X, intent.Y, colorToPlace)
			if cfg.IsHost {
				BroadcastSync(state, t)
			}
		}
	}
}
```
* **Bảo vệ chống thao tác nhầm:** Khi trận đấu đang tạm dừng (`opts.IsPaused == true`), người chơi bấm `Enter`/`Space` sẽ bị bỏ qua.
* **Quyền hạn (Authority):**
  * Nếu là Client online: Chỉ đóng gói tọa độ gửi lên Server qua `t.Send(NetMessage{Type: "move", ...})`, không tự ý vẽ cờ.
  * Nếu là Host / Offline: Gọi thẳng hàm `core.PlacePiece` và phát sóng cho đối thủ.

---

# BÀI HỌC 5: ĐIỀU PHỐI CLIENT & CONSOLE INTERACTION (`client.go`)

File `internal/client/client.go` thực hiện đúng yêu cầu thiết kế của dự án: **Không dựng thêm màn hình TUI phụ**. Toàn bộ khâu nhập tài khoản, mật khẩu và chờ ghép trận diễn ra trên dòng lệnh POSIX (`stdin`/`stdout`).

```go
func runCentralMatch(cfg *Config) {
	reader := bufio.NewReader(os.Stdin)

	if cfg.Username == "" {
		fmt.Print("Enter username: ")
		cfg.Username = readLine(reader)
	}
	if cfg.Password == "" {
		fmt.Print("Enter password: ")
		cfg.Password = readLine(reader)
	}

	// Kết nối mạng, xác thực và ghép trận trên Console
	session, err := ConnectCentral(cfg)
	if err != nil {
		fmt.Printf("Online error: %v\n", err)
		os.Exit(1)
	}
	defer session.Transport.Close()

	// In thông tin trận đấu và Match ID để người dùng lưu lại Rejoin nếu cần
	fmt.Println("\n==================================================")
	fmt.Printf(" [MATCH READY] Match ID: %s\n", session.MatchID)
	fmt.Printf(" (Tip: Save this Match ID to rejoin if disconnected: --rejoin %s)\n", session.MatchID)
	fmt.Printf(" Opponent: %s (%d Elo) | You: %s (%d Elo)\n",
		cfg.OpponentName, cfg.OpponentElo, colorStr, cfg.YourElo)
	fmt.Println("==================================================")
	fmt.Println("Entering game in 1 second...")
	time.Sleep(1 * time.Second)

	// CHỈ KHỞI TẠO TUI KHI TRẬN ĐẤU THỰC SỰ BẮT ĐẦU!
	state := core.NewGameState(cfg.Rows, cfg.Cols)
	InitUI(cfg.Rows, cfg.Cols)
	defer CloseUI()

	RunEngine(cfg, state, session.Transport)
}
```
* **`defer CloseUI()` & `defer session.Transport.Close()`:** Đảm bảo dù game kết thúc bình thường hay người dùng nhấn `Ctrl+C`, cửa sổ terminal luôn được hoàn trả về chế độ ban đầu và kết nối mạng gRPC được đóng giải phóng an toàn.

---

# BÀI HỌC 1: KIỂM THỬ P2P LAN & CỔNG ĐỘNG (`test/lan_integration_test.go`)

### 1. Khái niệm Kỹ thuật cho người mới bắt đầu
* **Tại sao không dùng cổng cố định `3333` trong Test?** Nếu bạn hardcode cổng `3333`, khi bạn chạy test 2 lần liên tiếp hoặc khi trên máy đang có một server khác chạy cổng `3333`, bài test sẽ báo lỗi `bind: address already in use` (cổng đang bận).
* **Bí quyết cổng `:0` của POSIX:** Khi bạn gọi `net.Listen("tcp", "127.0.0.1:0")`, số `0` là một chỉ thị đặc biệt yêu cầu nhân hệ điều hành (Kernel): *"Hãy tự động tìm một cổng bất kỳ đang rảnh rỗi trên máy và gán cho tôi"*. Nhờ đó, hàng trăm bài test có thể chạy song song cùng lúc mà không bao giờ bị đụng cổng.

---

### 2. Mổ xẻ chi tiết từng dòng code `test/lan_integration_test.go`

#### Hàm Lấy Cổng Ngẫu Nhiên (`getFreePort`)
```go
package test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ttasc/gotermoku/internal/client"
	"github.com/ttasc/gotermoku/internal/common/core"
)

func getFreePort(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to obtain free port: %v", err)
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
}
```
* **`t.Fatalf(...)`:** In thông báo lỗi và dừng ngay lập tức bài test nếu không mở được socket.
* **`l.Addr().(*net.TCPAddr).Port`:** Lấy số hiệu cổng thực tế mà hệ điều hành vừa cấp phát.
* **`defer l.Close()`:** Đóng socket ngay sau khi lấy được số cổng để nhường lại cổng đó cho Host Transport sử dụng.

---

#### Bắt tay Đồng bộ & Trao đổi Nước cờ LAN
```go
func TestLAN_MultiplayerIntegration(t *testing.T) {
	port := getFreePort(t)

	hostCfg := &client.Config{
		Rows: 15, Cols: 15, IsOnline: true, IsHost: true, Port: port,
	}
	clientCfg := &client.Config{
		Rows: 15, Cols: 15, IsOnline: true, IsHost: false, JoinAddr: "127.0.0.1", Port: port,
	}

	var hostTransport client.Transport
	hostReady := make(chan struct{})

	// 1. Chạy Host trên một Goroutine độc lập
	go func() {
		hostTransport = client.InitLANTransport(hostCfg)
		close(hostReady) // Báo hiệu Host đã sẵn sàng đón kết nối
	}()

	time.Sleep(100 * time.Millisecond)

	// 2. Client kết nối vào cổng của Host
	clientTransport := client.InitLANTransport(clientCfg)
	defer clientTransport.Close()

	select {
	case <-hostReady:
	case <-time.After(3 * time.Second): // Chống treo test nếu Host không phản hồi
		t.Fatal("timeout: Host did not finish handshake with Client")
	}
	defer hostTransport.Close()
```
* **Tại sao Host phải chạy trong Goroutine?** Vì hàm `InitLANTransport` phía Host sẽ đứng chờ (block) cho đến khi có Client bấm kết nối. Nếu không chạy trên goroutine riêng, chương trình test sẽ bị đơ ngay tại dòng khởi tạo Host!
* **Mẫu hình Timeout (`time.After(3s)`):** Nếu quá 3 giây mà handshake chưa xong $\rightarrow$ Báo lỗi `t.Fatal` và thoát, bảo vệ tiến trình CI/CD không bị treo vô tận.

---

#### Kiểm tra Đồng bộ Bàn cờ & Đổi lượt
```go
	// 3. Host tạo trạng thái bàn cờ ban đầu và phát sóng
	hostState := core.NewGameState(hostCfg.Rows, hostCfg.Cols)
	client.BroadcastSync(hostState, hostTransport)

	// 4. Client nhận gói sync đầu tiên
	select {
	case msg := <-clientTransport.Receive():
		if msg.Type != "sync" || msg.CurrentTurn != core.White {
			t.Fatalf("unexpected initial sync: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for initial sync")
	}

	// 5. Host (Trắng) đi ô (x=7, y=7)
	core.PlacePiece(hostState, 7, 7, core.White)
	client.BroadcastSync(hostState, hostTransport)

	// 6. Client nhận cập nhật: kiểm tra Board[y=7][x=7] là White và lượt chuyển sang Black
	select {
	case msg := <-clientTransport.Receive():
		if msg.Board[7][7] != core.White || msg.CurrentTurn != core.Black {
			t.Fatalf("expected White at (7, 7) and Black turn, got: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Host move")
	}

	// 7. Client (Đen) gửi nước đi tại (x=7, y=8)
	_ = clientTransport.Send(client.NetMessage{Type: "move", X: 7, Y: 8})

	// 8. Host nhận nước đi, kiểm tra luật và đặt quân
	select {
	case msg := <-hostTransport.Receive():
		core.PlacePiece(hostState, msg.X, msg.Y, core.Black)
		client.BroadcastSync(hostState, hostTransport)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Client move on Host")
	}

	// 9. Client nhận xác nhận bàn cờ: ô Board[y=8][x=7] là Black!
	select {
	case msg := <-clientTransport.Receive():
		if msg.Board[8][7] != core.Black || msg.CurrentTurn != core.White {
			t.Fatalf("expected Black at (x=7, y=8), got: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for sync after Client move")
	}

	// 10. Client ngắt kết nối -> Host phải nhận được sự kiện 'disconnect'
	clientTransport.Close()

	select {
	case msg := <-hostTransport.Receive():
		if msg.Type != "disconnect" {
			t.Fatalf("expected 'disconnect', got '%s'", msg.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for disconnect on Host")
	}
}
```
* **Lưu ý chỉ số mảng 2D:** Nước cờ tại tọa độ $X=7, Y=8$ khi truy xuất ma trận bàn cờ phải là:
  $$\text{Board}[Y][X] \equiv \text{Board}[8][7]$$
* **Kiểm tra sự kiện ngắt kết nối:** Đóng `clientTransport.Close()`, Host phải bắt được thông điệp `"disconnect"` ngay lập tức mà không bị crash.

---

# BÀI HỌC 2: SERVER INTEGRATION, BBOLT, ELO & REJOIN (`test/online_integration_test.go`)

### 1. Khái niệm Kỹ thuật cốt lõi
* **Thư mục tạm POSIX `t.TempDir()`:** Go tự động tạo một thư mục tạm thời ngẫu nhiên trên hệ điều hành (ví dụ: `/tmp/TestOnline...`). Mọi file database `bbolt` tạo trong thư mục này sẽ **tự động bị xóa sạch $100\%$** sau khi bài test chạy xong, không để lại bất kỳ file rác nào trên ổ cứng.
* **Cấu hình `keepalive.EnforcementPolicy` cho Server Test:** Như đã phân tích ở bài học trước, trong môi trường kiểm thử tốc độ cao, ta bắt buộc phải đặt `MinTime: 1 * time.Second` và `PermitWithoutStream: true` để Server không hiểu nhầm các gói tin PING dồn dập là tấn công DDoS (`too_many_pings`).

---

### 2. Mổ xẻ chi tiết từng dòng code `test/online_integration_test.go`

#### Khởi tạo Test Server Hoàn chỉnh (`setupTestServer`)
```go
func setupTestServer(t *testing.T) (string, *server.UserStore, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "gotermoku_test.db")
	userStore, err := server.NewUserStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create bbolt test db: %v", err)
	}

	authMgr := server.NewAuthManager(userStore)
	roomMgr := server.NewRoomManager()
	matchmaker := server.NewMatchmaker(roomMgr, authMgr, 15, 15)
	centralSrv := server.NewCentralServer(authMgr, matchmaker, roomMgr)

	ln, err := net.Listen("tcp", "127.0.0.1:0") // Cổng động ngẫu nhiên
	serverAddr := ln.Addr().String()

	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             1 * time.Second,
		PermitWithoutStream: true,
	}

	grpcSrv := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaPolicy),
		grpc.UnaryInterceptor(server.UnaryAuthInterceptor(authMgr)),
		grpc.StreamInterceptor(server.StreamAuthInterceptor(authMgr)),
	)
	pb.RegisterGomokuServiceServer(grpcSrv, centralSrv)

	go func() {
		_ = grpcSrv.Serve(ln)
	}()

	teardown := func() {
		grpcSrv.Stop()
		_ = userStore.Close()
	}

	return serverAddr, userStore, teardown
}
```
* **`t.Helper()`:** Đánh dấu đây là hàm trợ giúp kiểm thử. Khi xảy ra lỗi, Go sẽ trích xuất số dòng trong hàm test chính thay vì trỏ vào hàm `setupTestServer`.
* **Hàm Teardown:** Trả về một hàm dọn dẹp đóng cả `grpcSrv` và `userStore` (đóng database bbolt an toàn).

---

#### Test 1: Ghép trận Elo & Kiểm tra Cập nhật bbolt (`TestOnline_MatchmakingAndEloUpdate`)
```go
func TestOnline_MatchmakingAndEloUpdate(t *testing.T) {
	serverAddr, userStore, teardown := setupTestServer(t)
	defer teardown()

	// Khởi tạo 2 cấu hình đăng ký tài khoản mới (Elo khởi đầu: 1200)
	c1Cfg := &client.Config{ServerAddr: serverAddr, Username: "player_elo_1", Password: "password123", IsRegister: true, IsCentralMode: true, IsOnline: true}
	c2Cfg := &client.Config{ServerAddr: serverAddr, Username: "player_elo_2", Password: "password123", IsRegister: true, IsCentralMode: true, IsOnline: true}

	c1Chan := make(chan *client.CentralSession, 1)
	c2Chan := make(chan *client.CentralSession, 1)

	// Chạy 2 client kết nối đồng thời trên 2 Goroutine riêng biệt
	go func() { s, _ := client.ConnectCentral(c1Cfg); c1Chan <- s }()
	time.Sleep(50 * time.Millisecond)
	go func() { s, _ := client.ConnectCentral(c2Cfg); c2Chan <- s }()

	s1 := <-c1Chan
	defer s1.Transport.Close()
	s2 := <-c2Chan
	defer s2.Transport.Close()
```
* **Tránh Deadlock:** Cả hai Client bắt buộc phải chạy bất đồng bộ qua `go func()`, vì hàm `ConnectCentral` sẽ dừng chờ cho đến khi tìm được đối thủ.

```go
	// Cho quân Trắng đánh 5 nước liên tiếp để thắng trận
	for i := 0; i < 4; i++ {
		_ = whiteSession.Transport.Send(client.NetMessage{Type: "move", X: i, Y: 0})
		expectMsg(t, whiteSession, "sync")
		expectMsg(t, blackSession, "sync")

		_ = blackSession.Transport.Send(client.NetMessage{Type: "move", X: i, Y: 1})
		expectMsg(t, whiteSession, "sync")
		expectMsg(t, blackSession, "sync")
	}

	_ = whiteSession.Transport.Send(client.NetMessage{Type: "move", X: 4, Y: 0}) // Nước cờ thứ 5 -> Thắng
	finalW := expectMsg(t, whiteSession, "sync")
	if finalW.Winner != core.White {
		t.Fatalf("expected Winner White, got %d", finalW.Winner)
	}

	time.Sleep(100 * time.Millisecond) // Đợi giao dịch bbolt ghi xuống đĩa

	// Đọc trực tiếp từ cơ sở dữ liệu bbolt để kiểm tra tính toàn vẹn
	winnerUser, _ := userStore.Get(winnerUsername)
	loserUser, _ := userStore.Get(loserUsername)

	if winnerUser.Elo <= core.DefaultElo || loserUser.Elo >= core.DefaultElo {
		t.Fatalf("Elo calculation failed: winner=%d loser=%d", winnerUser.Elo, loserUser.Elo)
	}
	if winnerUser.Wins != 1 || loserUser.Losses != 1 {
		t.Fatalf("Stat counters failed: wins=%d losses=%d", winnerUser.Wins, loserUser.Losses)
	}
}
```
* **Kiểm tra tính nhất quán (ACID Verification):** Bằng cách gọi trực tiếp `userStore.Get()` trên file `bbolt` thực tế, bài test khẳng định thuật toán Elo và giao dịch ghi dữ liệu đã thực hiện thành công.

---

#### Test 2: Phòng riêng & Triệt tiêu Data Race (`TestOnline_PrivateRoom`)
```go
func TestOnline_PrivateRoom(t *testing.T) {
	serverAddr, _, teardown := setupTestServer(t)
	defer teardown()

	roomCodeChan := make(chan string, 1)

	c1Cfg := &client.Config{
		ServerAddr: serverAddr, Username: "host_user", Password: "password123",
		IsRegister: true, IsCentralMode: true, IsOnline: true, IsCreateRoom: true,
		Rows: 15, Cols: 15,
		OnRoomCreated: func(code string) {
			roomCodeChan <- code // Bắn mã phòng qua channel ngay khi nhận được
		},
	}

	go func() { s, _ := client.ConnectCentral(c1Cfg); c1Chan <- s }()

	// Đứng chờ mã phòng qua channel một cách an toàn (Zero Data Race)
	var roomCode string
	select {
	case roomCode = <-roomCodeChan:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for room code")
	}

	// Client 2 dùng mã phòng vừa nhận để vào
	c2Cfg := &client.Config{
		ServerAddr: serverAddr, Username: "guest_user", Password: "password123",
		IsRegister: true, IsCentralMode: true, IsOnline: true,
		JoinRoomCode: roomCode,
	}
	go func() { s, _ := client.ConnectCentral(c2Cfg); c2Chan <- s }()
```
* **Bí quyết chống Data Race:** Thay vì dùng biến chia sẻ và vòng lặp `for c1Cfg.RoomCode == ""`, ta dùng callback `OnRoomCreated` đẩy dữ liệu vào `roomCodeChan`. Channel trong Go có cơ chế đồng bộ hóa bộ nhớ ngầm (Memory Barrier), triệt tiêu hoàn toàn Data Race trước trình kiểm tra `-race`.

---

#### Test 3: Rớt mạng $\rightarrow$ Ân hạn 45s $\rightarrow$ Rejoin (`TestOnline_DisconnectAndRejoin`)
```go
	// 1. Quân Trắng đánh nước cờ đầu tiên tại (3, 3)
	_ = whiteSession.Transport.Send(client.NetMessage{Type: "move", X: 3, Y: 3})
	expectMsg(t, whiteSession, "sync")
	expectMsg(t, blackSession, "sync")

	// 2. Quân Đen bị rớt mạng đột ngột (đóng kết nối)
	matchID := whiteSession.MatchID
	blackSession.Transport.Close()

	// 3. Quân Trắng phải nhận được thông báo 'pause' kèm số giây đếm ngược
	pauseMsg := expectMsg(t, whiteSession, "pause")
	if !pauseMsg.IsPaused || pauseMsg.GraceSecondsRemaining <= 0 {
		t.Fatalf("expected pause event with grace period, got %+v", pauseMsg)
	}

	// 4. Quân Đen mở lại client với cờ --rejoin <match_id>
	rejoinCfg := &client.Config{
		ServerAddr: serverAddr, Username: blackCfg.Username, Password: blackCfg.Password,
		IsCentralMode: true, IsOnline: true, RejoinMatchID: matchID,
	}

	rejoinSession, err := client.ConnectCentral(rejoinCfg)
	if err != nil {
		t.Fatalf("rejoin failed: %v", err)
	}
	defer rejoinSession.Transport.Close()

	// 5. Cả Quân Trắng và Quân Đen đều nhận được thông báo 'resume' (trận đấu tiếp tục)
	resumeMsg := expectMsg(t, whiteSession, "resume")
	if resumeMsg.IsPaused {
		t.Fatalf("expected match to resume from pause, got %+v", resumeMsg)
	}

	resumeB := expectMsg(t, rejoinSession, "resume")
	if resumeB.IsPaused {
		t.Fatalf("expected black player to receive unpaused state, got %+v", resumeB)
	}

	// 6. Quân Đen tiếp tục đánh nước cờ tiếp theo tại (3, 4)
	_ = rejoinSession.Transport.Send(client.NetMessage{Type: "move", X: 3, Y: 4})

	syncW := expectMsg(t, whiteSession, "sync")
	syncB := expectMsg(t, rejoinSession, "sync")

	if syncW.Board[4][3] != core.Black || syncB.Board[4][3] != core.Black {
		t.Fatalf("expected Black piece at (3, 4) after rejoin, got W=%d B=%d", syncW.Board[4][3], syncB.Board[4][3])
	}
```
* **Kiểm chứng vòng đời Reconnect:**
  1. Đóng kết nối của Black $\rightarrow$ White nhận được sự kiện `pause` với đồng hồ đếm lùi.
  2. Black gọi Rejoin với `matchID` $\rightarrow$ Cả hai bên nhận được sự kiện `resume`.
  3. Bàn cờ cũ được khôi phục nguyên vẹn và ván đấu tiếp tục với nước cờ mới mà không bị gián đoạn.

---

### TỔNG KẾT TOÀN DIỆN KHÓA HỌC KIẾN TRÚC `gotermoku`

Dưới đây là bức tranh kiến trúc tổng thể bạn đã làm chủ:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        GOTERMOKU ARCHITECTURE                          │
├────────────────────────────────────────────────────────────────────────┤
│ 1. PROTOCOL LAYER (Protobuf / gRPC)                                    │
│    api/proto/gomoku.proto (HTTP/2 Binary Streaming, Metadata Auth)     │
├──────────────────────────────────────────────────┬─────────────────────┤
│ 2. COMMON CORE (Zero-Dependency)                 │ 3. STORAGE LAYER    │
│    - state.go : State Model (Single Truth)       │    - storage.go     │
│    - logic.go : O(1) Raycast & Full-Board Draw   │      (bbolt mmap,   │
│    - elo.go   : FIDE Elo Rating Formula (K=32)   │       POSIX flock)  │
├──────────────────────────────────────────────────┴─────────────────────┤
│ 4. SERVER SUBSYSTEM (gotermoku-server)                                 │
│    - interceptor.go : Unary/Stream Middleware (Token to Context)       │
│    - server.go      : Ephemeral In-Memory TLS (ECDSA), Keepalive,      │
│                       POSIX Signal Handling (SIGINT/SIGTERM)           │
│    - room.go        : State Machine (Active/Paused), 45s Grace Timer,  │
│                       Forfeit Logic & Atomic Elo Update                │
│    - matchmaker.go  : Dynamic Elo Pool Expansion, Private Rooms        │
│    - grpc_server.go : GomokuServiceServer Implementation               │
├────────────────────────────────────────────────────────────────────────┤
│ 5. CLIENT SUBSYSTEM (gotermoku)                                        │
│    - config.go            : POSIX CLI Flags & HOST[:PORT] Auto-Format  │
│    - transport.go         : Polymorphic Transport Interface            │
│    - transport_lan.go     : P2P LAN Direct Streaming                   │
│    - transport_central.go : Zero-Config TLS, Metadata Injection, Rejoin│
│    - engine.go            : Non-Blocking Game Loop (50ms Frame Pacing) │
│    - input.go             : Vim Keys, Arrows, 2-Click Mouse Tracking   │
│    - renderer.go          : Double-Buffering xterm-256 via ttbox       │
│    - bot.go               : Heuristic Attack/Defense Evaluation Matrix │
├────────────────────────────────────────────────────────────────────────┤
│ 6. VERIFICATION LAYER (Integration Tests with Race Detector)           │
│    - lan_integration_test.go    : Ephemeral Port (:0) P2P Handshake    │
│    - online_integration_test.go : bbolt, Elo, Private Room, Rejoin    │
└────────────────────────────────────────────────────────────────────────┘
```

Hệ thống đạt được sự cân bằng tuyệt đối giữa hai triết lý:
* **Chuẩn mực POSIX:** Tương thích cao với hệ thống UNIX, quản lý tiến trình bằng tín hiệu hệ điều hành, khóa file nguyên tử, và chạy trên terminal thuần túy.
* **Triết lý Suckless:** Không bloat, không kéo thư viện bừa bãi, mã nguồn mạch lạc, mỗi module chỉ làm tốt một việc duy nhất.

Bây giờ bạn đã hoàn toàn tự tin để đọc hiểu, bảo trì, hoặc viết thêm bất kỳ tính năng mới nào cho `gotermoku`!
