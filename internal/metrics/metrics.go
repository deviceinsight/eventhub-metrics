package metrics

type Metric struct {
	Name   string
	Help   string
	Labels []string
}

var NamespaceInfo = &Metric{
	Name:   "namespace_info",
	Help:   "eventhub namespace info",
	Labels: []string{"eh_namespace", "eh_endpoint"},
}

var EventhubInfo = &Metric{
	Name:   "eventhub_info",
	Help:   "eventhub info",
	Labels: []string{"eh_namespace", "eventhub", "partition_count", "retention_in_days"},
}

var EventhubPartitionSequenceNumberMin = &Metric{
	Name:   "eventhub_partition_sequence_min",
	Help:   "beginning sequence number of a partition",
	Labels: []string{"eh_namespace", "eventhub", "partition_id"},
}

var EventhubSequenceNumberMinSum = &Metric{
	Name:   "eventhub_sequence_min_sum",
	Help:   "sum of all the eventhub's partition beginning sequence numbers",
	Labels: []string{"eh_namespace", "eventhub"},
}

var EventhubPartitionSequenceNumberMax = &Metric{
	Name:   "eventhub_partition_sequence_max",
	Help:   "last enqueued sequence number of a partition",
	Labels: []string{"eh_namespace", "eventhub", "partition_id"},
}

var EventhubSequenceNumberMaxSum = &Metric{
	Name:   "eventhub_sequence_max_sum",
	Help:   "sum of all the eventhub's partition last enqueued sequence numbers",
	Labels: []string{"eh_namespace", "eventhub"},
}

var ConsumerGroupInfo = &Metric{
	Name:   "consumer_group_info",
	Help:   "consumer group info gauges. It will report 1 if the group is in the stable state, otherwise 0.",
	Labels: []string{"eh_namespace", "eventhub", "consumer_group"},
}

var ConsumerGroupOwners = &Metric{
	Name:   "consumer_group_owners",
	Help:   "consumer group owner count gauges. It will report the number of owners in the consumer group",
	Labels: []string{"eh_namespace", "eventhub", "consumer_group"},
}

var ConsumerGroupEventsSum = &Metric{
	Name:   "consumer_group_events_sum",
	Help:   "the sum of all committed sequence numbers across all partitions in an eventhub",
	Labels: []string{"eh_namespace", "eventhub", "consumer_group"},
}

var ConsumerGroupPartitionLag = &Metric{
	Name: "consumer_group_partition_lag",
	Help: "the number of messages a consumer group is lagging behind the last enqueued sequence number" +
		" of a partition",
	Labels: []string{"eh_namespace", "eventhub", "consumer_group", "partition_id"},
}

var ConsumerGroupLag = &Metric{
	Name:   "consumer_group_lag",
	Help:   "the number of messages a consumer group is lagging behind across all partitions in an eventhub",
	Labels: []string{"eh_namespace", "eventhub", "consumer_group"},
}

var allMetrics = []*Metric{NamespaceInfo, EventhubInfo, EventhubPartitionSequenceNumberMin,
	EventhubSequenceNumberMinSum, EventhubPartitionSequenceNumberMax, EventhubSequenceNumberMaxSum, ConsumerGroupInfo,
	ConsumerGroupOwners, ConsumerGroupEventsSum, ConsumerGroupPartitionLag, ConsumerGroupLag}
