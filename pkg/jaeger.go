package pkg

import (
	"context"
	"errors"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

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
