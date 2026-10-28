package test

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/ttasc/gotermoku/internal/client"
	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/gotermoku/internal/common/pb"
	"github.com/ttasc/gotermoku/internal/server"
)

type connectResult struct {
	session *client.CentralSession
	err     error
}

func TestOnline_CentralServerIntegration(t *testing.T) {
	// 1. Khởi tạo kho lưu trữ User tạm thời theo chuẩn POSIX (tự dọn dẹp)
	dbPath := filepath.Join(t.TempDir(), "users_test.json")
	userStore, err := server.NewUserStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create user store: %v", err)
	}

	authMgr := server.NewAuthManager(userStore)
	roomMgr := server.NewRoomManager()
	matchmaker := server.NewMatchmaker(roomMgr, 15, 15)
	centralSrv := server.NewCentralServer(authMgr, matchmaker, roomMgr)

	// 2. Mở cổng socket động (ephemeral port)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind server port: %v", err)
	}
	serverAddr := ln.Addr().String()

	grpcSrv := grpc.NewServer()
	pb.RegisterGomokuServiceServer(grpcSrv, centralSrv)
	go func() {
		_ = grpcSrv.Serve(ln)
	}()
	defer grpcSrv.Stop()

	// 3. Cấu hình 2 Client tham gia ghép trận
	c1Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "player1",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
	}

	c2Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "player2",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
	}

	c1Chan := make(chan connectResult, 1)
	c2Chan := make(chan connectResult, 1)

	// Chạy kết nối và hàng đợi bất đồng bộ cho cả 2 client
	go func() {
		s, err := client.ConnectCentral(c1Cfg)
		c1Chan <- connectResult{session: s, err: err}
	}()

	time.Sleep(50 * time.Millisecond)

	go func() {
		s, err := client.ConnectCentral(c2Cfg)
		c2Chan <- connectResult{session: s, err: err}
	}()

	// 4. Nhận phiên kết nối đã ghép cặp thành công
	var s1, s2 *client.CentralSession

	select {
	case res := <-c1Chan:
		if res.err != nil {
			t.Fatalf("client 1 failed to connect/match: %v", res.err)
		}
		s1 = res.session
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for client 1 to match")
	}
	defer s1.Transport.Close()

	select {
	case res := <-c2Chan:
		if res.err != nil {
			t.Fatalf("client 2 failed to connect/match: %v", res.err)
		}
		s2 = res.session
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for client 2 to match")
	}
	defer s2.Transport.Close()

	// 5. Kiểm tra tính toàn vẹn của kết quả ghép trận
	if s1.MatchID != s2.MatchID {
		t.Fatalf("MatchID mismatch: client 1 had %s, client 2 had %s", s1.MatchID, s2.MatchID)
	}
	if s1.AssignedColor == s2.AssignedColor {
		t.Fatalf("Both players assigned same color: %d", s1.AssignedColor)
	}
	if s1.Rows != 15 || s1.Cols != 15 || s2.Rows != 15 || s2.Cols != 15 {
		t.Fatalf("Board dimensions not synced properly: %dx%d vs %dx%d", s1.Rows, s1.Cols, s2.Rows, s2.Cols)
	}

	// 6. Phân định người chơi Trắng và Đen
	var whiteSession, blackSession *client.CentralSession
	if s1.AssignedColor == core.White {
		whiteSession, blackSession = s1, s2
	} else {
		whiteSession, blackSession = s2, s1
	}

	// Đọc gói tin sync ban đầu trên cả 2 bên (mỗi bên chỉ nhận duy nhất 1 gói)
	expectSyncMsg(t, whiteSession, core.White)
	expectSyncMsg(t, blackSession, core.White)

	// 7. Quân Trắng đánh nước cờ đầu tiên tại toạ độ (x=5, y=5)
	err = whiteSession.Transport.Send(client.NetMessage{
		Type: "move",
		X:    5,
		Y:    5,
	})
	if err != nil {
		t.Fatalf("failed to send move from White player: %v", err)
	}

	// Cả 2 bên nhận được sync cập nhật: Board[y=5][x=5] là White, lượt kế tiếp là Black
	msgW1 := expectSyncMsg(t, whiteSession, core.Black)
	if msgW1.Board[5][5] != core.White {
		t.Fatalf("expected White at (x=5, y=5) on White client, got %d", msgW1.Board[5][5])
	}

	msgB1 := expectSyncMsg(t, blackSession, core.Black)
	if msgB1.Board[5][5] != core.White {
		t.Fatalf("expected White at (x=5, y=5) on Black client, got %d", msgB1.Board[5][5])
	}

	// 8. Quân Đen đánh nước cờ tiếp theo tại toạ độ (x=5, y=6)
	err = blackSession.Transport.Send(client.NetMessage{
		Type: "move",
		X:    5,
		Y:    6,
	})
	if err != nil {
		t.Fatalf("failed to send move from Black player: %v", err)
	}

	// Cả 2 bên nhận được sync cập nhật: Board[y=6][x=5] là Black, lượt về White
	msgW2 := expectSyncMsg(t, whiteSession, core.White)
	if msgW2.Board[6][5] != core.Black {
		t.Fatalf("expected Black at (x=5, y=6) on White client, got %d", msgW2.Board[6][5])
	}

	msgB2 := expectSyncMsg(t, blackSession, core.White)
	if msgB2.Board[6][5] != core.Black {
		t.Fatalf("expected Black at (x=5, y=6) on Black client, got %d", msgB2.Board[6][5])
	}

	// 9. Bên Đen ngắt kết nối -> Bên Trắng phải nhận được thông điệp 'disconnect'
	blackSession.Transport.Close()

	select {
	case msg := <-whiteSession.Transport.Receive():
		if msg.Type != "disconnect" {
			t.Fatalf("expected 'disconnect' event on White client, got '%s'", msg.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for White client to receive disconnect notification")
	}
}

func expectSyncMsg(t *testing.T, s *client.CentralSession, expectedTurn uint8) client.NetMessage {
	t.Helper()
	select {
	case msg := <-s.Transport.Receive():
		if msg.Type != "sync" {
			t.Fatalf("expected message 'sync', got '%s'", msg.Type)
		}
		if expectedTurn != 0 && msg.CurrentTurn != expectedTurn {
			t.Fatalf("expected CurrentTurn %d, got %d", expectedTurn, msg.CurrentTurn)
		}
		return msg
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for sync on player with opponent: %s", s.OpponentName)
		return client.NetMessage{}
	}
}
