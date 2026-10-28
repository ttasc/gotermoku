package server

import (
	"context"
	"net"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

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
	return &pb.AuthResponse{Success: true, Message: "Registration successful", Token: token}, nil
}

func (s *CentralServer) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	ip := getClientIP(ctx)
	token, err := s.auth.Login(ip, req.Username, req.Password)
	if err != nil {
		return &pb.AuthResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.AuthResponse{Success: true, Message: "Login successful", Token: token}, nil
}

func (s *CentralServer) QueueMatch(req *pb.QueueRequest, stream pb.GomokuService_QueueMatchServer) error {
	username, ok := s.auth.ValidateToken(req.Token)
	if !ok {
		return stream.Send(&pb.QueueResponse{
			Status:  pb.QueueStatus_QUEUE_ERROR,
			Message: "Invalid or expired session token",
		})
	}

	if err := stream.Send(&pb.QueueResponse{
		Status:  pb.QueueStatus_QUEUE_WAITING,
		Message: "Searching for an opponent...",
	}); err != nil {
		return err
	}

	resultCh, cancel := s.matchmaker.Enqueue(username, req.Token)
	defer cancel()

	select {
	case <-stream.Context().Done():
		return stream.Context().Err()
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
		})
	}
}

func (s *CentralServer) PlayOnline(stream pb.GomokuService_PlayOnlineServer) error {
	firstMsg, err := stream.Recv()
	if err != nil {
		return err
	}

	_, ok := s.auth.ValidateToken(firstMsg.Token)
	if !ok {
		return status.Error(codes.Unauthenticated, "unauthorized: invalid token")
	}

	room, exists := s.roomMgr.Get(firstMsg.MatchId)
	if !exists {
		return status.Error(codes.NotFound, "game room not found or closed")
	}

	player := room.getPlayerByToken(firstMsg.Token)
	if player == nil {
		return status.Error(codes.PermissionDenied, "player not permitted in this room")
	}

	errChan := make(chan error, 2)

	go func() {
		for syncMsg := range player.SendChan {
			if err := stream.Send(syncMsg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Chỉ đồng bộ trạng thái ban đầu riêng cho người chơi vừa hoàn tất handshake
	room.SyncToPlayer(firstMsg.Token)

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				room.HandleDisconnect(firstMsg.Token)
				errChan <- err
				return
			}
			if req.Action != nil {
				_ = room.HandleAction(firstMsg.Token, req.Action)
			}
		}
	}()

	err = <-errChan
	return err
}

func (s *CentralServer) PlayStream(stream pb.GomokuService_PlayStreamServer) error {
	return status.Error(codes.Unimplemented, "PlayStream is reserved for LAN P2P mode")
}
