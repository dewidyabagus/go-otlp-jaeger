package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "go-simple-http"
const serviceVersion = "1.0.1"
const defAddress = ":8080"

const otelCollectorEndpoint = "172.17.0.1:4318"
const otelCollectorPath = "/v1/traces"

var tracer trace.Tracer

func main() {
	otlpResources, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			// attribute.String("environment", "development"),
		),
	)
	if err != nil {
		log.Fatalln("Init Resources OTLP:", err.Error())
	}

	otlpHttpOptions := []otlptracehttp.Option{
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(otelCollectorEndpoint),
		otlptracehttp.WithURLPath(otelCollectorPath),
		otlptracehttp.WithTimeout(30 * time.Second),
	}
	otlpExporter, err := otlptrace.New(context.Background(), otlptracehttp.NewClient(otlpHttpOptions...))
	if err != nil {
		log.Fatalln("Init OTLP Exporter:", err.Error())
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(otlpExporter),
		sdktrace.WithResource(otlpResources),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	defer tracerProvider.Shutdown(context.Background())

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	tracer = otel.GetTracerProvider().Tracer(
		"http-tracer",
		trace.WithInstrumentationVersion("1.0.0"),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	welcomeHandler := otelhttp.NewHandler(welcomeHandlerFunc(), "/welcome")
	http.Handle("/welcome", welcomeHandler)

	fmt.Printf("Listen HTTP server %s \n", defAddress)
	log.Fatalln(http.ListenAndServe(defAddress, nil))
}

func welcomeHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := tracer.Start(r.Context(), "welcome handler")
		defer span.End()

		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello, World! I am instrumented automatically!")
	}
}
