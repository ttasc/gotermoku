package server

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	userContextKey  contextKey = "auth_user"
	tokenContextKey contextKey = "auth_token"
)

// UserFromContext trích xuất Username đã được xác thực từ Context
func UserFromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(userContextKey).(string)
	return u, ok
}

// TokenFromContext trích xuất Token phiên từ Context
func TokenFromContext(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(tokenContextKey).(string)
	return t, ok
}

func isPublicMethod(fullMethod string) bool {
	switch fullMethod {
	case "/pb.GomokuService/Register",
		"/pb.GomokuService/Login",
		"/pb.GomokuService/PlayStream": // Dành riêng cho LAN P2P
		return true
	default:
		return false
	}
}

func extractAndValidateToken(ctx context.Context, auth *AuthManager) (string, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "missing metadata headers")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	rawToken := authHeaders[0]
	// Hỗ trợ cả định dạng "Bearer <token>" hoặc raw token
	token := strings.TrimPrefix(rawToken, "Bearer ")
	token = strings.TrimSpace(token)

	username, valid := auth.ValidateToken(token)
	if !valid {
		return "", "", status.Error(codes.Unauthenticated, "invalid or expired session token")
	}

	return username, token, nil
}

// UnaryAuthInterceptor tự động xác thực token cho các Unary RPCs
func UnaryAuthInterceptor(auth *AuthManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		username, token, err := extractAndValidateToken(ctx, auth)
		if err != nil {
			return nil, err
		}

		newCtx := context.WithValue(ctx, userContextKey, username)
		newCtx = context.WithValue(newCtx, tokenContextKey, token)
		return handler(newCtx, req)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// StreamAuthInterceptor tự động xác thực token cho các Streaming RPCs
func StreamAuthInterceptor(auth *AuthManager) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		username, token, err := extractAndValidateToken(ss.Context(), auth)
		if err != nil {
			return err
		}

		newCtx := context.WithValue(ss.Context(), userContextKey, username)
		newCtx = context.WithValue(newCtx, tokenContextKey, token)
		wrapped := &wrappedServerStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}
