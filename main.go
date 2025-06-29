package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"payment-service/internal/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Payment struct {
	ID       string  `json:"id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Status   string  `json:"status"`
	Date     string  `json:"date"`
}

var payments []Payment

type Metrics struct {
	paymentAmount      metric.Float64Histogram
	paymentsByStatus   metric.Int64Counter
	paymentsByCurrency metric.Int64Counter
}

var metrics *Metrics

func main() {
	ctx := context.Background()

	closer, err := telemetry.Setup(ctx, "1.0.0", "local/otel.yaml")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := closer(ctx); err != nil {
			telemetry.Logger().Error("Failed to shutdown telemetry", zap.Error(err))
		}
	}()

	if err := initMetrics(); err != nil {
		panic(err)
	}

	ctx, span := telemetry.Tracer().Start(ctx, "run")

	logger := telemetry.Logger()
	logger.Info("Starting payment service")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/payment", paymentHandler)

	// Wrap the mux with otelhttp for automatic HTTP instrumentation
	handler := otelhttp.NewHandler(mux, "payment-service", otelhttp.WithSpanNameFormatter(
		func(_ string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		},
	))

	logger.Info("Server starting on :8080")
	span.End()

	if err := http.ListenAndServe(":8080", handler); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// otelhttp automatically creates spans and records HTTP metrics for us
	// We only need to handle business logic here
	switch r.Method {
	case http.MethodGet:
		handleGetPayments(w, r)
	case http.MethodPost:
		handleCreatePayment(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func handleGetPayments(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payments)
}

func handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	var payment Payment

	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if payment.Currency == "" {
		payment.Currency = "USD"
	}

	payment.ID = fmt.Sprintf("pay_%d", time.Now().Unix())
	payment.Date = time.Now().Format(time.RFC3339)
	payment.Status = "pending"

	// Get the current span from context (created by otelhttp) and add custom attributes
	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(
		attribute.String("payment.currency", payment.Currency),
		attribute.Float64("payment.amount", payment.Amount),
		attribute.String("payment.status", payment.Status),
		attribute.String("payment.id", payment.ID),
	)

	payments = append(payments, payment)

	metrics.paymentAmount.Record(r.Context(), payment.Amount, metric.WithAttributes(
		attribute.String("currency", payment.Currency),
	))

	metrics.paymentsByStatus.Add(r.Context(), 1, metric.WithAttributes(
		attribute.String("status", payment.Status),
	))

	metrics.paymentsByCurrency.Add(r.Context(), 1, metric.WithAttributes(
		attribute.String("currency", payment.Currency),
	))

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(payment)
}

func initMetrics() error {
	meter := telemetry.Meter()

	// Business-specific metrics (HTTP metrics are handled by otelhttp)

	paymentAmount, err := meter.Float64Histogram(
		"payment_amount",
		metric.WithDescription("Payment amounts processed"),
		metric.WithUnit("currency_unit"),
	)
	if err != nil {
		return err
	}

	// advanced: those could actually be the same counter, but different views
	paymentsByStatus, err := meter.Int64Counter(
		"payments_by_status_total",
		metric.WithDescription("Total number of payments by status"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	paymentsByCurrency, err := meter.Int64Counter(
		"payments_by_currency_total",
		metric.WithDescription("Total number of payments by currency"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	metrics = &Metrics{
		paymentAmount:      paymentAmount,
		paymentsByStatus:   paymentsByStatus,
		paymentsByCurrency: paymentsByCurrency,
	}

	return nil
}
