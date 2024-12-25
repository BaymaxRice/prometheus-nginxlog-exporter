package metrics

import (
	"github.com/BaymaxRice/prometheus-nginxlog-exporter/pkg/config"
	"github.com/BaymaxRice/prometheus-nginxlog-exporter/pkg/relabeling"
	"github.com/prometheus/client_golang/prometheus"
)

// Init initializes a metrics struct
func (m *Collection) Init(cfg *config.NamespaceConfig) {
	cfg.MustCompile()

	labels := cfg.OrderedLabelNames
	counterLabels := labels

	relabelings := relabeling.NewRelabelings(cfg.RelabelConfigs)
	relabelings = append(relabelings, relabeling.DefaultRelabelings...)
	relabelings = relabeling.UniqueRelabelings(relabelings)

	for _, r := range relabelings {
		if !r.OnlyCounter {
			labels = append(labels, r.TargetLabel)
		}
		counterLabels = append(counterLabels, r.TargetLabel)
	}

	m.CountTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_response_count_total",
		Help:        "Amount of processed HTTP requests",
	}, counterLabels)

	m.ParseErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "parse_errors_total",
		Help:        "Total number of log file lines that could not be parsed",
	})

	m.OthersMetrics = make(map[string]prometheus.Collector, len(cfg.OthersMetrics))
	for field, v := range cfg.OthersMetrics {
		if v.MetricsType&config.MetricsTypeCounter != 0 {
			m.OthersMetrics[field+config.METRICS_SPLIT+config.METRICS_COUNTER_TAIL] = prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace:   cfg.NamespacePrefix,
				ConstLabels: cfg.NamespaceLabels,
				Name:        v.MetricsName + config.METRICS_COUNTER_TAIL,
				Help:        v.MetricsHelp,
			}, labels)
		}
		if v.MetricsType&config.MetricsTypeGauge != 0 {
			m.OthersMetrics[field+config.METRICS_SPLIT+config.METRICS_GAUGE_TAIL] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace:   cfg.NamespacePrefix,
				ConstLabels: cfg.NamespaceLabels,
				Name:        v.MetricsName + config.METRICS_GAUGE_TAIL,
				Help:        v.MetricsHelp,
			}, labels)
		}
		if v.MetricsType&config.MetricsTypeHistogram != 0 {
			m.OthersMetrics[field+config.METRICS_SPLIT+config.METRICS_HIST_TAIL] = prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace:   cfg.NamespacePrefix,
				ConstLabels: cfg.NamespaceLabels,
				Name:        v.MetricsName + config.METRICS_HIST_TAIL,
				Help:        v.MetricsHelp,
				Buckets:     v.HistogramBuckets,
			}, labels)
		}
		if v.MetricsType&config.MetricsTypeSummary != 0 {
			opts := prometheus.SummaryOpts{
				Namespace:   cfg.NamespacePrefix,
				ConstLabels: cfg.NamespaceLabels,
				Name:        v.MetricsName + config.METRICS_SUMMARY_TAIL,
				Help:        v.MetricsHelp,
				Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			}
			if v.MaxAge > 0 {
				opts.MaxAge = v.MaxAge
			}
			if len(v.Objectives) > 0 {
				opts.Objectives = v.Objectives
			}
			m.OthersMetrics[field+config.METRICS_SPLIT+config.METRICS_SUMMARY_TAIL] = prometheus.NewSummaryVec(opts, labels)
		}
	}
}
