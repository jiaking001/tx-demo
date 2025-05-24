package service

import (
	"context"
	"io/ioutil"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"tx-demo/system/proto"
)

// mockSystemServiceSendFileServer 是 SystemService_SendFileServer 的模拟实现
type mockSystemServiceSendFileServer struct {
	grpc.ServerStream
	sentChunks []*system.FileChunk
	err        error
}

func (m *mockSystemServiceSendFileServer) Send(chunk *system.FileChunk) error {
	if m.err != nil {
		return m.err
	}
	m.sentChunks = append(m.sentChunks, chunk)
	return nil
}

// GetSentBytes 用于获取发送的所有字节
func (m *mockSystemServiceSendFileServer) GetSentBytes() []byte {
	var allBytes []byte
	for _, chunk := range m.sentChunks {
		allBytes = append(allBytes, chunk.Data...)
	}
	return allBytes
}

func TestSystemServiceServer_SendFile(t *testing.T) {
	// 创建一个测试用的 zap 日志记录器
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()
	type fields struct {
		UnimplementedSystemServiceServer system.UnimplementedSystemServiceServer
		logger                           *zap.Logger
	}
	type args struct {
		req    *system.SendFileRequest
		stream system.SystemService_SendFileServer
		ctx    context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "File exists and sent successfully",
			fields: fields{
				logger: logger,
			},
			args: args{
				ctx: context.Background(),
				req: &system.SendFileRequest{
					FilePath: "../../resources/001.jpg",
				},
				stream: &mockSystemServiceSendFileServer{},
			},
			wantErr: false,
		},
		{
			name: "File exists and sent successfully",
			fields: fields{
				logger: logger,
			},
			args: args{
				ctx: context.Background(),
				req: &system.SendFileRequest{
					FilePath: "../../resources/002.mp3",
				},
				stream: &mockSystemServiceSendFileServer{},
			},
			wantErr: false,
		},
		{
			name: "File exists and sent successfully",
			fields: fields{
				logger: logger,
			},
			args: args{
				ctx: context.Background(),
				req: &system.SendFileRequest{
					FilePath: "../../resources/003.mp4",
				},
				stream: &mockSystemServiceSendFileServer{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SystemServiceServer{
				UnimplementedSystemServiceServer: tt.fields.UnimplementedSystemServiceServer,
				logger:                           tt.fields.logger,
			}
			if err := s.SendFile(tt.args.ctx, tt.args.req, tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("SendFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if err == nil {
					t.Errorf("SendFile() expected error, but got nil")
				}
				if status.Code(err) == codes.OK {
					t.Errorf("SendFile() expected non-OK status code, but got OK")
				}
			}

			// 读取原始文件内容
			fileContent, err := ioutil.ReadFile(tt.args.req.FilePath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// 获取发送的字节流
			mockStream := tt.args.stream.(*mockSystemServiceSendFileServer)
			sentBytes := mockStream.GetSentBytes()

			// 比较发送的字节流和原始文件的内容
			if string(sentBytes) != string(fileContent) {
				t.Errorf("SendFile() sent bytes do not match file content")
			}
		})
	}
}
