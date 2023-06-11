package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "go-simple-http"
const serviceVersion = "1.0.1"
const defAddress = ":8080"

var tracer trace.Tracer

func main() {
	otlpResources, err := NewOTLPResources(serviceName, serviceVersion)
	if err != nil {
		log.Fatalln("Init OTLP Resources:", err.Error())
	}

	otlpExporter, err := SetupOTLPExporter(context.Background(), otlpResources)
	if err != nil {
		log.Fatalln("Setup OTLP Exporter:", err.Error())
	}
	defer otlpExporter.Shutdown(context.Background())

	tracer = otel.GetTracerProvider().Tracer(
		"http-tracer",
		trace.WithInstrumentationVersion("1.0.2"),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	welcomeHandler := otelhttp.NewHandler(welcomeHandlerFunc(), "/welcome")
	http.Handle("/welcome", welcomeHandler)

	fmt.Printf("Listen HTTP server %s \n", defAddress)
	log.Fatalln(http.ListenAndServe(defAddress, nil))
}

func welcomeHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span1 := tracer.Start(r.Context(), "Process 1")
		time.Sleep(150 * time.Millisecond)
		span1.End()

		_, span2 := tracer.Start(r.Context(), "Process 2")
		time.Sleep(150 * time.Millisecond)
		span2.End()

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello, World! I am instrumented automatically!")
	}
}
