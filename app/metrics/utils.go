package metrics

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	gometrics "github.com/hashicorp/go-metrics"
)

// NewLabel creates new gometrics.Label.
func NewLabel(name, value string) gometrics.Label {
	return gometrics.Label{Name: name, Value: value}
}

// ModuleMeasureSinceWithLabels emits a time measure metric for a module with a given set of keys and labels.
func ModuleMeasureSinceWithLabels(module string, start time.Time, keys []string, extraLabels []gometrics.Label) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	labels := append(
		[]gometrics.Label{
			telemetry.NewLabel(telemetry.MetricLabelNameModule, module),
		},
		extraLabels...,
	)

	gometrics.MeasureSinceWithLabels(
		keys,
		start.UTC(),
		labels,
	)
}

// ModuleIncrCounterWithLabels emits a counter metric for a module with a given set of keys and labels.
func ModuleIncrCounterWithLabels(module string, value float32, keys []string, extraLabels []gometrics.Label) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	labels := append(
		[]gometrics.Label{
			telemetry.NewLabel(telemetry.MetricLabelNameModule, module),
		},
		extraLabels...,
	)

	gometrics.IncrCounterWithLabels(
		keys,
		value,
		labels,
	)
}

// ModuleMeasureWithCounter emits latency and success/error counter metrics for a module message with optional labels.
func ModuleMeasureWithCounter(module, msgType string, start time.Time, err error, extraLabels []gometrics.Label) {
	labels := []gometrics.Label{
		telemetry.NewLabel(Msg, msgType),
	}

	ModuleMeasureSinceWithLabels(
		module,
		start,
		[]string{App, Msg, Latency},
		labels,
	)

	keys := []string{App, Msg, Count}
	if err != nil {
		keys = []string{App, Error, Count}
	}

	ModuleIncrCounterWithLabels(
		module,
		1,
		keys,
		append(labels, extraLabels...),
	)
}
