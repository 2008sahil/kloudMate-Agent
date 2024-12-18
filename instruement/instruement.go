package instrument

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var stopTelemetry chan struct{}

func StartTelemetry() {
	stopTelemetry = make(chan struct{})
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("km-agent"),
		),
	)
	if err != nil {
		fmt.Printf("Failed to create resource: %v\n", err)
		return
	}


	metricExporter, err := otlpmetrichttp.New(ctx,otlpmetrichttp.WithInsecure()) 


	if err != nil {
		fmt.Printf("Failed to create metrics exporter: %v\n", err)
		return
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)
	meter := meterProvider.Meter("km-agent-metrics")

	uptimeCounter, _ := meter.Int64Counter("agent_uptime", metric.WithDescription("Agent uptime in seconds"))
	heartbeatCounter, _ := meter.Int64Counter("agent_heartbeat", metric.WithDescription("Heartbeat count"))


	go func() {
		startTime := time.Now()
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				uptimeCounter.Add(ctx, int64(time.Since(startTime).Seconds()))
				heartbeatCounter.Add(ctx, 1)
				// fmt.Println("Heartbeat and uptime telemetry sent")
			case <-stopTelemetry:
				ticker.Stop()
				return
			}
		}
	}()
}

func StopTelemetry() {
	close(stopTelemetry)
	fmt.Println("Telemetry stopped")
}


