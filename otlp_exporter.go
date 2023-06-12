package main

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const defOtelCollectorEndpoint = "172.17.0.1:4318"
const defOtelCollectorPath = "/v1/traces"
const defOtelPushTimeout = 30 * time.Second

func NewOTLPResources(name, version string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
			// attribute.String("environment", "development"),
		),
	)
}

func SetupOTLPExporter(ctx context.Context, resource *resource.Resource) (*sdktrace.TracerProvider, error) {
	otlpHttpOptions := []otlptracehttp.Option{
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(defOtelCollectorEndpoint),
		otlptracehttp.WithURLPath(defOtelCollectorPath),
		otlptracehttp.WithTimeout(defOtelPushTimeout),
	}
	otlpExporter, err := otlptrace.New(ctx, otlptracehttp.NewClient(otlpHttpOptions...))
	if err != nil {
		return nil, err
	}

	otlpTraceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(otlpExporter),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTracerProvider(otlpTraceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return otlpTraceProvider, nil
}
