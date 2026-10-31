package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/ttasc/gotermoku/internal/common/pb"
)

type CentralSession struct {
	Transport     Transport
	MatchID       string
	OpponentName  string
	AssignedColor uint8
	Rows          int
	Cols          int
	OpponentElo   int
	YourElo       int
	RoomCode      string
}

type centralTransport struct {
	conn     *grpc.ClientConn
	stream   pb.GomokuService_PlayOnlineClient
	incoming chan NetMessage
	matchID  string
}

func (t *centralTransport) Send(msg NetMessage) error {
	action := &pb.ClientAction{
		Type: msg.Type,
		X:    int32(msg.X),
		Y:    int32(msg.Y),
	}
	return t.stream.Send(&pb.OnlineAction{
		MatchId: t.matchID,
		Action:  action,
	})
}

func (t *centralTransport) Receive() <-chan NetMessage {
	return t.incoming
}

func (t *centralTransport) Close() {
	if t.stream != nil {
		_ = t.stream.CloseSend()
	}
	if t.conn != nil {
		_ = t.conn.Close()
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
			Type:                  resp.Type,
			CurrentTurn:           uint8(resp.CurrentTurn),
			Winner:                uint8(resp.Winner),
			Message:               resp.Message,
			IsPaused:              resp.IsPaused,
			GraceSecondsRemaining: int(resp.GraceSecondsRemaining),
			WhiteElo:              int(resp.WhiteElo),
			BlackElo:              int(resp.BlackElo),
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

	var creds credentials.TransportCredentials
	if cfg.EnableTLS {
		if cfg.CACertFile != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(cfg.CACertFile, "")
			if err != nil {
				return nil, fmt.Errorf("failed to load CA certificate: %w", err)
			}
		} else {
			creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})
		}
	} else {
		creds = insecure.NewCredentials()
	}

	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.NewClient(cfg.ServerAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kaParams),
	)
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
		cfg.YourElo = int(res.Elo)
		fmt.Printf("[AUTH] Registered successfully! Elo: %d\n", cfg.YourElo)
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
		cfg.YourElo = int(res.Elo)
		fmt.Printf("[AUTH] Login successful! Elo: %d\n", cfg.YourElo)
	}

	authCtx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	var session *CentralSession

	if cfg.RejoinMatchID != "" {
		fmt.Printf("[REJOIN] Checking match '%s' status...\n", cfg.RejoinMatchID)
		rejoinRes, err := client.CheckRejoin(authCtx, &pb.RejoinRequest{MatchId: cfg.RejoinMatchID})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("rpc check rejoin error: %w", err)
		}
		if !rejoinRes.Success {
			conn.Close()
			return nil, fmt.Errorf("cannot rejoin: %s", rejoinRes.Message)
		}

		session = &CentralSession{
			MatchID:       rejoinRes.MatchId,
			OpponentName:  rejoinRes.OpponentName,
			AssignedColor: uint8(rejoinRes.AssignedColor),
			Rows:          int(rejoinRes.Rows),
			Cols:          int(rejoinRes.Cols),
			OpponentElo:   int(rejoinRes.OpponentElo),
			YourElo:       int(rejoinRes.YourElo),
		}

	} else if cfg.IsCreateRoom {
		fmt.Printf("[PRIVATE] Creating private room (%dx%d)...\n", cfg.Rows, cfg.Cols)
		createRes, err := client.CreatePrivateRoom(authCtx, &pb.CreateRoomRequest{
			Rows: int32(cfg.Rows),
			Cols: int32(cfg.Cols),
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("rpc create room error: %w", err)
		}
		if !createRes.Success {
			conn.Close()
			return nil, fmt.Errorf("failed to create room: %s", createRes.Message)
		}

		cfg.RoomCode = createRes.RoomCode
		if cfg.OnRoomCreated != nil {
			cfg.OnRoomCreated(createRes.RoomCode)
		}

		fmt.Println("==================================================")
		fmt.Printf(" [PRIVATE ROOM CREATED] ROOM CODE: %s\n", cfg.RoomCode)
		fmt.Println(" Share this code with your friend to start playing!")
		fmt.Println("==================================================")

		qStream, err := client.JoinPrivateRoom(authCtx, &pb.JoinRoomRequest{RoomCode: cfg.RoomCode})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to wait in private room: %w", err)
		}

		session, err = pollQueueStream(qStream, conn)
		if err != nil {
			return nil, err
		}

	} else if cfg.JoinRoomCode != "" {
		fmt.Printf("[PRIVATE] Joining private room with code '%s'...\n", cfg.JoinRoomCode)
		qStream, err := client.JoinPrivateRoom(authCtx, &pb.JoinRoomRequest{RoomCode: cfg.JoinRoomCode})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to join private room: %w", err)
		}

		session, err = pollQueueStream(qStream, conn)
		if err != nil {
			return nil, err
		}

	} else {
		fmt.Println("[MATCHMAKING] Entering Elo matchmaking queue (Press Ctrl+C to abort)...")
		qStream, err := client.QueueMatch(authCtx, &pb.QueueRequest{})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to queue for match: %w", err)
		}

		session, err = pollQueueStream(qStream, conn)
		if err != nil {
			return nil, err
		}
	}

	playStream, err := client.PlayOnline(authCtx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open play stream: %w", err)
	}

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

	session.Transport = transport
	return session, nil
}

func pollQueueStream(qStream pb.GomokuService_QueueMatchClient, conn *grpc.ClientConn) (*CentralSession, error) {
	for {
		resp, err := qStream.Recv()
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("queue connection lost: %w", err)
		}

		if resp.Status == pb.QueueStatus_QUEUE_WAITING {
			fmt.Printf("\r[WAITING] %s", resp.Message)
			_ = os.Stdout.Sync()
		} else if resp.Status == pb.QueueStatus_QUEUE_MATCHED {
			return &CentralSession{
				MatchID:       resp.MatchId,
				OpponentName:  resp.OpponentName,
				AssignedColor: uint8(resp.AssignedColor),
				Rows:          int(resp.Rows),
				Cols:          int(resp.Cols),
				OpponentElo:   int(resp.OpponentElo),
				YourElo:       int(resp.YourElo),
				RoomCode:      resp.RoomCode,
			}, nil
		} else {
			conn.Close()
			return nil, fmt.Errorf("queue terminated: %s", resp.Message)
		}
	}
}
