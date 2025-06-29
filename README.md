# Getting Started with OpenTelemetry

This is a demonstration project created for the CloudLand presentation on **July 3, 2025**.

## Overview

A simple Go payment service that demonstrates how to implement OpenTelemetry instrumentation for observability. The service provides basic payment processing capabilities through a REST API.

## Features

- Payment creation and retrieval
- RESTful API endpoints
- In-memory storage (for demo purposes)
- OpenTelemetry instrumentation with traces, metrics, and logs

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

## OpenTelemetry Configuration

This service sends telemetry data (traces, metrics, and logs) to an OTLP endpoint on `localhost:4318`. 

### Visualizing Telemetry Data

To easily ingest and visualize the telemetry data, you can use the Grafana OTEL LGTM Docker image:

```bash
docker run --name og-lgtm -p 3000:3000 -p 4318:4318 --rm -d grafana/otel-lgtm
```

This will start:
- Grafana on port 3000 (admin/admin)
- OTLP receiver on port 4318

Once running, start the payment service and make some API calls to generate telemetry data that you can explore in Grafana.

## About the Presentation

This project serves as the foundation for demonstrating OpenTelemetry concepts including:
- Distributed tracing
- Metrics collection
- Observability best practices
- Integration with monitoring systems

---

**CloudLand 2025 - Getting Started with OpenTelemetry**