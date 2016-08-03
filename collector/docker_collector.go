package collector

import (
	"github.com/docker/engine-api/types"
	"github.com/jhorwit2/docker_collector/aggregator"
	"github.com/prometheus/client_golang/prometheus"
)

type containerMetric struct {
	name       string
	help       string
	labels     []string
	metricType prometheus.ValueType
	getValue   func(s types.StatsJSON) float64
}

func (cm *containerMetric) desc(baseLabels []string) *prometheus.Desc {
	return prometheus.NewDesc(cm.name, cm.help, append(baseLabels, cm.labels...), nil)
}

type dockerCollector struct {
	prometheus.Collector

	metrics []containerMetric
	stats   *aggregator.StatAggregator
}

// New creates a new docker collector to collect
// stats for containers/tasks/services.
func New(stats *aggregator.StatAggregator) (prometheus.Collector, error) {
	return &dockerCollector{
		stats: stats,
		metrics: []containerMetric{
			containerMetric{
				name:       "memory_usage",
				help:       "Current memory usage in MiB",
				metricType: prometheus.CounterValue,
				getValue: func(s types.StatsJSON) float64 {
					return float64(s.MemoryStats.Usage)
				},
			},
			containerMetric{
				name:       "memory_limit",
				help:       "Memory limit in MiB",
				metricType: prometheus.CounterValue,
				getValue: func(s types.StatsJSON) float64 {
					return float64(s.MemoryStats.Limit)
				},
			},
			containerMetric{
				name:       "memory_percentage",
				help:       "Current memory percentage utilized in MiB",
				metricType: prometheus.CounterValue,
				getValue: func(s types.StatsJSON) float64 {
					return float64(s.MemoryStats.Usage) / float64(s.MemoryStats.Limit) * 100.0
				},
			},
		},
	}, nil
}

func (d *dockerCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, c := range d.metrics {
		ch <- c.desc([]string{})
	}
}

func (d *dockerCollector) Collect(ch chan<- prometheus.Metric) {
	stats := d.stats.Stats()

	labels := []string{
		"task_id",
		"service_id",
	}

	for containerInfo, stat := range stats {
		labelValues := []string{
			containerInfo.TaskID,
			containerInfo.ServiceID,
		}

		for _, cm := range d.metrics {
			desc := cm.desc(labels)
			value := cm.getValue(stat)
			ch <- prometheus.MustNewConstMetric(desc, cm.metricType, value, labelValues...)
		}
	}
}

func calculateCPUPercent(previousCPU, previousSystem uint64, v types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func calculateNetwork(network map[string]types.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}
