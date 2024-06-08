package prometheus

import (
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
)

// counterAdd wraps Add of prom.Counter.
func counterAdd(counterVec *prom.CounterVec, value int, labels prom.Labels) error {
	counter, err := counterVec.GetMetricWith(labels)
	if err != nil {
		return err
	}
	counter.Add(float64(value))
	return nil
}

// histogramObserve wraps Observe of prom.Observer.
func histogramObserve(histogramVec *prom.HistogramVec, value time.Duration, labels prom.Labels) error {
	histogram, err := histogramVec.GetMetricWith(labels)
	if err != nil {
		return err
	}
	histogram.Observe(float64(value.Seconds()))
	return nil
}
