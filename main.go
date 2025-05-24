package main

import (
	"context"
	"errors"
	"github.com/joho/godotenv"
	"net"
	"tx-demo/pkg"
	"tx-demo/repository"
	systemService "tx-demo/system/service"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	user "tx-demo/user/proto"
	userService "tx-demo/user/service"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

func main() {
	// 加载环境变量
	_ = godotenv.Load("./.env")

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			logger, _ := zap.NewDevelopment()
			return &fxevent.ZapLogger{Logger: logger}
		}),
		fx.Provide(
			repository.NewRepository,
			// 数据库
			repository.NewDB,
			// Redis
			repository.NewRedis,
			repository.NewUserRepository,
			repository.NewTransaction,
			userService.NewUserServiceServer,
			systemService.NewSystemServiceServer,

			NewJwt,
			NewGRPCServer,
			NewConfig,
			NewLogger,
			NewJaegerTracer,
		),
		fx.Invoke(StartServer),
	).Run()
}

type Config struct {
	GRPCPort string
}

func NewJwt() *pkg.JWT {
	return &pkg.JWT{
		JwtIssuer: "tx-demo",
		JwtKey:    []byte("tx-demo-key"),
	}
}

func NewConfig() *Config {
	return &Config{
		GRPCPort: ":50051",
	}
}

func NewLogger() (*zap.Logger, error) {
	return zap.NewDevelopment()
}

func NewGRPCServer(logger *zap.Logger, userSvc userService.UserServiceServer, tracer opentracing.Tracer) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(JaegerServerInterceptor(tracer)),
	)
	user.RegisterUserServiceServer(server, &userSvc)
	logger.Info("gRPC server created")
	return server
}

func StartServer(lc fx.Lifecycle, server *grpc.Server, config *Config, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", config.GRPCPort)
			if err != nil {
				logger.Fatal("failed to listen", zap.Error(err))
				return err
			}

			logger.Info("starting gRPC server", zap.String("port", config.GRPCPort))
			go func() {
				if err := server.Serve(listener); err != nil {
					logger.Fatal("failed to serve", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping gRPC server")
			server.GracefulStop()
			return nil
		},
	})
}

func NewJaegerTracer() (opentracing.Tracer, func(), error) {
	cfg := &config.Configuration{
		ServiceName: "tx-demo",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "localhost:6831",
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return tracer, func() { closer.Close() }, nil
}

func JaegerServerInterceptor(tracer opentracing.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		wireContext, err := tracer.Extract(opentracing.HTTPHeaders, metadataReaderWriter{md})
		if err != nil && !errors.Is(err, opentracing.ErrSpanContextNotFound) {
			zap.L().Error("Failed to extract span context", zap.Error(err))
		}
		span := tracer.StartSpan(info.FullMethod, ext.RPCServerOption(wireContext))
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
		return handler(ctx, req)
	}
}

type metadataReaderWriter struct {
	metadata.MD
}

func (w metadataReaderWriter) Set(key, value string) {
	w.MD.Set(key, value)
}

func (w metadataReaderWriter) ForeachKey(handler func(key, value string) error) error {
	for k, vs := range w.MD {
		for _, v := range vs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}
