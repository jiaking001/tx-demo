package main

import (
	"context"
	"github.com/joho/godotenv"
	"net"
	"tx-demo/pkg"
	"tx-demo/repository"
	systemService "tx-demo/system/service"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	user "tx-demo/user/proto"
	userService "tx-demo/user/service"
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

func NewGRPCServer(logger *zap.Logger, userSvc userService.UserServiceServer) *grpc.Server {
	server := grpc.NewServer()
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
