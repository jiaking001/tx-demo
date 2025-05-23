package service

import (
	"context"
	"errors"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"os"
	"tx-demo/pkg"
	"tx-demo/repository"

	"tx-demo/model"
	pb "tx-demo/user/proto"
)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	logger      *zap.Logger
	jwt         *pkg.JWT
	userRepo    repository.UserRepository
	opentracing opentracing.Tracer
}

func NewUserServiceServer(logger *zap.Logger, jwt *pkg.JWT, userRepo repository.UserRepository, opentracing opentracing.Tracer) UserServiceServer {
	return UserServiceServer{
		logger:      logger,
		jwt:         jwt,
		userRepo:    userRepo,
		opentracing: opentracing,
	}
}

// Register 用户注册（幂等）
func (s UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.logger.Info("Register called", zap.String("username", req.Username))

	_, err := s.userRepo.FindByUsername(req.Username)
	// 检查用户名是否已存在（幂等）
	if err == nil {
		// 如果用户名已存在，则返回错误
		return nil, status.Errorf(codes.AlreadyExists, "用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果查询过程中发生其他错误，则记录日志并返回内部错误
		s.logger.Error("Failed to check username existence", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to process request")
	}

	// 如果用户名不存在，创建新用户
	userID := pkg.GenerateUUID()
	// 加密
	hashedPassword := pkg.HashPassword(req.Password)
	// 将喜好嵌入向量
	likeEmbedding, err := pkg.NewClient(os.Getenv("DASHSCOPE_API_KEY")).GetEmbeddings(req.Like, "text-embedding-v3", "1024")
	if err != nil {
		// 如果嵌入过程中发生错误，则记录日志并返回内部错误
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// 使用jeager实现链路追踪
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserService.Register")
	span.SetTag("userId", userID)
	defer span.Finish()

	newUser := &model.User{
		UserID:        userID,
		Username:      req.Username,
		Password:      hashedPassword,
		Like:          req.Like,
		LikeEmbedding: pkg.ConvertToPGVector(likeEmbedding.Data[0].Embedding),
	}

	err = s.userRepo.CreateUser(newUser)
	if err != nil {
		// 如果创建过程中发生其他错误，则记录日志并返回内部错误
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to create user")
	}

	s.logger.Info("User registered successfully", zap.String("user_id", userID))

	return &pb.RegisterResponse{
		UserId:  userID,
		Message: "注册成功！",
	}, nil
}

// Login 用户登录
func (s UserServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	s.logger.Info("Login called", zap.String("username", req.Username))

	var user *model.User

	// 查询用户是否存在
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "用户不存在")
		}
		// 如果查询过程中发生其他错误，则记录日志并返回内部错误
		s.logger.Error("Failed to query user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to process request")
	}

	// 使用jeager实现链路追踪
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserService.Login")
	span.SetTag("userId", user.UserID)
	defer span.Finish()

	// 验证密码
	if pkg.HashPassword(req.Password) != user.Password {
		return nil, status.Errorf(codes.Unauthenticated, "密码错误")
	}

	// 生成JWT令牌
	token, expiresIn, err := pkg.GenerateJWT(user.UserID, *s.jwt)
	if err != nil {
		// 如果生成过程中发生错误，则记录日志并返回内部错误
		s.logger.Error("Failed to generate token", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to generate token")
	}

	return &pb.LoginResponse{
		AccessToken: token,
		ExpiresIn:   expiresIn,
	}, nil
}

// GetUserInfo 获取用户信息
func (s UserServiceServer) GetUserInfo(ctx context.Context, req *emptypb.Empty) (*pb.UserInfoResponse, error) {
	s.logger.Info("GetUserInfo called")

	// 从上下文获取用户ID（通过认证中间件注入）
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User not authenticated")
	}

	// 使用jeager实现链路追踪
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserService.GetUserInfo")
	span.SetTag("userId", userID)
	defer span.Finish()

	var user *model.User

	user, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "用户不存在")
		}
		// 如果查询过程中发生其他错误，则记录日志并返回内部错误
		s.logger.Error("Failed to query user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to process request")
	}

	return &pb.UserInfoResponse{
		UserId:   user.UserID,
		Username: user.Username,
		Like:     user.Like,
		CreateAt: timestamppb.New(user.CreatedAt),
		UpdateAt: timestamppb.New(user.UpdatedAt),
	}, nil
}
