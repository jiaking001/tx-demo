package service

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"os"
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
	// 使用jeager实现链路追踪
	span, ctx := opentracing.StartSpanFromContext(ctx, "SystemService.SendFile")
	span.SetTag("file_path", req.FilePath)
	defer span.Finish()

	s.logger.Info("SendFile called", zap.String("file_path", req.FilePath))

	// 打开文件
	file, err := os.Open(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return status.Errorf(codes.NotFound, "文件不存在")
		}
		s.logger.Error("Failed to open file", zap.Error(err))
		return status.Errorf(codes.Internal, "Failed to open file")
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		s.logger.Error("Failed to get file info", zap.Error(err))
		return status.Errorf(codes.Internal, "Failed to get file info")
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
			return status.Errorf(codes.Internal, "Failed to read file")
		}

		// 发送文件块
		chunk := &system.FileChunk{
			Data:      buffer[:n],
			Offset:    offset,
			TotalSize: fileSize,
		}
		if err := stream.Send(chunk); err != nil {
			s.logger.Error("Failed to send file chunk", zap.Error(err))
			return status.Errorf(codes.Internal, "Failed to send file chunk")
		}

		offset += int64(n)
	}

	s.logger.Info("File sent successfully", zap.String("file_path", req.FilePath))
	return nil
}
