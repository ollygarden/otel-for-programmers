package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type TestPayment struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

var currencies = []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD", "CHF", "CNY", "SEK", "NZD"}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down traffic generator...")
		cancel()
	}()

	baseURL := "http://localhost:8080"
	if len(os.Args) > 1 {
		baseURL = os.Args[1]
	}

	log.Printf("Starting traffic generator against %s", baseURL)
	log.Println("Press Ctrl+C to stop")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	requestCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("Generated %d requests total", requestCount)
			return
		case <-ticker.C:
			go func() {
				if rand.Float32() < 0.8 {
					// 80% POST requests (create payments)
					createPayment(client, baseURL)
				} else {
					// 20% GET requests (list payments)
					getPayments(client, baseURL)
				}
			}()
			requestCount++

			if requestCount%50 == 0 {
				log.Printf("Sent %d requests", requestCount)
			}
		}
	}
}

func createPayment(client *http.Client, baseURL string) {
	payment := TestPayment{
		Amount:   randomAmount(),
		Currency: randomCurrency(),
	}

	jsonData, err := json.Marshal(payment)
	if err != nil {
		log.Printf("Error marshaling payment: %v", err)
		return
	}

	resp, err := client.Post(baseURL+"/api/payment", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating payment: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Payment creation failed with status: %d", resp.StatusCode)
	}
}

func getPayments(client *http.Client, baseURL string) {
	resp, err := client.Get(baseURL + "/api/payment")
	if err != nil {
		log.Printf("Error getting payments: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Get payments failed with status: %d", resp.StatusCode)
	}
}

func randomAmount() float64 {
	// Generate amounts between $1.00 and $999.99
	return float64(rand.Intn(99999)+100) / 100.0
}

func randomCurrency() string {
	return currencies[rand.Intn(len(currencies))]
}