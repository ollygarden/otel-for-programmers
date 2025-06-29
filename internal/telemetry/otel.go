package telemetry

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const Scope = "payment-service"

type Providers struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	LoggerProvider log.LoggerProvider
	Logger         *zap.Logger
	Closer         func(ctx context.Context) error
}

var gProviders *Providers

func Setup(ctx context.Context, version, cfgFile string) (func(context.Context) error, error) {
	providers, err := ProvidersFromConfig(ctx, Scope, version, cfgFile)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(providers.TracerProvider)
	otel.SetMeterProvider(providers.MeterProvider)
	global.SetLoggerProvider(providers.LoggerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	gProviders = providers
	return providers.Closer, nil
}

func Logger() *zap.Logger {
	if gProviders == nil {
		logger := zap.Must(zap.NewDevelopment())
		logger.Info("No telemetry providers found, using development logger")
		return logger
	}
	return gProviders.Logger
}

func Meter() metric.Meter {
	if gProviders == nil {
		return otel.Meter(Scope)
	}
	return gProviders.MeterProvider.Meter(Scope)
}

func Tracer() trace.Tracer {
	if gProviders == nil {
		return otel.Tracer(Scope)
	}
	return gProviders.TracerProvider.Tracer(Scope)
}

func ProvidersFromConfig(ctx context.Context, scope, version, cfgFile string) (*Providers, error) {
	b, err := os.ReadFile(cfgFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Providers{
				TracerProvider: nil,
				MeterProvider:  nil,
				Logger:         zap.Must(zap.NewProduction()),
				Closer:         func(ctx context.Context) error { return nil },
			}, nil
		}

		return nil, err
	}

	b = []byte(os.ExpandEnv(string(b)))

	conf, err := otelconf.ParseYAML(b)
	if err != nil {
		return nil, err
	}

	sdk, err := otelconf.NewSDK(otelconf.WithContext(ctx), otelconf.WithOpenTelemetryConfiguration(*conf))
	if err != nil {
		return nil, err
	}

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore(scope, otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	)

	return &Providers{
		TracerProvider: sdk.TracerProvider(),
		MeterProvider:  sdk.MeterProvider(),
		LoggerProvider: sdk.LoggerProvider(),
		Logger:         zap.New(core),
		Closer:         sdk.Shutdown,
	}, nil
}
