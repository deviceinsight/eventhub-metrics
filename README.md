# Eventhub metrics

## Configuration

```yaml
namespaces:
  - 
    # fully qualified namespace name of the Event Hubs namespace name
    endpoint: my-eventhub.servicebus.windows.net
    # name of the Blob Service endpoint which stores the checkpoints
    storageAccountEndpoint: mystorage.blob.core.windows.net
    # name of the container which stores the checkpoints
    checkpointContainer: checkpoints
    # regex pattern to exclude consumer groups
    excludedEventHubs: .+test.+
    # regex pattern to exclude consumer groups
    excludedConsumerGroups: \$Default|test.+

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

collector:
  # duration after which an ownership is considered expired (default: 1m)
  ownershipExpirationDuration: 1m
  # determines how many eventhubs are processed concurrently (default: 8)
  concurrency: 10
  # interval in which the metrics are updated
  interval: 5m

log:
  # one of debug, info, warn, error (default: info)
  level: info
  # one of json, text (default: text)
  format: json
```
