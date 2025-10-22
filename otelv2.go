// https://opentelemetry.io/docs/languages/go/getting-started/
// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/examples/otel-collector/main.go
// https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp
package golib

import (
	"context"
	"errors"
	"fmt"

	"github.com/meilihao/golib/v2/log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Initialize a gRPC connection to be used by both the tracer and meter
// providers.
func initConn(endpoint string) (*grpc.ClientConn, error) {
	// It connects the OpenTelemetry Collector through local gRPC connection.
	conn, err := grpc.NewClient(endpoint,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	return conn, err
}

func SetupOTelSDK(ctx context.Context, endpoint, serviceName string, resourceAttributes ...attribute.KeyValue) (func(ctx context.Context) error, error) {
	if endpoint == "" {
		log.Glog.Info("otel disabled")
		return func(ctx context.Context) error { return nil }, nil
	}
	log.Glog.Info("otel enabled", zap.String("server", endpoint), zap.String("service", serviceName))

	conn, err := initConn(endpoint)
	if err != nil {
		return nil, err
	}

	resAttributes := []attribute.KeyValue{semconv.ServiceName(serviceName)}
	if len(resourceAttributes) > 0 {
		resAttributes = append(resAttributes, resourceAttributes...)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// The service name used to display traces in backends
			resAttributes...,
		),
	)
	if err != nil {
		return nil, err
	}

	var shutdownFuncs []func(context.Context) error
	shutdownFn := func(ctx context.Context) error {
		var err error

		log.Glog.Info("otel start to call shutdown")
		for _, fn := range shutdownFuncs {
			if errFn := fn(ctx); errFn != nil {
				log.Glog.Error("failed to shutdown otelProvider", zap.Error(err))
				err = errors.Join(err, errFn)
			}
		}
		log.Glog.Info("otel call shutdown done")

		shutdownFuncs = nil
		return err
	}
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdownFn(ctx))
	}

	shutdownTracerProvider, err := initTracerProvider(ctx, res, conn)
	if err != nil {
		handleErr(err)
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, shutdownTracerProvider)

	shutdownMeterProvider, err := initMeterProvider(ctx, res, conn)
	if err != nil {
		handleErr(err)
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, shutdownMeterProvider)

	shutdownLogProvider, err := initLogProvider(ctx, res, conn)
	if err != nil {
		handleErr(err)
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, shutdownLogProvider)

	log.Glog.Info("init otel done")

	return shutdownFn, nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// Initializes an OTLP exporter, and configures the corresponding trace provider.
func initTracerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (func(context.Context) error, error) {
	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 全量采样（开发环境用）
		sdktrace.WithResource(res),                    // 关联服务资源
		sdktrace.WithBatcher(traceExporter),           // 批量发送追踪数据（提高性能）
	)
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (the default is no-op).
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

// Initializes an OTLP exporter, and configures the corresponding meter provider.
func initMeterProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (func(context.Context) error, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}

func initLogProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (func(context.Context) error, error) {
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	processor := sdklog.NewBatchProcessor(logExporter)
	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(processor),
	)

	global.SetLoggerProvider(logProvider)

	return logProvider.Shutdown, nil
}

// --- copy from old otel.go
type (
	LoggerKey struct{}
)

var (
	_spanLogger *zap.Logger
)

func SpanLog(ctx context.Context, span trace.Span, l zapcore.Level, msg string, kv ...attribute.KeyValue) {
	//var logger *zap.Logger
	// if tmp := ctx.Value(LoggerKey{}); tmp == nil { // 不使用该方式, 因为代码实现不美观且会导致在gin handler context中注入LoggerKey{}前的gin middleware DebugReq()无法使用SpanLog()
	if l == zapcore.ErrorLevel {
		span.SetAttributes(attribute.Bool("error", true)) // error mark
	}
	if _spanLogger == nil {
		span.AddEvent(msg, trace.WithAttributes(kv...))

		return
	}

	if ce := _spanLogger.Check(l, msg); ce != nil {
		sctx := span.SpanContext()

		var fs []zap.Field
		if sctx.IsValid() {
			fs = make([]zap.Field, 0, len(kv)+2)
			fs = append(fs, zap.String("trace_id", sctx.TraceID().String()))
			fs = append(fs, zap.String("span_id", sctx.SpanID().String()))
		} else {
			fs = make([]zap.Field, 0, len(kv))
		}

		if len(kv) > 0 {
			for _, attr := range kv {
				switch attr.Value.Type() {
				case attribute.STRING:
					fs = append(fs, zap.String(string(attr.Key), attr.Value.AsString()))
				case attribute.INT64:
					fs = append(fs, zap.Int64(string(attr.Key), attr.Value.AsInt64()))
				case attribute.BOOL:
					fs = append(fs, zap.Bool(string(attr.Key), attr.Value.AsBool()))
				case attribute.FLOAT64:
					fs = append(fs, zap.Float64(string(attr.Key), attr.Value.AsFloat64()))
				default:
					fs = append(fs, zap.Any(string(attr.Key), attr.Value))
				}
			}
		}

		ce.Write(fs...)

		kv = append(kv, attribute.String("level", l.String()))
		span.AddEvent(msg, trace.WithAttributes(kv...))
	}
}
