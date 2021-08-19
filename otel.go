// trace from https://github.com/open-telemetry/opentelemetry-go/blob/master/example/otel-collector/main.go
// metric from https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/otlp/otlpmetric/otlpmetricgrpc/example_test.go
// see [opentelemetry-java/QUICKSTART.md](https://github.com/open-telemetry/opentelemetry-java/blob/master/QUICKSTART.md)
// [Documentation / Go / Getting Started](https://opentelemetry.io/docs/go/getting-started/)
package lib

import (
	"context"
	"time"

	"github.com/meilihao/golib/v1/log"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

type (
	LoggerKey struct{}
)

var (
	_spanLogger *zap.Logger
)

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
// endpoint != "" && otel collector isn't started, InitOTEL will hung
func InitOTEL(endpoint, serviceName string, logger, spanLogger *zap.Logger) (func(), error) {
	_spanLogger = spanLogger
	if endpoint == "" {
		log.Glog.Info("trace status", zap.Bool("status", false))
		return func() {}, nil
	}
	log.Glog.Info("trace status", zap.String("server", endpoint))

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create resource")
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns
	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()), // useful for testing
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create trace exporter")
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global TracerProvider (the default is noopTracerProvider).
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// config and start metric exporter
	exp, err := otlpmetric.New(ctx, otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metric exporter")
	}
	pusher := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		controller.WithExporter(exp),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(pusher.MeterProvider())
	if err = pusher.Start(context.Background()); err != nil {
		return nil, errors.Wrap(err, "failed to start metric controller")
	}

	log.Glog.Info("init otel done")

	return func() {
		// Shutdown will flush any remaining spans and shut down the exporter.
		if err := tracerProvider.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown TracerProvider", zap.Error(err))
		}

		// Push any last metric events to the exporter.
		if err := pusher.Stop(context.Background()); err != nil {
			logger.Error("failed to stop metric controller", zap.Error(err))
		}
	}, nil
}

func SpanLog(ctx context.Context, span trace.Span, l zapcore.Level, msg string, kv ...attribute.KeyValue) {
	//var logger *zap.Logger
	// if tmp := ctx.Value(LoggerKey{}); tmp == nil { // 不使用该方式, 因为代码实现不美观且会导致在gin handler context中注入LoggerKey{}前的gin middleware DebugReq()无法使用SpanLog()
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
