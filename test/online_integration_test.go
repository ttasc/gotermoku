package test

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/ttasc/gotermoku/internal/client"
	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/gotermoku/internal/common/pb"
	"github.com/ttasc/gotermoku/internal/server"
)

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

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = userStore.Close()
		t.Fatalf("failed to bind server port: %v", err)
	}
	serverAddr := ln.Addr().String()

	// Cấu hình Keepalive Enforcement Policy cho phép ping thường xuyên trong môi trường Test
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

// 1. Kiểm thử Ghép trận Elo & Tự động cập nhật điểm vào bbolt
func TestOnline_MatchmakingAndEloUpdate(t *testing.T) {
	serverAddr, userStore, teardown := setupTestServer(t)
	defer teardown()

	c1Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "player_elo_1",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
	}

	c2Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "player_elo_2",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
	}

	c1Chan := make(chan *client.CentralSession, 1)
	c2Chan := make(chan *client.CentralSession, 1)

	go func() {
		s, err := client.ConnectCentral(c1Cfg)
		if err != nil {
			t.Errorf("c1 connect failed: %v", err)
			return
		}
		c1Chan <- s
	}()

	time.Sleep(50 * time.Millisecond)

	go func() {
		s, err := client.ConnectCentral(c2Cfg)
		if err != nil {
			t.Errorf("c2 connect failed: %v", err)
			return
		}
		c2Chan <- s
	}()

	var s1, s2 *client.CentralSession
	select {
	case s1 = <-c1Chan:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for c1")
	}
	defer s1.Transport.Close()

	select {
	case s2 = <-c2Chan:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for c2")
	}
	defer s2.Transport.Close()

	var whiteSession, blackSession *client.CentralSession
	if s1.AssignedColor == core.White {
		whiteSession, blackSession = s1, s2
	} else {
		whiteSession, blackSession = s2, s1
	}

	expectMsg(t, whiteSession, "sync")
	expectMsg(t, blackSession, "sync")

	// Đánh cờ cho đến khi Quân Trắng thắng (5 ô liên tiếp hàng y=0)
	for i := 0; i < 4; i++ {
		_ = whiteSession.Transport.Send(client.NetMessage{Type: "move", X: i, Y: 0})
		expectMsg(t, whiteSession, "sync")
		expectMsg(t, blackSession, "sync")

		_ = blackSession.Transport.Send(client.NetMessage{Type: "move", X: i, Y: 1})
		expectMsg(t, whiteSession, "sync")
		expectMsg(t, blackSession, "sync")
	}

	_ = whiteSession.Transport.Send(client.NetMessage{Type: "move", X: 4, Y: 0})

	finalW := expectMsg(t, whiteSession, "sync")
	if finalW.Winner != core.White {
		t.Fatalf("expected Winner White (%d), got %d", core.White, finalW.Winner)
	}

	time.Sleep(100 * time.Millisecond)

	// Kiểm tra bbolt: Thắng tăng Elo, Thua giảm Elo
	winnerUsername := c1Cfg.Username
	loserUsername := c2Cfg.Username
	if s1.AssignedColor != core.White {
		winnerUsername = c2Cfg.Username
		loserUsername = c1Cfg.Username
	}

	winnerUser, _ := userStore.Get(winnerUsername)
	loserUser, _ := userStore.Get(loserUsername)

	if winnerUser.Elo <= core.DefaultElo {
		t.Fatalf("expected winner Elo > %d, got %d", core.DefaultElo, winnerUser.Elo)
	}
	if loserUser.Elo >= core.DefaultElo {
		t.Fatalf("expected loser Elo < %d, got %d", core.DefaultElo, loserUser.Elo)
	}
	if winnerUser.Wins != 1 || loserUser.Losses != 1 {
		t.Fatalf("expected 1 win and 1 loss in bbolt, got wins=%d, losses=%d", winnerUser.Wins, loserUser.Losses)
	}
}

