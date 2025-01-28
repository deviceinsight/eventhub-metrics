# Eventhub metrics

Metrics exporter for Eventhubs when using *Event Hubs consumer groups*.

For the difference between *Event Hubs consumer groups* and *Kafka consumer groups* see:
https://learn.microsoft.com/en-us/azure/event-hubs/apache-kafka-frequently-asked-questions#event-hubs-consumer-group-vs--kafka-consumer-group

## 🚀 Features

- **Multiple Eventhub Namespaces:** Multiple Namespaces can be monitored together
- **Partition Owners:** Monitor which instance owns a partition for a consumer group
- **Consumer Group Lags:** Number of messages a consumer group is lagging behind the latest enqueued sequence number
- **Exporters:** Metrics can be exported to Prometheus, AppInsights, PushGateway 
- **Configurable targets:** You can configure what eventhubs or groups you'd like to export using regex expressions
- **Deployment:** The application can be deployed as Kubernetes Deployment or Cron Job or with docker directly.

## Getting started

### 🐳 Docker image

A docker image is available on [docker hub](https://hub.docker.com/repository/docker/deviceinsight/eventhub-metrics):

```shell
docker pull deviceinsight/eventhub-metrics:latest
```

### ☸ Helm chart

A helm chart is maintained as part of this repository.
Install instructions:

```shell
helm repo add eventhub-metrics https://raw.githubusercontent.com/deviceinsight/eventhub-metrics/main/helm/archives
helm repo update
helm install -f values.yaml --name=eventhub-metrics eventhub-metrics/eventhub-metrics
```

### 📊 Grafana Dashboards

The repo contains three separate Grafana dashboards that can be used as inspiration in order to create your own dashboards. Please take note that these dashboards might not immediately work for you due to different labeling in your Prometheus config.

- [Namespace Dashboard](./grafana/namespace_dashboard.json)
- [Consumer Group Dashboard](./grafana/consumer_group_dashboard.json)
- [Eventhub Dashboard](./grafana/eventhub_dashboard.json)

<p float="left">
  <img src="/grafana/screenshots/namespace_dashboard.jpg" width="250" />
  <img src="/grafana/screenshots/consumer_group_dashboard.jpg" width="250" /> 
  <img src="/grafana/screenshots/evenhub_dashboard.jpg" width="250" />
</p>

## Exported Metrics

This section lists all exported metrics in an exemplary way.

### Namespace Metrics

```
# HELP eh_metrics_namespace_info eventhub namespace info
# TYPE eh_metrics_namespace_info gauge
eh_metrics_namespace_info{eh_endpoint="my-eventhub-ns.servicebus.windows.net",eh_namespace="my-eventhub-ns"} 1
```

### Eventhub & Partition Metrics

```
# HELP eh_metrics_eventhub_info eventhub info
# TYPE eh_metrics_eventhub_info gauge
eh_metrics_eventhub_info{eh_namespace="my-eventhub-ns",eventhub="eventhub-1",partition_count="4",retention_in_days="7"} 1

# HELP eh_metrics_eventhub_partition_sequence_min beginning sequence number of a partition
# TYPE eh_metrics_eventhub_partition_sequence_min gauge
eh_metrics_eventhub_partition_sequence_min{eh_namespace="my-eventhub-ns",eventhub="eventhub-1",partition_id="0"} 2.260468e+06

# HELP eh_metrics_eventhub_partition_sequence_max last enqueued sequence number of a partition
# TYPE eh_metrics_eventhub_partition_sequence_max gauge
eh_metrics_eventhub_partition_sequence_max{eh_namespace="my-eventhub-ns",eventhub="eventhub-1",partition_id="0"} 2.273118e+06

# HELP eh_metrics_eventhub_sequence_min_sum sum of all the eventhub's partition beginning sequence numbers
# TYPE eh_metrics_eventhub_sequence_min_sum gauge
eh_metrics_eventhub_sequence_min_sum{eh_namespace="my-eventhub-ns",eventhub="eventhub-1"} 1.3318065e+07

# HELP eh_metrics_eventhub_sequence_max_sum sum of all the eventhub's partition last enqueued sequence numbers
# TYPE eh_metrics_eventhub_sequence_max_sum gauge
eh_metrics_eventhub_sequence_max_sum{eh_namespace="my-eventhub-ns",eventhub="eventhub-1"} 1.3395102e+07
```

### Consumer Group Metrics

```
# HELP eh_metrics_consumer_group_info consumer group info gauges. It will report 1 if the group is in the stable state, otherwise 0.
# TYPE eh_metrics_consumer_group_info gauge
eh_metrics_consumer_group_info{consumer_group="my-group",eh_namespace="my-eventhub-ns",eventhub="eventhub-1",state="stable"} 1

# HELP eh_metrics_consumer_group_owners consumer group owner count gauges. It will report the number of owners in the consumer group
# TYPE eh_metrics_consumer_group_owners gauge
eh_metrics_consumer_group_owners{consumer_group="my-group",eh_namespace="my-eventhub-ns",eventhub="eventhub-1"} 4

# HELP eh_metrics_consumer_group_events_sum the sum of all committed sequence numbers across all partitions in an eventhub
# TYPE eh_metrics_consumer_group_events_sum gauge
eh_metrics_consumer_group_events_sum{consumer_group="my-group",eh_namespace="my-eventhub-ns",eventhub="eventhub-1"} 1.3395102e+07

# HELP eh_metrics_consumer_group_partition_owner info about owner of a partition in a consumer group. Value is 0 if owner is expired, otherwise 1.
# TYPE eh_metrics_consumer_group_partition_owner gauge
eh_metrics_consumer_group_partition_owner{consumer_group="my-group",eh_namespace="my-eventhub-ns",eventhub="eventhub-1",owner="int-my-group-655dcf764f-99lzg",partition_id="0"} 1

# HELP eh_metrics_consumer_group_partition_lag the number of messages a consumer group is lagging behind the last enqueued sequence number of a partition
# TYPE eh_metrics_consumer_group_partition_lag gauge
eh_metrics_consumer_group_partition_lag{consumer_group="my-group",eh_namespace="my-eventhub-ns",eventhub="eventhub-1",partition_id="0"} 0

# HELP eh_metrics_consumer_group_lag the number of messages a consumer group is lagging behind across all partitions in an eventhub
# TYPE eh_metrics_consumer_group_lag gauge
eh_metrics_consumer_group_lag{consumer_group="my-group",eh_namespace="my-eventhub-ns",eventhub="eventhub-1"} 0
```

## 🔧 Configuration

All options can be configured via YAML or environment variables. Configuring some options via YAML and some via environment variables is also possible. Environment variables take precedence in this case.

If you want to use a YAML config file, specify the path to the config file by setting the env variable `CONFIG_FILEPATH`.

The following block shows the reference config with all possible configuration parameters:

```yaml
namespaces:
  - 
    # fully qualified namespace name of the Event Hubs namespace name
    endpoint: my-eventhub.servicebus.windows.net
    # regex pattern to include event hubs (optional)
    includedEventHubs: .+test.+
    # regex pattern to exclude event hubs (optional)
    excludedEventHubs: .+test.+
    # regex pattern to exclude consumer groups (optional)
    excludedConsumerGroups: \$Default|test.+

storageAccounts:
  -
    # name of the Blob Service endpoint which stores the checkpoints
    endpoint: mystorage.blob.core.windows.net
    # regex pattern to include containers which store checkpoints (optional)
    includedContainers: .+test.+
    # regex pattern to exclude containers which store checkpoints (optional)
    excludedContainers: .+test.+

exporter:
  # export metrics to AppInsights
  appInsights:
    # enable appInsights exporter (default: false)
    enabled: true
    # instrumentation key to use
    instrumentationKey: xxx

  # run a http server which exposes /metrics
  prometheus:
    # enable prometheus exporter (default: false)
    enabled: true
    # read timeout for http request (default: 1s)
    readTimeout: 15s
    # address for the http server (default: :8080)
    address: localhost:9090
  
  # export metrics by sending them to a pushGateway
  pushGateway:
    # enabled pushGateway exporter (default: false)
    enabled: true
    # baseUrl of the pushGateway
    baseUrl: http://pushgateway.monitoring.svc.cluster.local
    
  # export metrics by sending them to an OpenTelemetry collector
  otlp:
    # enable OpenTelemetry exporter (default: false)
    enabled: true
    # protocol to use (default: grpc)
    protocol: http
    # baseUrl of the OpenTelemetry collector
    baseUrl: http://localhost:4318

collector:
  # duration after which an ownership is considered expired (default: 1m)
  ownershipExpirationDuration: 1m
  # determines how many eventhubs are processed concurrently (default: 8)
  concurrency: 10
  # interval in which the metrics are updated.
  # if not specified the application will exit after one iteration.
  interval: 5m

log:
  # one of debug, info, warn, error (default: info)
  level: info
  # one of json, text (default: text)
  format: json
```

## Running in Azure Kubernetes Service (AKS)

### Required Workload identity

The application requires a workload identity with the following role assignments:

1. [azure-event-hubs-data-receiver role](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#azure-event-hubs-data-receiver) is required with scope on all
  eventhub namespaces that need to be queried. Since the application lists all eventhubs in a namespace it is currently
  not sufficient to have role assignments on individual eventhubs.
2. [storage-blob-data-reader role](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/storage#storage-blob-data-reader) for all configured storage
  accounts is required, so that the checkpoints for all consumerGroups can be read from the storage accounts.

### Example Helm configuration

```yaml
managedIdentity:
    clientId: <clientId of the workload identity>
resources:
    limits:
        cpu: 100m
        memory: 128Mi
    requests:
        cpu: 100m
        memory: 128Mi
config:
    namespaces:
        - endpoint: <your-namespace>.servicebus.windows.net
          storageAccountEndpoint: <your-storage>.blob.core.windows.net
          excludedConsumerGroups: \$Default
    storageAccounts:
      - endpoint: <your-storage>.blob.core.windows.net
    exporter:
        prometheus:
            enabled: true
    collector:
        concurrency: 10
        interval: 5m
    log:
        level: debug
        format: json
```

## Development

### OTLP Exporter

In order to test the _OTLP Exporter_ locally, we can spin up a docker container with Grafana and an OpenTelemetry Collector:

```shell
docker run --rm \
  --env GF_AUTH_ANONYMOUS_ENABLED=true \
  --env GF_AUTH_ANONYMOUS_ORG_ROLE=Admin \
  -p "3000:3000" \
  -p "4317:4317" \
  -p "4318:4318" \
  grafana/otel-lgtm:latest
```

and use the following config:

```yaml
...
exporter:
  otlp:
    enabled: true
    baseURL: "http://localhost:4317"
```
