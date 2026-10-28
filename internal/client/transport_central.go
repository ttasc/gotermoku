package client

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ttasc/gotermoku/internal/common/pb"
)

type CentralSession struct {
	Transport     Transport
	MatchID       string
	OpponentName  string
	AssignedColor uint8
	Rows          int
	Cols          int
}

type centralTransport struct {
	conn     *grpc.ClientConn
	stream   pb.GomokuService_PlayOnlineClient
	incoming chan NetMessage
	token    string
	matchID  string
}

func (t *centralTransport) Send(msg NetMessage) error {
	action := &pb.ClientAction{
		Type: msg.Type,
		X:    int32(msg.X),
		Y:    int32(msg.Y),
	}
	return t.stream.Send(&pb.OnlineAction{
		Token:   t.token,
		MatchId: t.matchID,
		Action:  action,
	})
}

func (t *centralTransport) Receive() <-chan NetMessage {
	return t.incoming
}

func (t *centralTransport) Close() {
	if t.stream != nil {
		t.stream.CloseSend()
	}
	if t.conn != nil {
		t.conn.Close()
	}
}

func (t *centralTransport) readLoop() {
	for {
		resp, err := t.stream.Recv()
		if err != nil {
			t.incoming <- NetMessage{Type: "disconnect"}
			break
		}

		msg := NetMessage{
			Type:        resp.Type,
			CurrentTurn: uint8(resp.CurrentTurn),
			Winner:      uint8(resp.Winner),
		}
		for _, row := range resp.Board {
			msg.Board = append(msg.Board, row.Cells)
		}
		for _, pos := range resp.WinningPositions {
			msg.WinningPositions = append(msg.WinningPositions, [2]int{int(pos.X), int(pos.Y)})
		}

		t.incoming <- msg
	}
}

func ConnectCentral(cfg *Config) (*CentralSession, error) {
	fmt.Printf("[NET] Connecting to Central Server at %s...\n", cfg.ServerAddr)
	conn, err := grpc.NewClient(cfg.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial server: %w", err)
	}

	client := pb.NewGomokuServiceClient(conn)
	ctx := context.Background()

	var token string
	if cfg.IsRegister {
		fmt.Printf("[AUTH] Registering account '%s'...\n", cfg.Username)
		res, err := client.Register(ctx, &pb.AuthRequest{
			Username: cfg.Username,
			Password: cfg.Password,
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("rpc register error: %w", err)
		}
		if !res.Success {
			conn.Close()
			return nil, fmt.Errorf("registration failed: %s", res.Message)
		}
		token = res.Token
		fmt.Println("[AUTH] Registered successfully!")
	} else {
		fmt.Printf("[AUTH] Logging in as '%s'...\n", cfg.Username)
		res, err := client.Login(ctx, &pb.AuthRequest{
			Username: cfg.Username,
			Password: cfg.Password,
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("rpc login error: %w", err)
		}
		if !res.Success {
			conn.Close()
			return nil, fmt.Errorf("login failed: %s", res.Message)
		}
		token = res.Token
		fmt.Println("[AUTH] Login successful!")
	}

	fmt.Println("[MATCHMAKING] Entering matchmaking queue (Press Ctrl+C to abort)...")
	qStream, err := client.QueueMatch(ctx, &pb.QueueRequest{Token: token})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to queue for match: %w", err)
	}

	var matchResp *pb.QueueResponse
	for {
		resp, err := qStream.Recv()
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("queue connection lost: %w", err)
		}

		if resp.Status == pb.QueueStatus_QUEUE_WAITING {
			fmt.Printf("\r[MATCHMAKING] %s", resp.Message)
			os.Stdout.Sync()
		} else if resp.Status == pb.QueueStatus_QUEUE_MATCHED {
			matchResp = resp
			break
		} else {
			conn.Close()
			return nil, fmt.Errorf("queue ended with status: %v, message: %s", resp.Status, resp.Message)
		}
	}

	playStream, err := client.PlayOnline(context.Background())
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open play stream: %w", err)
	}

	if err := playStream.Send(&pb.OnlineAction{
		Token:   token,
		MatchId: matchResp.MatchId,
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to handshake game room: %w", err)
	}

	transport := &centralTransport{
		conn:     conn,
		stream:   playStream,
		incoming: make(chan NetMessage, 10),
		token:    token,
		matchID:  matchResp.MatchId,
	}

	go transport.readLoop()

	return &CentralSession{
		Transport:     transport,
		MatchID:       matchResp.MatchId,
		OpponentName:  matchResp.OpponentName,
		AssignedColor: uint8(matchResp.AssignedColor),
		Rows:          int(matchResp.Rows),
		Cols:          int(matchResp.Cols),
	}, nil
}
