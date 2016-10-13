package stackdriver

import "reflect"

type MetricsBuffer interface {
	PostMetric(*Metric)
}

type metricsBuffer struct {
	size    int
	adapter MetricAdapter
	errs    chan error
	metrics []Metric
}

func NewMetricsBuffer(size int, adapter MetricAdapter) (MetricsBuffer, <-chan error) {
	errs := make(chan error)
	return &metricsBuffer{size, adapter, errs, []Metric{}}, errs
}

func (mb *metricsBuffer) PostMetric(metric *Metric) {
	mb.addMetric(metric)

	if len(mb.metrics) < mb.size {
		return
	}

	mb.postMetrics(mb.metrics)
	mb.metrics = []Metric{}
}

func (mb *metricsBuffer) addMetric(newMetric *Metric) {
	var existingMetric *Metric

	for _, metric := range mb.metrics {
		if metric.Name == newMetric.Name &&
			reflect.DeepEqual(metric.Labels, newMetric.Labels) {
			existingMetric = &metric
			break
		}
	}

	if existingMetric == nil {
		mb.metrics = append(mb.metrics, *newMetric)
	} else {
		mb.postMetrics([]Metric{*newMetric})
	}
}

func (mb *metricsBuffer) postMetrics(metrics []Metric) {
	err := mb.adapter.PostMetrics(metrics)
	if err != nil {
		go func() { mb.errs <- err }()
	}

}