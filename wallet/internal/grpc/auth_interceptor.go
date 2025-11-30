package grpc

import (
    "context"
    "strings"

    "aegis/wallet/internal/auth"
    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
)

func AuthUnaryInterceptor(logger *zap.Logger, tm *auth.TokenManager) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if strings.HasSuffix(info.FullMethod, "/RequestNonce") || strings.HasSuffix(info.FullMethod, "/VerifySignature") || strings.HasSuffix(info.FullMethod, "/GetUserByWallet") || strings.HasSuffix(info.FullMethod, "/GetUser") || strings.HasSuffix(info.FullMethod, "/CreateUser") || strings.HasSuffix(info.FullMethod, "/UpdateUser") {
            return handler(ctx, req)
        }
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        vals := md.Get("authorization")
        if len(vals) == 0 {
            return nil, status.Error(codes.Unauthenticated, "authorization required")
        }
        token := vals[0]
        token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
        _, err := tm.Validate(token)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        return handler(ctx, req)
    }
}
