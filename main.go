package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/michgboxy2/carzone/driver"
	carHandler "github.com/michgboxy2/carzone/handler/car"
	engineHandler "github.com/michgboxy2/carzone/handler/engine"
	loginHandler "github.com/michgboxy2/carzone/handler/login"
	middleware "github.com/michgboxy2/carzone/middleware"
	carService "github.com/michgboxy2/carzone/service/car"
	engineService "github.com/michgboxy2/carzone/service/engine"
	carStore "github.com/michgboxy2/carzone/store/car"
	engineStore "github.com/michgboxy2/carzone/store/engine"

	// "github.com/prometheus/client_golang/promhttp"
	"github.com/m3db/prometheus_client_golang/prometheus/promhttp"
	otelmux "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("error loading .env file")
	}

	traceProvider, err := startTracing()

	if err != nil {
		log.Fatalf("failed to start tracing: %v", err)
	}

	defer func() {
		err := traceProvider.Shutdown(context.Background()) // Capture the error from Shutdown
		if err != nil {
			log.Printf("failed to shut down tracing: %v", err)
		}
	}()

	otel.SetTracerProvider(traceProvider)

	driver.InitDB()

	defer driver.CloseDB()

	db := driver.GetDB()

	carStore := carStore.New(db)
	carService := carService.NewCarService(carStore)

	engineStore := engineStore.New(db)
	engineService := engineService.NewEngineService(engineStore)

	carHandler := carHandler.NewCarHandler(carService)
	engineHandler := engineHandler.NewEngineHandler(engineService)

	router := mux.NewRouter()

	router.Use(otelmux.Middleware("carzone"))
	router.Use(middleware.MetricMiddleware)

	schemaFile := "store/schema.sql"

	if err := executeSchemaFile(db, schemaFile); err != nil {
		log.Fatal("Error while executing THE SCHEMA file", err)
	}

	router.HandleFunc("/login", loginHandler.LoginHandler).Methods("POST")

	//Middleware
	protected := router.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware)

	protected.HandleFunc("/car/{id}", carHandler.GetCarById).Methods("GET")
	protected.HandleFunc("/cars/{brand}", carHandler.GetCarByBrand).Methods("GET")
	protected.HandleFunc("/cars", carHandler.CreateCar).Methods("POST")
	protected.HandleFunc("/cars/{id}", carHandler.UpdateCar).Methods("PUT")
	protected.HandleFunc("/cars/{id}", carHandler.DeleteCar).Methods("DELETE")

	protected.HandleFunc("/engine/{id}", engineHandler.GetEngineById).Methods("GET")
	protected.HandleFunc("/engine", engineHandler.CreateEngine).Methods("POST")
	protected.HandleFunc("/engine/{id}", engineHandler.UpdateEngine).Methods("PUT")
	protected.HandleFunc("/engine/{id}", engineHandler.DeleteEngine).Methods("DELETE")

	router.Handle("/metrics", promhttp.Handler())
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))

}

func executeSchemaFile(db *sql.DB, fileName string) error {
	sqlFile, err := os.ReadFile(fileName)

	if err != nil {
		return err
	}

	_, err = db.Exec(string(sqlFile))

	if err != nil {
		return err
	}

	return nil

}

func startTracing() (*trace.TracerProvider, error) {
	header := map[string]string{
		"Content-Type": "application/json",
	}

	expoter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint("jaeger:4318"),
			otlptracehttp.WithHeaders(header),
			otlptracehttp.WithInsecure(),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("error creating new Exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(
			expoter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("carzone"),
			),
		),
	)

	return tracerProvider, nil
}
