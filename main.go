package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"payment-service/internal/telemetry"

	"go.uber.org/zap"
)

type Payment struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
	Date   string  `json:"date"`
}

var payments []Payment

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
	_, span := telemetry.Tracer().Start(r.Context(), "paymentHandler")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

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

func handleGetPayments(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(payments)
}

func handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	var payment Payment

	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	payment.ID = fmt.Sprintf("pay_%d", time.Now().Unix())
	payment.Date = time.Now().Format(time.RFC3339)
	payment.Status = "pending"

	payments = append(payments, payment)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(payment)
}