// 2. Kiểm thử Tạo phòng riêng & Vào phòng bằng mã Code (Zero Data Race)
func TestOnline_PrivateRoom(t *testing.T) {
	serverAddr, _, teardown := setupTestServer(t)
	defer teardown()

	roomCodeChan := make(chan string, 1)

	c1Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "host_user",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
		IsCreateRoom:  true,
		Rows:          15,
		Cols:          15,
		OnRoomCreated: func(code string) {
			roomCodeChan <- code
		},
	}

	c1Chan := make(chan *client.CentralSession, 1)
	go func() {
		s, err := client.ConnectCentral(c1Cfg)
		if err != nil {
			t.Errorf("host create room failed: %v", err)
			return
		}
		c1Chan <- s
	}()

	var roomCode string
	select {
	case roomCode = <-roomCodeChan:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for room code")
	}

	if len(roomCode) != 6 {
		t.Fatalf("expected 6-char room code, got '%s'", roomCode)
	}

	c2Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "guest_user",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
		JoinRoomCode:  roomCode,
	}

	c2Chan := make(chan *client.CentralSession, 1)
	go func() {
		s, err := client.ConnectCentral(c2Cfg)
		if err != nil {
			t.Errorf("guest join room failed: %v", err)
			return
		}
		c2Chan <- s
	}()

	var s1, s2 *client.CentralSession
	select {
	case s1 = <-c1Chan:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for host session")
	}
	defer s1.Transport.Close()

	select {
	case s2 = <-c2Chan:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for guest session")
	}
	defer s2.Transport.Close()

	if s1.MatchID != s2.MatchID {
		t.Fatalf("private room MatchID mismatch: %s vs %s", s1.MatchID, s2.MatchID)
	}
}

// 3. Kiểm thử Rớt mạng -> Tạm dừng Ân hạn (Grace Period) -> Rejoin kết nối lại thành công
func TestOnline_DisconnectAndRejoin(t *testing.T) {
	serverAddr, _, teardown := setupTestServer(t)
	defer teardown()

	c1Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "rejoin_player1",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
	}

	c2Cfg := &client.Config{
		ServerAddr:    serverAddr,
		Username:      "rejoin_player2",
		Password:      "password123",
		IsRegister:    true,
		IsCentralMode: true,
		IsOnline:      true,
	}

	c1Chan := make(chan *client.CentralSession, 1)
	c2Chan := make(chan *client.CentralSession, 1)

	go func() {
		s, err := client.ConnectCentral(c1Cfg)
		if err != nil {
			t.Errorf("c1 connect failed: %v", err)
			return
		}
		c1Chan <- s
	}()

	time.Sleep(50 * time.Millisecond)

	go func() {
		s, err := client.ConnectCentral(c2Cfg)
		if err != nil {
			t.Errorf("c2 connect failed: %v", err)
			return
		}
		c2Chan <- s
	}()

	var s1, s2 *client.CentralSession
	select {
	case s1 = <-c1Chan:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for s1 to match")
	}
	defer s1.Transport.Close()

	select {
	case s2 = <-c2Chan:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for s2 to match")
	}

	var whiteSession, blackSession *client.CentralSession
	var blackCfg *client.Config
	if s1.AssignedColor == core.White {
		whiteSession, blackSession = s1, s2
		blackCfg = c2Cfg
	} else {
		whiteSession, blackSession = s2, s1
		blackCfg = c1Cfg
	}

	expectMsg(t, whiteSession, "sync")
	expectMsg(t, blackSession, "sync")

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
		ServerAddr:    serverAddr,
		Username:      blackCfg.Username,
		Password:      blackCfg.Password,
		IsCentralMode: true,
		IsOnline:      true,
		RejoinMatchID: matchID,
	}

	rejoinSession, err := client.ConnectCentral(rejoinCfg)
	if err != nil {
		t.Fatalf("rejoin failed: %v", err)
	}
	defer rejoinSession.Transport.Close()

	// 5. Quân Trắng nhận thông báo 'resume'
	resumeMsg := expectMsg(t, whiteSession, "resume")
	if resumeMsg.IsPaused {
		t.Fatalf("expected match to resume from pause, got %+v", resumeMsg)
	}

	// 6. Quân Đen nhận thông báo 'resume' (kèm dữ liệu bàn cờ) và thực hiện tiếp nước đi tại (3, 4)
	resumeB := expectMsg(t, rejoinSession, "resume")
	if resumeB.IsPaused {
		t.Fatalf("expected black player to receive unpaused state, got %+v", resumeB)
	}

	_ = rejoinSession.Transport.Send(client.NetMessage{Type: "move", X: 3, Y: 4})

	syncW := expectMsg(t, whiteSession, "sync")
	syncB := expectMsg(t, rejoinSession, "sync")

	if syncW.Board[4][3] != core.Black || syncB.Board[4][3] != core.Black {
		t.Fatalf("expected Black piece at (3, 4) after rejoin, got W=%d B=%d",
			syncW.Board[4][3], syncB.Board[4][3])
	}
}

func expectMsg(t *testing.T, s *client.CentralSession, expectedType string) client.NetMessage {
	t.Helper()
	select {
	case msg := <-s.Transport.Receive():
		if expectedType != "" && msg.Type != expectedType {
			t.Fatalf("expected message type '%s', got '%s' (message: %s)", expectedType, msg.Type, msg.Message)
		}
		return msg
	case <-time.After(4 * time.Second):
		t.Fatalf("timeout waiting for message '%s' on player", expectedType)
		return client.NetMessage{}
	}
}
