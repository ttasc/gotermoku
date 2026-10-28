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

func TestLAN_MultiplayerIntegration(t *testing.T) {
	port := getFreePort(t)

	hostCfg := &client.Config{
		Rows:     15,
		Cols:     15,
		IsOnline: true,
		IsHost:   true,
		Port:     port,
	}

	clientCfg := &client.Config{
		Rows:     15,
		Cols:     15,
		IsOnline: true,
		IsHost:   false,
		JoinAddr: "127.0.0.1",
		Port:     port,
	}

	// 1. Khởi chạy Host Transport bất đồng bộ (chờ Client kết nối)
	var hostTransport client.Transport
	hostReady := make(chan struct{})

	go func() {
		hostTransport = client.InitLANTransport(hostCfg)
		close(hostReady)
	}()

	// Đợi 100ms để đảm bảo socket của Host đã lắng nghe
	time.Sleep(100 * time.Millisecond)

	// 2. Client kết nối vào Host
	clientTransport := client.InitLANTransport(clientCfg)
	defer clientTransport.Close()

	// Đảm bảo Host đã hoàn tất handshake
	select {
	case <-hostReady:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: Host did not finish handshake with Client")
	}
	defer hostTransport.Close()

	// 3. Host tạo trạng thái bàn cờ ban đầu và phát sóng cho Client
	hostState := core.NewGameState(hostCfg.Rows, hostCfg.Cols)
	client.BroadcastSync(hostState, hostTransport)

	// 4. Client nhận gói tin đồng bộ đầu tiên
	select {
	case msg := <-clientTransport.Receive():
		if msg.Type != "sync" {
			t.Fatalf("expected message type 'sync', got '%s'", msg.Type)
		}
		if msg.CurrentTurn != core.White {
			t.Fatalf("expected CurrentTurn White (%d), got %d", core.White, msg.CurrentTurn)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Client to receive initial board sync")
	}

	// 5. Host (Quân Trắng) đặt quân tại toạ độ (x=7, y=7) và phát sóng
	if !core.CanPlacePiece(hostState, 7, 7) {
		t.Fatal("Host cannot place piece at (7, 7)")
	}
	core.PlacePiece(hostState, 7, 7, core.White)
	client.BroadcastSync(hostState, hostTransport)

	// 6. Client nhận cập nhật: kiểm tra Board[y=7][x=7] là White và lượt chuyển sang Black
	select {
	case msg := <-clientTransport.Receive():
		if msg.Type != "sync" {
			t.Fatalf("expected 'sync', got '%s'", msg.Type)
		}
		if msg.Board[7][7] != core.White {
			t.Fatalf("expected cell (x=7, y=7) to be White (%d), got %d", core.White, msg.Board[7][7])
		}
		if msg.CurrentTurn != core.Black {
			t.Fatalf("expected CurrentTurn to be Black (%d), got %d", core.Black, msg.CurrentTurn)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Client to receive Host's move")
	}

	// 7. Client (Quân Đen) gửi nước đi tại toạ độ (x=7, y=8) lên Host
	err := clientTransport.Send(client.NetMessage{
		Type: "move",
		X:    7,
		Y:    8,
	})
	if err != nil {
		t.Fatalf("failed to send move from Client: %v", err)
	}

	// 8. Host nhận nước đi của Client, kiểm tra luật và đặt quân
	select {
	case msg := <-hostTransport.Receive():
		if msg.Type != "move" || msg.X != 7 || msg.Y != 8 {
			t.Fatalf("expected move message at (x=7, y=8), got %+v", msg)
		}
		if !core.CanPlacePiece(hostState, msg.X, msg.Y) {
			t.Fatalf("Host rejected valid move at (%d, %d)", msg.X, msg.Y)
		}
		core.PlacePiece(hostState, msg.X, msg.Y, core.Black)
		client.BroadcastSync(hostState, hostTransport)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Host to receive Client's move")
	}

	// 9. Client nhận xác nhận: kiểm tra Board[y=8][x=7] phải là Black và lượt về White
	select {
	case msg := <-clientTransport.Receive():
		if msg.Type != "sync" {
			t.Fatalf("expected 'sync', got '%s'", msg.Type)
		}
		if msg.Board[8][7] != core.Black {
			t.Fatalf("expected cell (x=7, y=8) to be Black (%d), got %d", core.Black, msg.Board[8][7])
		}
		if msg.CurrentTurn != core.White {
			t.Fatalf("expected CurrentTurn to return to White (%d), got %d", core.White, msg.CurrentTurn)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Client to receive sync after its move")
	}

	// 10. Client ngắt kết nối -> Host phải nhận được thông điệp 'disconnect'
	clientTransport.Close()

	select {
	case msg := <-hostTransport.Receive():
		if msg.Type != "disconnect" {
			t.Fatalf("expected 'disconnect' event on Host, got '%s'", msg.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Host to detect Client disconnection")
	}
}
