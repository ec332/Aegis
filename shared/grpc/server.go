package grpc

import (
    "context"
    "time"

    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// ServerMetrics provides metrics for gRPC servers
type ServerMetrics struct{}

// UnaryServerInterceptor provides metrics and logging for unary gRPC calls
func UnaryServerInterceptor(logger *zap.Logger, _ *ServerMetrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
        
		// Log request
		logger.Debug("gRPC request received",
			zap.String("method", info.FullMethod),
			zap.Any("request", req))
		
		// Call handler with timer
	var resp interface{}
	var err error
	
        resp, err = handler(ctx, req)
		
		duration := time.Since(start)
		
		if err != nil {
            
			st, _ := status.FromError(err)
			code := st.Code()
			switch code {
			case codes.NotFound, codes.InvalidArgument, codes.FailedPrecondition, codes.AlreadyExists, codes.OutOfRange, codes.Unauthenticated, codes.PermissionDenied:
				logger.Warn("gRPC request failed",
					zap.String("method", info.FullMethod),
					zap.Duration("duration", duration),
					zap.String("code", code.String()),
					zap.String("message", st.Message()),
					zap.Error(err))
			default:
				logger.Error("gRPC request failed",
					zap.String("method", info.FullMethod),
					zap.Duration("duration", duration),
					zap.String("code", code.String()),
					zap.String("message", st.Message()),
					zap.Error(err))
			}
		} else {
			logger.Info("gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration))
		}
		
		return resp, err
	}
}

// StreamServerInterceptor provides metrics and logging for streaming gRPC calls
func StreamServerInterceptor(logger *zap.Logger, _ *ServerMetrics) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
        
		// Log request
		logger.Debug("gRPC stream request received",
			zap.String("method", info.FullMethod))
		
		// Call handler
		err := handler(srv, ss)
		
		duration := time.Since(start)
        
        if err != nil {
			st, _ := status.FromError(err)
			code := st.Code()
			switch code {
			case codes.NotFound, codes.InvalidArgument, codes.FailedPrecondition, codes.AlreadyExists, codes.OutOfRange, codes.Unauthenticated, codes.PermissionDenied:
				logger.Warn("gRPC stream request failed",
					zap.String("method", info.FullMethod),
					zap.Duration("duration", duration),
					zap.String("code", code.String()),
					zap.String("message", st.Message()),
					zap.Error(err))
			default:
				logger.Error("gRPC stream request failed",
					zap.String("method", info.FullMethod),
					zap.Duration("duration", duration),
					zap.String("code", code.String()),
					zap.String("message", st.Message()),
					zap.Error(err))
			}
		} else {
			logger.Info("gRPC stream request completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration))
		}
		
		return err
	}
}

// RecoveryInterceptor recovers from panics in gRPC handlers
func RecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC handler panicked",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor recovers from panics in gRPC stream handlers
func StreamRecoveryInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC stream handler panicked",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		
		return handler(srv, ss)
	}
}
