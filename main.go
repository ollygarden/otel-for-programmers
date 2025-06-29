package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"payment-service/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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
	requestCounter      metric.Int64Counter
	responseDuration    metric.Float64Histogram
	errorCounter        metric.Int64Counter
	paymentAmount       metric.Float64Histogram
	paymentsByStatus    metric.Int64Counter
	paymentsByCurrency  metric.Int64Counter
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

	http.HandleFunc("/api/payment", paymentHandler)

	logger.Info("Server starting on :8080")
	span.End()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	_, span := telemetry.Tracer().Start(r.Context(), "paymentHandler")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	metrics.requestCounter.Add(r.Context(), 1, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", r.URL.Path),
	))

	var statusCode int
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.responseDuration.Record(r.Context(), duration, metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", r.URL.Path),
			attribute.String("status_code", strconv.Itoa(statusCode)),
		))

		if statusCode >= 400 {
			metrics.errorCounter.Add(r.Context(), 1, metric.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("endpoint", r.URL.Path),
				attribute.String("status_code", strconv.Itoa(statusCode)),
			))
		}
	}()

	switch r.Method {
	case http.MethodGet:
		statusCode = handleGetPayments(w, r)
	case http.MethodPost:
		statusCode = handleCreatePayment(w, r)
	default:
		statusCode = http.StatusMethodNotAllowed
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func handleGetPayments(w http.ResponseWriter, _ *http.Request) int {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payments)
	return http.StatusOK
}

func handleCreatePayment(w http.ResponseWriter, r *http.Request) int {
	var payment Payment

	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return http.StatusBadRequest
	}

	if payment.Currency == "" {
		payment.Currency = "USD"
	}

	payment.ID = fmt.Sprintf("pay_%d", time.Now().Unix())
	payment.Date = time.Now().Format(time.RFC3339)
	payment.Status = "pending"

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
	return http.StatusCreated
}

func initMetrics() error {
	meter := telemetry.Meter()

	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	responseDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	errorCounter, err := meter.Int64Counter(
		"http_errors_total",
		metric.WithDescription("Total number of HTTP errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	paymentAmount, err := meter.Float64Histogram(
		"payment_amount",
		metric.WithDescription("Payment amounts processed"),
		metric.WithUnit("currency_unit"),
	)
	if err != nil {
		return err
	}

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
		requestCounter:      requestCounter,
		responseDuration:    responseDuration,
		errorCounter:        errorCounter,
		paymentAmount:       paymentAmount,
		paymentsByStatus:    paymentsByStatus,
		paymentsByCurrency:  paymentsByCurrency,
	}

	return nil
}
