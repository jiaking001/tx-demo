package service

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"tx-demo/pkg"
	"tx-demo/system/proto"
)

type SystemServiceServer struct {
	system.UnimplementedSystemServiceServer
	logger      *zap.Logger
	opentracing opentracing.Tracer
}

func NewSystemServiceServer(logger *zap.Logger, opentracing opentracing.Tracer) SystemServiceServer {
	return SystemServiceServer{
		logger:      logger,
		opentracing: opentracing,
	}
}

// SendFile 读取文件（以流的形式返回）
func (s SystemServiceServer) SendFile(ctx context.Context, req *system.SendFileRequest, stream system.SystemService_SendFileServer) error {
	s.logger.Info("SendFile called", zap.String("file_path", req.FilePath))

	// 使用jeager实现链路追踪
	span, ctx := opentracing.StartSpanFromContext(ctx, "SystemService.SendFile")
	span.SetTag("file_path", req.FilePath)
	defer span.Finish()

	// 打开文件
	file, err := os.Open(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return status.Errorf(codes.NotFound, pkg.ErrFileNotFound)
		}
		s.logger.Error("Failed to open file", zap.Error(err))
		return status.Errorf(codes.Internal, pkg.ErrInternalServerError)
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		s.logger.Error("Failed to get file info", zap.Error(err))
		return status.Errorf(codes.Internal, pkg.ErrInternalServerError)
	}
	fileSize := fileInfo.Size()

	// 缓冲区大小
	bufferSize := 2 * 1024 * 1024 // 2MB
	buffer := make([]byte, bufferSize)
	offset := int64(0)

	for {
		// 读取文件块
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			s.logger.Error("Failed to read file", zap.Error(err))
			return status.Errorf(codes.Internal, pkg.ErrInternalServerError)
		}

		// 发送文件块
		chunk := &system.FileChunk{
			Data:      buffer[:n],
			Offset:    offset,
			TotalSize: fileSize,
		}
		if err := stream.Send(chunk); err != nil {
			s.logger.Error("Failed to send file chunk", zap.Error(err))
			return status.Errorf(codes.Internal, pkg.ErrInternalServerError)
		}

		offset += int64(n)
	}

	s.logger.Info("File sent successfully", zap.String("file_path", req.FilePath))

	return nil
}
