// [Golang gRPC实践 连载四 gRPC认证](https://segmentfault.com/a/1190000007933303)
// [grpc-go](ttps://github.com/grpc/grpc-go)
// gRPC 本身只能设置一个拦截器, 但 go-grpc-middleware 项目可以解决这个问题
package lib

import (
	"context"
	"runtime/debug"
	"time"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	ErrInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

// tokenCredential 自定义认证
// 定义了一个tokenCredential结构，并实现了两个方法GetRequestMetadata和RequireTransportSecurity. 这是gRPC提供的自定义认证方式，每次RPC调用都会传输认证信息
type TokenCredential struct {
	token string
}

func NewTokenCredential(token string) *TokenCredential {
	return &TokenCredential{
		token: token,
	}
}

// GetRequestMetadata：获取当前请求认证所需的元数据（metadata）
func (c *TokenCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"token": c.token,
	}, nil
}

// RequireTransportSecurity：是否需要基于 TLS 认证进行安全传输
func (c *TokenCredential) RequireTransportSecurity() bool {
	return true
}

// Option applies an option value for a config.
type serverTokenOption func(*serverTokenConfig)

type serverTokenConfig struct {
	token string
}

func newServerTokenConfig(opts []serverTokenOption) *serverTokenConfig {
	c := &serverTokenConfig{}

	for _, o := range opts {
		o(c)
	}

	return c
}

func WithServerToken(token string) serverTokenOption {
	return func(c *serverTokenConfig) {
		c.token = token
	}
}

func NewUnaryServerToken(opts ...serverTokenOption) grpc.UnaryServerInterceptor {
	c := newServerTokenConfig(opts)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, ErrMissingMetadata
		}

		// The keys within metadata.MD are normalized to lowercase.
		// See: https://godoc.org/google.golang.org/grpc/metadata#New
		if len(md["token"]) < 1 || md["token"][0] != c.token {
			return nil, ErrInvalidToken
		}

		return handler(ctx, req)
	}
}

func NewStreamServerToken(opts ...serverTokenOption) grpc.StreamServerInterceptor {
	c := newServerTokenConfig(opts)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return ErrMissingMetadata
		}

		// The keys within metadata.MD are normalized to lowercase.
		// See: https://godoc.org/google.golang.org/grpc/metadata#New
		if len(md["token"]) < 1 || md["token"][0] != c.token {
			return ErrInvalidToken
		}

		return handler(srv, ss)
	}
}

// from github.com/grpc-ecosystem/go-grpc-middleware/logging/zap.UnaryServerInterceptor
func UnaryLogServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		resp, err := handler(ctx, req)

		code := grpc.Code(err)
		level := grpc_zap.DefaultCodeToLevel(code)
		md, _ := metadata.FromIncomingContext(ctx)

		n := 6
		if err != nil {
			n += 1
		}

		fs := make([]attribute.KeyValue, 0, n)
		fs = append(fs,
			attribute.String("method", info.FullMethod),
			attribute.Any("req", req),
			attribute.Any("md", md),
			attribute.String("code", code.String()),
			attribute.Any("resp", resp),
			attribute.String("duration", time.Since(startTime).String()),
		)

		// _, spanCtx := otelgrpc.Extract(ctx, &md)
		if err != nil {
			fs = append(fs, attribute.String("error", err.Error()))
		}

		SpanLog(ctx, trace.SpanFromContext(ctx), level, "grpc", fs...)

		return resp, err
	}
}

func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			debug.PrintStack()
			err = status.Errorf(codes.Internal, "Panic err: %v", e)
		}
	}()

	return handler(ctx, req)
}
