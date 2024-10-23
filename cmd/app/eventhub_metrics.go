package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/deviceinsight/eventhub-metrics/internal/blobstorage"
	"github.com/deviceinsight/eventhub-metrics/internal/collector"
	"github.com/deviceinsight/eventhub-metrics/internal/concurrency"
	"github.com/deviceinsight/eventhub-metrics/internal/config"
	"github.com/deviceinsight/eventhub-metrics/internal/metrics"
)

var Version string
var BuildTime string
var GitCommit string

func main() {

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load cfg", "error", err)
		os.Exit(1)
	}

	config.InitLogger(cfg.Log)

	slog.Info("service starting", "version", Version, "buildTime", BuildTime, "gitCommit", GitCommit)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		slog.Error("failed to get default azure credential", "error", err)
		os.Exit(1)
	}

	var metricExporters []metrics.RecordService

	if cfg.Exporter.Prometheus.Enabled {
		exporter := metrics.NewPrometheusService()

		go exporter.RunHTTPServer(cfg.Exporter.Prometheus.Address, cfg.Exporter.Prometheus.ReadTimeout)

		metricExporters = append(metricExporters, exporter)
	}

	metricsService := metrics.NewDelegateService(metricExporters...)
	collectorService := collector.NewService(metricsService, cfg.Collector)

	for {
		start := time.Now()
		slog.Info("starting metrics collector")
		collectMetrics(credential, cfg, collectorService)
		elapsed := time.Since(start)

		slog.Info("metrics collector finished", "elapsed", elapsed.String())
		if cfg.Collector.Interval == nil {
			break
		}
		slog.Debug("waiting for next iteration", "interval", cfg.Collector.Interval.String)
		time.Sleep(*cfg.Collector.Interval)
	}
}

func collectMetrics(credential *azidentity.DefaultAzureCredential, cfg *config.Config,
	collectorService collector.Service) {

	for _, namespaceCfg := range cfg.Namespaces {

		ctx := context.Background()
		namespace, eventHubs, err := collectorService.ProcessNamespace(ctx, credential, namespaceCfg.Endpoint)
		if err != nil {
			slog.Error("failed process namespace", "namespace", namespace, "error", err)
			os.Exit(1)
		}

		blobStore, err := blobstorage.GetBlobStore(credential, namespaceCfg.StorageAccountEndpoint,
			namespaceCfg.CheckpointContainer)
		if err != nil {
			slog.Error("failed to get blob store", "namespace", namespace, "error", err)
			os.Exit(1)
		}

		limiter := concurrency.NewLimiter(cfg.Collector.Concurrency)
		slog.Debug("using concurrency limit", "concurrency", cfg.Collector.Concurrency)

		for _, eventHub := range eventHubs {
			started := limiter.Go(ctx, func() {
				err := collectorService.ProcessEventHub(ctx, credential, blobStore, namespace, namespaceCfg.Endpoint,
					&eventHub)
				if err != nil {
					slog.Error("failed to process eventhub", "namespace", namespace, "eventHub",
						eventHub, "error", err)
					os.Exit(1)
				}
			})
			if !started && ctx.Err() != nil {
				slog.Error("failed to start process", "namespace", namespace, "eventHub", eventHub,
					"error", err)
				os.Exit(1)
			}
		}

		limiter.Wait()
	}
}
