// Command emit drives synthetic OpenTelemetry logs, metrics and traces into an
// OTLP/gRPC endpoint so the CI harness has real data to build a report from.
//
// Unlike a raw telemetrygen it deliberately *spreads* records over a time
// window and varies severity, metric value and span duration/status, so the
// generated static report shows an interesting time-series, a severity
// breakdown, a metric curve and a mix of OK/ERROR traces — instead of a single
// flat bar. It uses only the OTel Go SDK, so it builds in seconds.
//
// Usage: emit -endpoint host:port [-logs N] [-traces N] [-window 15m]
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "otelhouseview-emit"

func main() {
	var (
		endpoint = flag.String("endpoint", "127.0.0.1:4317", "OTLP/gRPC endpoint (host:port, no scheme)")
		nLogs    = flag.Int("logs", 300, "number of log records to emit")
		nTraces  = flag.Int("traces", 30, "number of traces to emit")
		window   = flag.Duration("window", 15*time.Minute, "time window to spread backdated records over")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		fail("build resource: %v", err)
	}

	// End the window "now" and spread records backwards, so the report's
	// window ends at report-generation time.
	end := time.Now().UTC()
	start := end.Add(-*window)

	if err := emitLogs(ctx, *endpoint, *nLogs, start, end, res); err != nil {
		fail("emit logs: %v", err)
	}
	if err := emitTraces(ctx, *endpoint, *nTraces, start, end, res); err != nil {
		fail("emit traces: %v", err)
	}
	if err := emitMetrics(ctx, *endpoint, start, end, res); err != nil {
		fail("emit metrics: %v", err)
	}
	fmt.Fprintln(os.Stderr, "emit: done")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "emit: "+format+"\n", args...)
	os.Exit(1)
}

// emitLogs spreads n backdated records across [start,end]. Most are INFO with
// periodic WARN, plus two ERROR bursts, so the severity breakdown and the
// stacked volume chart both have something to show.
func emitLogs(ctx context.Context, endpoint string, n int, start, end time.Time, res *resource.Resource) error {
	exp, err := otlploggrpc.New(ctx, otlploggrpc.WithEndpoint(endpoint), otlploggrpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp, sdklog.WithExportInterval(50*time.Millisecond))),
		sdklog.WithResource(res),
	)
	defer func() { _ = lp.Shutdown(ctx) }()

	logger := lp.Logger(serviceName)
	span := end.Sub(start)
	for i := 0; i < n; i++ {
		frac := float64(i) / float64(max(1, n-1))
		ts := start.Add(time.Duration(frac * float64(span)))

		sev, sevText := otellog.SeverityInfo, "INFO"
		switch {
		// Two ERROR bursts, at ~30% and ~70% through the window.
		case (frac > 0.28 && frac < 0.34) || (frac > 0.68 && frac < 0.73):
			if i%2 == 0 {
				sev, sevText = otellog.SeverityError, "ERROR"
			}
		case i%9 == 0:
			sev, sevText = otellog.SeverityWarn, "WARN"
		}

		var rec otellog.Record
		rec.SetTimestamp(ts)
		rec.SetObservedTimestamp(ts)
		rec.SetSeverity(sev)
		rec.SetSeverityText(sevText)
		rec.SetBody(otellog.StringValue(fmt.Sprintf("request %d handled sev=%s", i, sevText)))
		rec.AddAttributes(otellog.Int("iter", i), otellog.String("component", "api"))
		logger.Emit(ctx, rec)
	}
	return lp.ForceFlush(ctx)
}

// emitTraces emits n traces, each a root "checkout" span with one child. Start
// times are spread over the window and durations vary; roughly one in six is
// marked ERROR so the traces table shows a status mix.
func emitTraces(ctx context.Context, endpoint string, n int, start, end time.Time, res *resource.Resource) error {
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	defer func() { _ = tp.Shutdown(ctx) }()

	tracer := tp.Tracer(serviceName)
	span := end.Sub(start)
	for i := 0; i < n; i++ {
		frac := float64(i) / float64(max(1, n-1))
		startT := start.Add(time.Duration(frac * float64(span)))
		// Duration between ~40ms and ~360ms, shaped by a sine so the bar
		// chart of durations is not monotonic.
		dur := time.Duration((60 + 300*(0.5+0.5*math.Sin(frac*math.Pi*3))) * float64(time.Millisecond))
		childDur := dur - 20*time.Millisecond

		rootCtx, root := tracer.Start(ctx, "checkout", trace.WithTimestamp(startT))
		_, child := tracer.Start(rootCtx, "db.query", trace.WithTimestamp(startT.Add(10*time.Millisecond)))
		child.SetAttributes(attribute.Int("iter", i))
		child.End(trace.WithTimestamp(startT.Add(10*time.Millisecond + childDur)))
		if i%6 == 3 {
			root.SetStatus(codes.Error, "payment declined")
		} else {
			root.SetStatus(codes.Ok, "")
		}
		root.End(trace.WithTimestamp(startT.Add(dur)))
	}
	return tp.ForceFlush(ctx)
}

// emitMetrics records a gauge that follows a sine wave. A PeriodicReader
// exports every 100ms; recording a new value each cycle yields a genuine
// time-series in otel_metrics_gauge (distinct TimeUnix per row).
func emitMetrics(ctx context.Context, endpoint string, start, end time.Time, res *resource.Resource) error {
	exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(endpoint), otlpmetricgrpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("metric exporter: %w", err)
	}
	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(100*time.Millisecond))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithResource(res))
	defer func() { _ = mp.Shutdown(ctx) }()

	meter := mp.Meter(serviceName)
	gauge, err := meter.Float64Gauge("otelhouseview.demo.load", metric.WithUnit("1"))
	if err != nil {
		return fmt.Errorf("create gauge: %w", err)
	}
	const cycles = 16
	for i := 0; i < cycles; i++ {
		frac := float64(i) / float64(cycles-1)
		v := 0.5 + 0.45*math.Sin(frac*math.Pi*2)
		gauge.Record(ctx, v)
		// Let the PeriodicReader export this value before the next one.
		time.Sleep(120 * time.Millisecond)
	}
	return mp.ForceFlush(ctx)
}
