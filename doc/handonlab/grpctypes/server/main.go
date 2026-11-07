package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "github.com/ttasc/gotermoku/doc/handonlab/grpctypes/api"
)

type profileServer struct {
	api.UnimplementedProfileServiceServer

	mu       sync.Mutex
	profiles []*api.Profile
}

func (s *profileServer) SaveProfile(_ context.Context, req *api.ProfileRequest) (*api.ProfileResponse, error) {
	profile := req.GetProfile()
	if profile == nil || profile.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if profile.GetUpdatedAtUnix() == 0 {
		profile.UpdatedAtUnix = time.Now().Unix()
	}

	s.mu.Lock()
	s.profiles = append(s.profiles, profile)
	s.mu.Unlock()

	// In toàn bộ kiểu dữ liệu để thấy protobuf giải mã đúng.
	fmt.Printf(
		"name=%s age=%d updated_at=%d active=%t score=%.1f tags=%v city=%s country=%s role=%s avatar=%q attributes=%v\n",
		profile.GetName(),
		profile.GetAge(),
		profile.GetUpdatedAtUnix(),
		profile.GetActive(),
		profile.GetScore(),
		profile.GetTags(),
		profile.GetAddress().GetCity(),
		profile.GetAddress().GetCountry(),
		profile.GetRole().String(),
		string(profile.GetAvatar()),
		profile.GetAttributes(),
	)

	return &api.ProfileResponse{
		Success:      true,
		Summary:      fmt.Sprintf("Đã nhận hồ sơ của %s (role=%s)", profile.GetName(), profile.GetRole()),
		SavedProfile: profile,
	}, nil
}

// ListProfiles là server-streaming RPC: server gửi nhiều message trên cùng
// một lời gọi, client đọc lần lượt bằng stream.Recv().
func (s *profileServer) ListProfiles(req *api.ListProfilesRequest, stream api.ProfileService_ListProfilesServer) error {
	s.mu.Lock()
	snapshot := make([]*api.Profile, len(s.profiles))
	copy(snapshot, s.profiles)
	s.mu.Unlock()

	limit := int(req.GetLimit())
	for i, profile := range snapshot {
		if limit > 0 && i >= limit {
			break
		}
		if err := stream.Context().Err(); err != nil {
			return status.Error(codes.Canceled, "client canceled")
		}
		if err := stream.Send(profile); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	addr := flag.String("addr", ":50051", "địa chỉ lắng nghe của gRPC server")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterProfileServiceServer(grpcServer, &profileServer{})

	log.Printf("gRPC Profile server đang chạy tại %s", *addr)
	log.Fatal(grpcServer.Serve(listener))
}
