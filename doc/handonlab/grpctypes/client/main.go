package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	api "github.com/ttasc/gotermoku/doc/handonlab/grpctypes/api"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "địa chỉ gRPC server")
	name := flag.String("name", "An", "tên trong hồ sơ")
	flag.Parse()

	// insecure chỉ dùng cho lab chạy local. Production phải dùng TLS.
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := api.NewProfileServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	profile := &api.Profile{
		Name:   *name,
		Age:    21,
		Active: true,
		Score:  9.5,
		Tags:   []string{"go", "grpc", "protobuf"},
		Address: &api.Address{
			City:    "Hà Nội",
			Country: "Việt Nam",
		},
		Role:       api.Role_ROLE_EDITOR,
		Avatar:     []byte("demo-avatar"),
		Attributes: map[string]string{"team": "backend", "level": "beginner"},
	}

	// 1. Unary RPC: gửi 1 request, nhận 1 response.
	reply, err := client.SaveProfile(ctx, &api.ProfileRequest{Profile: profile})
	if err != nil {
		log.Fatalf("SaveProfile: %v", err)
	}

	fmt.Println("success:", reply.GetSuccess())
	fmt.Println("summary:", reply.GetSummary())
	fmt.Println("saved.name:", reply.GetSavedProfile().GetName())
	fmt.Println("saved.updated_at_unix:", reply.GetSavedProfile().GetUpdatedAtUnix())
	fmt.Println("saved.attributes:", reply.GetSavedProfile().GetAttributes())

	// 2. Server-streaming RPC: nhận nhiều Profile trên cùng một lời gọi.
	stream, err := client.ListProfiles(ctx, &api.ListProfilesRequest{Limit: 10})
	if err != nil {
		log.Fatalf("ListProfiles: %v", err)
	}

	fmt.Println("\ndanh sách hồ sơ từ server stream:")
	for {
		item, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalf("Recv: %v", err)
		}
		fmt.Printf("- %s | role=%s | tags=%v | city=%s\n",
			item.GetName(), item.GetRole(), item.GetTags(), item.GetAddress().GetCity())
	}
}
