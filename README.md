# OTel for Programmers

This is a demonstration project used with our "OTel for Programmers" presentation.

## Overview

A simple Go payment service that demonstrates how to implement OpenTelemetry instrumentation for observability. The service provides basic payment processing capabilities through a REST API.

## Features

- Payment creation and retrieval
- RESTful API endpoints
- In-memory storage (for demo purposes)
- Ready for OpenTelemetry instrumentation

## API Endpoints

- `GET /api/payment` - Retrieve all payments
- `POST /api/payment` - Create a new payment

### Payment Structure

```json
{
  "id": "pay_1234567890",
  "amount": 100.50,
  "status": "pending",
  "date": "2025-07-03T10:30:00Z"
}
```

## Running the Service

```bash
go run main.go
```

The service will start on port 8080.

## Testing the API

Create a payment:
```bash
curl -X POST http://localhost:8080/api/payment \
  -H "Content-Type: application/json" \
  -d '{"amount": 100.50}'
```

Get all payments:
```bash
curl http://localhost:8080/api/payment
```

## About the Presentation

This project serves as the foundation for demonstrating OpenTelemetry concepts including:
- Distributed tracing
- Metrics collection
- Observability best practices
- Integration with monitoring systems
