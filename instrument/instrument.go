package instrument

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/telemetry"
	"github.com/opentracing/opentracing-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const (
	otelContextKey = "otel-ctx"
	userContextKey = "user-context"
)

var (
	log = golog.LoggerFor("autoupdate-server.instrument")
	//Tracer = trace.NewNoopTracerProvider().Tracer("noop") // no op by default (for tests, dev)
	Tracer = otel.Tracer("autoupdate-server")
)

func NewOTELMiddleware() (func(next http.Handler) http.Handler, func() error) {
	ctx := context.Background()
	log.Debug("Enabling OpenTelemetry trace exporting")
	stopTracing := telemetry.EnableOTELTracingWithSampleRate(ctx, 1)

	Tracer = otel.Tracer("autoupdate-server")

	// Start the regular tracer and return it as an opentracing.Tracer interface. You
	// may use the same set of options as you normally would with the Datadog tracer.
	t := opentracer.New(tracer.WithServiceName("autoupdate-server"))

	// Set the global OpenTracing tracer.
	opentracing.SetGlobalTracer(t)

	stop := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer func() {
			cancel()
			tracer.Stop()
		}()
		return stopTracing(ctx)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := r.Context()
			traceID, err := trace.TraceIDFromHex(Value(c, "X-Lantern-Trace"))
			// we only want to trace things that are part of an existing flow
			if err != nil || !traceID.IsValid() {
				next.ServeHTTP(w, r)
				return
			}

			spanOptions := []trace.SpanStartOption{
				trace.WithAttributes(attribute.String("request.id", Value(c, "requestid"))),
				trace.WithSpanKind(trace.SpanKindServer),
			}

			// if we know userId, attach it to the span
			if userID := Value(c, "X-Lantern-User-Id"); userID != "" {
				spanOptions = append(spanOptions, trace.WithAttributes(semconv.EnduserIDKey.String(userID)))
			}

			sc := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				TraceFlags: 0,
			})

			ctx := trace.ContextWithRemoteSpanContext(UserContext(c), sc)

			otelCtx, span := Tracer.Start(
				ctx,
				fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				spanOptions...,
			)
			context.WithValue(c, otelContextKey, otelCtx)
			defer span.End()

			statusCode, _ := strconv.Atoi(w.Header().Get("Status"))
			attrs := semconv.HTTPAttributesFromHTTPStatusCode(statusCode)
			spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(statusCode)
			span.SetAttributes(attrs...)
			span.SetStatus(spanStatus, spanMessage)
			if err != nil {
				span.RecordError(err)
			}
			next.ServeHTTP(w, r)
		})
	}, stop
}
