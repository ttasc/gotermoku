// internal/netio/grpc_transport.go
package netio

import (
	"context"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ttasc/gotermoku/internal/config"
	"github.com/ttasc/gotermoku/internal/netio/pb"
)

// ==========================================
// HOST (SERVER) TRANSPORT
// ==========================================

type grpcHostTransport struct {
	pb.UnimplementedGomokuServiceServer
	server   *grpc.Server
	stream   pb.GomokuService_PlayStreamServer
	incoming chan NetMessage
	streamOk chan pb.GomokuService_PlayStreamServer
}

// PlayStream được gRPC tự động gọi khi Client kết nối tới
func (t *grpcHostTransport) PlayStream(stream pb.GomokuService_PlayStreamServer) error {
	t.streamOk <- stream // Báo cho InitTransport biết đã có Client kết nối
	for {
		req, err := stream.Recv()
		if err != nil {
			t.incoming <- NetMessage{Type: "disconnect"}
			return err
		}
		t.incoming <- NetMessage{
			Type: req.Type,
			X:    int(req.X),
			Y:    int(req.Y),
		}
	}
}

func (t *grpcHostTransport) Send(msg NetMessage) error {
	if t.stream == nil {
		return nil
	}
	sync := &pb.GameSync{
		Type:        msg.Type,
		CurrentTurn: int32(msg.CurrentTurn),
		Winner:      int32(msg.Winner),
	}
	for _, row := range msg.Board {
		sync.Board = append(sync.Board, &pb.BoardRow{Cells: row})
	}
	for _, pos := range msg.WinningPositions {
		sync.WinningPositions = append(sync.WinningPositions, &pb.Position{X: int32(pos[0]), Y: int32(pos[1])})
	}
	return t.stream.Send(sync)
}

func (t *grpcHostTransport) Receive() <-chan NetMessage { return t.incoming }
func (t *grpcHostTransport) Close() { t.server.Stop() }

// ==========================================
// CLIENT TRANSPORT
// ==========================================

type grpcClientTransport struct {
	conn     *grpc.ClientConn
	stream   pb.GomokuService_PlayStreamClient
	incoming chan NetMessage
}

func (t *grpcClientTransport) Send(msg NetMessage) error {
	return t.stream.Send(&pb.ClientAction{
		Type: msg.Type,
		X:    int32(msg.X),
		Y:    int32(msg.Y),
	})
}

func (t *grpcClientTransport) readLoop() {
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

func (t *grpcClientTransport) Receive() <-chan NetMessage { return t.incoming }
func (t *grpcClientTransport) Close() { t.conn.Close() }

// ==========================================
// INITIALIZATION
// ==========================================

func InitTransport(cfg *config.Config) Transport {
	if cfg.IsHost {
		fmt.Printf("Starting gRPC Host... Waiting for client to connect on port %s...\n", cfg.Port)
		ln, err := net.Listen("tcp", ":"+cfg.Port)
		if err != nil {
			fmt.Printf("Network error: %v\n", err)
			os.Exit(1)
		}

		srv := grpc.NewServer()
		t := &grpcHostTransport{
			server:   srv,
			incoming: make(chan NetMessage, 10),
			streamOk: make(chan pb.GomokuService_PlayStreamServer, 1),
		}
		pb.RegisterGomokuServiceServer(srv, t)

		go func() {
			if err := srv.Serve(ln); err != nil {
				fmt.Printf("gRPC serve error: %v\n", err)
			}
		}()

		t.stream = <-t.streamOk // Block chương trình cho tới khi Client bấm join
		return t

	} else {
		addr := fmt.Sprintf("%s:%s", cfg.JoinAddr, cfg.Port)
		fmt.Printf("Connecting to gRPC Host at %s...\n", addr)

		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Connection error: %v\n", err)
			os.Exit(1)
		}

		client := pb.NewGomokuServiceClient(conn)
		stream, err := client.PlayStream(context.Background())
		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			os.Exit(1)
		}

		t := &grpcClientTransport{
			conn:     conn,
			stream:   stream,
			incoming: make(chan NetMessage, 10),
		}
		go t.readLoop()
		return t
	}
}
