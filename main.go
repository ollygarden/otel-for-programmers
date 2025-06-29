package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Payment struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
	Date   string  `json:"date"`
}

var payments []Payment

func main() {
	http.HandleFunc("/api/payment", paymentHandler)
	
	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
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