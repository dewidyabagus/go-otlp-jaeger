package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	var service = flag.String("service", "", "Started service name")
	flag.Parse()

	switch *service {
	default:
		log.Fatalln("Unregistered service")

	case "order-service":
		go newHttpService(context.TODO(), "order-service", "1.1.0", ":8080", "/orders", orderHandler)

	case "payment-service":
		go newHttpService(context.TODO(), "payment-service", "1.1.0", ":8081", "/payments", paymentHandler)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	val := <-quit

	fmt.Println("Signal:", val.String())
}

func paymentHandler(tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span1 := tracer.Start(r.Context(), "Process 1")
		time.Sleep(150 * time.Millisecond)
		span1.End()

		_, span2 := tracer.Start(r.Context(), "Process 2")
		time.Sleep(150 * time.Millisecond)
		span2.End()

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Hello, World! I am instrumented payment service!"})
	}
}

func orderHandler(tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span1 := tracer.Start(r.Context(), "Process 1")
		time.Sleep(150 * time.Millisecond)
		span1.End()

		client := &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}
		request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "http://172.17.0.1:8081/payments", nil)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintln(w, "Request payment error #1:", err.Error())
			return
		}

		response, err := client.Do(request)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintln(w, "Request payment error #2:", err.Error())
			return
		}
		defer response.Body.Close()

		contents := map[string]interface{}{}
		json.NewDecoder(response.Body).Decode(&contents)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"response": contents})
	}
}

func newHttpService(ctx context.Context, name, version, host, path string, funcHandler func(trace.Tracer) http.HandlerFunc) {
	otlpResources, err := NewOTLPResources(name, version)
	if err != nil {
		log.Fatalln("Init OTLP Resources:", err.Error())
	}

	otlpExporter, err := SetupOTLPExporter(ctx, otlpResources)
	if err != nil {
		log.Fatalln("Setup OTLP Exporter:", err.Error())
	}
	defer otlpExporter.Shutdown(ctx)

	tracer := otel.GetTracerProvider().Tracer(
		"http-tracer",
		trace.WithInstrumentationVersion("1.0.2"),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	spanOptions := otelhttp.WithSpanOptions(
		trace.WithAttributes(semconv.ServiceName(name), semconv.ServiceInstanceID("svc-id-"+name)),
	)
	handler := otelhttp.NewHandler(funcHandler(tracer), path, spanOptions)
	http.Handle(path, handler)

	fmt.Printf("Starting HTTP service %s listen %s\n", name, host)
	log.Fatalln(http.ListenAndServe(host, nil))
}
