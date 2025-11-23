package grpc

import (
	"context"
	"time"

	"github.com/aegis/shared/metrics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ServerMetrics provides metrics for gRPC servers
type ServerMetrics struct {
	registry         *metrics.Registry
	requestCounter   *metrics.Counter
	errorCounter     *metrics.Counter
	requestTimer     *metrics.Timer
}

func NewServerMetrics(serviceName string, registry *metrics.Registry) *ServerMetrics {
	return &ServerMetrics{
		registry:       registry,
		requestCounter: registry.Counter(serviceName + "_server_requests_total"),
		errorCounter:   registry.Counter(serviceName + "_server_errors_total"),
		requestTimer:   registry.Timer(serviceName + "_server_request_duration"),
	}
}

// UnaryServerInterceptor provides metrics and logging for unary gRPC calls
func UnaryServerInterceptor(logger *zap.Logger, metrics *ServerMetrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Record request
		metrics.requestCounter.Inc()
		
		// Log request
		logger.Debug("gRPC request received",
			zap.String("method", info.FullMethod),
			zap.Any("request", req))
		
		// Call handler with timer
	var resp interface{}
	var err error
	
	_ = metrics.requestTimer.TimeFunc(func() error {
		resp, err = handler(ctx, req)
		return err
	})
		
		duration := time.Since(start)
		
		if err != nil {
			metrics.errorCounter.Inc()
			
			st, _ := status.FromError(err)
			logger.Error("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", st.Code().String()),
				zap.String("message", st.Message()),
				zap.Error(err))
		} else {
			logger.Info("gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration))
		}
		
		return resp, err
	}
}

// StreamServerInterceptor provides metrics and logging for streaming gRPC calls
func StreamServerInterceptor(logger *zap.Logger, metrics *ServerMetrics) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		// Record request
		metrics.requestCounter.Inc()
		
		// Log request
		logger.Debug("gRPC stream request received",
			zap.String("method", info.FullMethod))
		
		// Call handler
		err := handler(srv, ss)
		
		duration := time.Since(start)
		
		if err != nil {
			metrics.errorCounter.Inc()
			
			st, _ := status.FromError(err)
			logger.Error("gRPC stream request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", st.Code().String()),
				zap.String("message", st.Message()),
				zap.Error(err))
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