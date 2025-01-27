package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/deviceinsight/eventhub-metrics/internal/concurrency"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/deviceinsight/eventhub-metrics/internal/blobstorage"
	"github.com/deviceinsight/eventhub-metrics/internal/collector"
	"github.com/deviceinsight/eventhub-metrics/internal/config"
	"github.com/deviceinsight/eventhub-metrics/internal/metrics"
)

var Version string
var BuildTime string
var GitCommit string

func main() {

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		slog.Debug("service terminated")
		os.Exit(1)
	}()

	os.Exit(run())
}

func run() int {

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load cfg", "error", err)
		return 1
	}

	config.InitLogger(cfg.Log)

	slog.Info("service starting", "version", Version, "buildTime", BuildTime, "gitCommit", GitCommit)

	defer func() {
		slog.Debug("service stopping")
	}()

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		slog.Error("failed to get default azure credential", "error", err)
		return 1
	}

	var metricExporters []metrics.RecordService

	if cfg.Exporter.AppInsights.Enabled {
		exporter := metrics.NewAppInsightsService(cfg.Exporter.AppInsights.InstrumentationKey)
		metricExporters = append(metricExporters, exporter)
	}

	if cfg.Exporter.Prometheus.Enabled {
		exporter := metrics.NewPrometheusService()

		go exporter.RunHTTPServer(cfg.Exporter.Prometheus.Address, cfg.Exporter.Prometheus.ReadTimeout)

		metricExporters = append(metricExporters, exporter)
	}

	if cfg.Exporter.PushGateway.Enabled {
		exporter := metrics.NewPushGatewayService(cfg.Exporter.PushGateway.BaseURL)
		metricExporters = append(metricExporters, exporter)
	}

	metricsService := metrics.NewDelegateService(metricExporters...)
	collectorService := collector.NewService(metricsService, cfg.Collector)

	for {
		start := time.Now()
		slog.Info("starting metrics collector")
		collectMetrics(credential, cfg, collectorService)

		if err := metricsService.PushMetrics(); err != nil {
			slog.Error("failed to push metrics", "error", err)
			return 1
		}

		elapsed := time.Since(start)

		slog.Info("metrics collector finished", "elapsed", elapsed.String())
		if cfg.Collector.Interval == nil {
			break
		}
		slog.Debug("waiting for next iteration", "interval", cfg.Collector.Interval.String())
		time.Sleep(*cfg.Collector.Interval)
	}

	return 0
}

//nolint:gocognit
func collectMetrics(credential *azidentity.DefaultAzureCredential, cfg *config.Config,
	collectorService collector.Service) int {

	ctx := context.Background()
	storedGroups, err := getCheckpointContainerInfos(ctx, credential, cfg.StorageAccounts)
	if err != nil {
		slog.Error("failed to get checkpoint container infos", "error", err)
		return 1
	}

	for _, namespaceCfg := range cfg.Namespaces {

		namespace, eventHubs, err := collectorService.ProcessNamespace(ctx, credential, namespaceCfg.Endpoint)
		if err != nil {
			slog.Error("failed to process namespace", "namespace", namespace, "error", err)
			return 1
		}

		includedEventHubsRegex, err := parseRegex(namespaceCfg.IncludedEventHubs)
		if err != nil {
			slog.Error("failed to compile includedEventHubs regex", "error", err)
			return 1
		}

		excludeEventHubsRegex, err := parseRegex(namespaceCfg.ExcludedEventHubs)
		if err != nil {
			slog.Error("failed to compile excludedEventHubs regex", "error", err)
			return 1
		}

		excludeConsumerGroupsRegex, err := parseRegex(namespaceCfg.ExcludedConsumerGroups)
		if err != nil {
			slog.Error("failed to compile excludedConsumerGroups regex", "error", err)
			return 1
		}

		limiter := concurrency.NewLimiter(cfg.Collector.Concurrency)
		slog.Debug("using concurrency limit", "concurrency", cfg.Collector.Concurrency)

		inProgress := 0
		resultChan := make(chan int)

		for _, eventHub := range eventHubs {

			if includedEventHubsRegex != nil && !includedEventHubsRegex.MatchString(eventHub.Name) {
				slog.Debug("skipping non-included eventhub", "eventhub", eventHub.Name,
					"regex", includedEventHubsRegex.String())
				continue
			}

			if excludeEventHubsRegex != nil && excludeEventHubsRegex.MatchString(eventHub.Name) {
				slog.Debug("skipping excluded eventhub", "eventhub", eventHub.Name,
					"regex", excludeEventHubsRegex.String())
				continue
			}

			blobStores, err := blobstorage.GetBlobStores(credential, storedGroups, namespaceCfg.Endpoint, eventHub.Name)
			if err != nil {
				slog.Error("failed to get blob stores", "namespace", namespace, "error", err)
				return 1
			}

			started := limiter.Go(ctx, func() {
				err := collectorService.ProcessEventHub(ctx, credential, blobStores, namespace, namespaceCfg.Endpoint,
					&eventHub, excludeConsumerGroupsRegex)
				if err != nil {
					slog.Error("failed to process eventhub", "namespace", namespace, "eventHub",
						eventHub, "error", err)
					resultChan <- 1
				}
				resultChan <- 0
			})
			if !started && ctx.Err() != nil {
				slog.Error("failed to start process", "namespace", namespace, "eventHub", eventHub,
					"error", err)
				return 1
			}
			inProgress++
		}

		for i := range inProgress {
			slog.Debug("awaiting result", "index", i, "count", inProgress)
			result := <-resultChan
			if result != 0 {
				return result
			}
		}

		limiter.Wait()
	}

	return 0
}

func getCheckpointContainerInfos(ctx context.Context, credential *azidentity.DefaultAzureCredential,
	storageAccounts []config.BlobStorageConfig) (blobstorage.StoredGroupsMap, error) {

	containerInfos := make(blobstorage.StoredGroupsMap)

	for _, storageAccountCfg := range storageAccounts {

		includedContainersRegex, err := parseRegex(storageAccountCfg.IncludedContainers)
		if err != nil {
			slog.Error("failed to compile includedContainers regex", "error", err)
			return nil, err
		}

		excludedContainersRegex, err := parseRegex(storageAccountCfg.ExcludedContainers)
		if err != nil {
			slog.Error("failed to compile excludedContainers regex", "error", err)
			return nil, err
		}

		infos, err := blobstorage.GetContainerInfos(ctx, credential, storageAccountCfg.Endpoint,
			includedContainersRegex, excludedContainersRegex)
		if err != nil {
			return nil, err
		}

		for storageContainer, storedConsumerGroups := range infos {
			containerInfos[storageContainer] = storedConsumerGroups
		}
	}

	return containerInfos, nil
}

func parseRegex(regexString string) (*regexp.Regexp, error) {
	var regex *regexp.Regexp
	var err error

	if regexString != "" {
		regex, err = regexp.Compile(regexString)
		if err != nil {
			return nil, err
		}
	}

	return regex, nil
}
