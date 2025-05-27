package main

import (
	"context"
	"errors"
	"github.com/spf13/viper"
	"net"
	"net/http"
	_ "net/http/pprof"
	"tx-demo/pkg"
	"tx-demo/repository"
	systemService "tx-demo/system/service"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	user "tx-demo/user/proto"
	userService "tx-demo/user/service"

	"github.com/opentracing/opentracing-go"
)

func main() {
	// 加载环境变量
	// _ = godotenv.Load("./.env")

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

			pkg.NewViper,
			pkg.NewJwt,
			NewGRPCServer,
			NewConfig,
			pkg.NewLogger,
			pkg.NewJaegerTracer,
		),
		fx.Invoke(StartServer, StartPprofServer), // 调用 StartPprofServer 启动 pprof 服务器
	).Run()
}

type Config struct {
	GRPCPort  string
	PprofPort string
}

func NewConfig(conf *viper.Viper) *Config {
	return &Config{
		GRPCPort:  ":" + conf.GetString("http.port"),
		PprofPort: ":" + conf.GetString("pprof.port"),
	}
}

func NewGRPCServer(logger *zap.Logger, userSvc userService.UserServiceServer, tracer opentracing.Tracer) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(pkg.JaegerServerInterceptor(tracer)),
	)
	user.RegisterUserServiceServer(server, &userSvc)
	logger.Info("gRPC server created")
	return server
}

// StartServer 启动 gRPC 服务器
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

func StartPprofServer(lc fx.Lifecycle, config *Config, logger *zap.Logger) {
	server := &http.Server{Addr: config.PprofPort}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("starting pprof server", zap.String("port", config.PprofPort))
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("failed to start pprof server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping pprof server")
			return server.Shutdown(ctx)
		},
	})
}
