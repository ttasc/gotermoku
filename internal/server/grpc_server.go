package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/gotermoku/internal/common/pb"
)

type CentralServer struct {
	pb.UnimplementedGomokuServiceServer
	auth       *AuthManager
	matchmaker *Matchmaker
	roomMgr    *RoomManager
}

func NewCentralServer(auth *AuthManager, matchmaker *Matchmaker, roomMgr *RoomManager) *CentralServer {
	return &CentralServer{
		auth:       auth,
		matchmaker: matchmaker,
		roomMgr:    roomMgr,
	}
}

func getClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "127.0.0.1"
	}
	if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
		return tcpAddr.IP.String()
	}
	return p.Addr.String()
}

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

func (s *CentralServer) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	ip := getClientIP(ctx)
	token, err := s.auth.Login(ip, req.Username, req.Password)
	if err != nil {
		return &pb.AuthResponse{Success: false, Message: err.Error()}, nil
	}

	elo := core.DefaultElo
	if u, ok := s.auth.GetUser(req.Username); ok && u.Elo > 0 {
		elo = u.Elo
	}

	return &pb.AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		Elo:     int32(elo),
	}, nil
}

func (s *CentralServer) QueueMatch(req *pb.QueueRequest, stream pb.GomokuService_QueueMatchServer) error {
	ctx := stream.Context()
	username, okUser := UserFromContext(ctx)
	token, okToken := TokenFromContext(ctx)
	if !okUser || !okToken {
		return status.Error(codes.Unauthenticated, "unauthenticated")
	}

	if err := stream.Send(&pb.QueueResponse{
		Status:  pb.QueueStatus_QUEUE_WAITING,
		Message: "Searching for an opponent with balanced Elo...",
	}); err != nil {
		return err
	}

	resultCh, cancel := s.matchmaker.Enqueue(username, token)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res, ok := <-resultCh:
		if !ok {
			return stream.Send(&pb.QueueResponse{
				Status:  pb.QueueStatus_QUEUE_CANCELLED,
				Message: "Matchmaking cancelled",
			})
		}
		return stream.Send(&pb.QueueResponse{
			Status:        pb.QueueStatus_QUEUE_MATCHED,
			Message:       "Match found! Starting match...",
			MatchId:       res.MatchID,
			OpponentName:  res.OpponentName,
			AssignedColor: int32(res.AssignedColor),
			Rows:          int32(res.Rows),
			Cols:          int32(res.Cols),
			OpponentElo:   int32(res.OpponentElo),
			YourElo:       int32(res.YourElo),
		})
	}
}

func (s *CentralServer) CreatePrivateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	_, ok := UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	code, err := s.matchmaker.CreatePrivateRoom(int(req.Rows), int(req.Cols))
	if err != nil {
		return &pb.CreateRoomResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.CreateRoomResponse{
		Success:  true,
		Message:  "Private room created successfully",
		RoomCode: code,
	}, nil
}

func (s *CentralServer) JoinPrivateRoom(req *pb.JoinRoomRequest, stream pb.GomokuService_JoinPrivateRoomServer) error {
	ctx := stream.Context()
	username, okUser := UserFromContext(ctx)
	token, okToken := TokenFromContext(ctx)
	if !okUser || !okToken {
		return status.Error(codes.Unauthenticated, "unauthenticated")
	}

	resultCh, cancel, err := s.matchmaker.JoinPrivateRoom(req.RoomCode, username, token)
	if err != nil {
		return stream.Send(&pb.QueueResponse{
			Status:   pb.QueueStatus_QUEUE_ERROR,
			Message:  err.Error(),
			RoomCode: req.RoomCode,
		})
	}
	defer cancel()

	_ = stream.Send(&pb.QueueResponse{
		Status:   pb.QueueStatus_QUEUE_WAITING,
		Message:  fmt.Sprintf("Waiting for opponent in room %s...", req.RoomCode),
		RoomCode: req.RoomCode,
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res, ok := <-resultCh:
		if !ok {
			return stream.Send(&pb.QueueResponse{
				Status:   pb.QueueStatus_QUEUE_CANCELLED,
				Message:  "Private room closed or cancelled",
				RoomCode: req.RoomCode,
			})
		}
		return stream.Send(&pb.QueueResponse{
			Status:        pb.QueueStatus_QUEUE_MATCHED,
			Message:       "Opponent connected! Starting private match...",
			MatchId:       res.MatchID,
			OpponentName:  res.OpponentName,
			AssignedColor: int32(res.AssignedColor),
			Rows:          int32(res.Rows),
			Cols:          int32(res.Cols),
			OpponentElo:   int32(res.OpponentElo),
			YourElo:       int32(res.YourElo),
			RoomCode:      res.RoomCode,
		})
	}
}

func (s *CentralServer) CheckRejoin(ctx context.Context, req *pb.RejoinRequest) (*pb.RejoinResponse, error) {
	username, ok := UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	room, exists := s.roomMgr.Get(req.MatchId)
	if !exists {
		return &pb.RejoinResponse{Success: false, Message: "Match not found or already closed"}, nil
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Status == RoomClosed {
		return &pb.RejoinResponse{Success: false, Message: "Match has already ended"}, nil
	}

	var (
		opponentName  string
		assignedColor int32
		opponentElo   int32
		yourElo       int32
	)

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

func (s *CentralServer) PlayOnline(stream pb.GomokuService_PlayOnlineServer) error {
	ctx := stream.Context()
	username, okUser := UserFromContext(ctx)
	token, okToken := TokenFromContext(ctx)
	if !okUser || !okToken {
		return status.Error(codes.Unauthenticated, "unauthorized: missing session in metadata")
	}

	// Gói tin đầu tiên gửi MatchId
	firstMsg, err := stream.Recv()
	if err != nil {
		return err
	}
	if firstMsg.MatchId == "" {
		return status.Error(codes.InvalidArgument, "match_id required in initial handshake")
	}

	room, exists := s.roomMgr.Get(firstMsg.MatchId)
	if !exists {
		return status.Error(codes.NotFound, "game room not found or closed")
	}

	sendChan := make(chan *pb.GameSync, 10)
	var player *RoomPlayer

	room.mu.Lock()
	// Tự động liên kết token mới nếu người chơi đăng nhập lại
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

	// Nếu phòng đang tạm dừng (Grace Period), thực hiện Rejoin
	if room.Status == RoomPaused {
		_, err := room.Rejoin(token, sendChan)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	} else {
		room.mu.Lock()
		player.Connected = true
		player.SendChan = sendChan
		room.mu.Unlock()
		room.SyncToPlayer(token)
	}

	errChan := make(chan error, 2)

	// Luồng gửi dữ liệu từ phòng đấu về Client
	go func() {
		for syncMsg := range sendChan {
			if err := stream.Send(syncMsg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Luồng tiếp nhận hành động từ Client
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				room.HandleDisconnect(token)
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

func (s *CentralServer) PlayStream(stream pb.GomokuService_PlayStreamServer) error {
	return status.Error(codes.Unimplemented, "PlayStream is reserved for LAN P2P mode")
}
