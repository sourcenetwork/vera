package metrics

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	gometrics "github.com/hashicorp/go-metrics"
)

type Label = gometrics.Label

// NewLabel creates new Label with given name and value.
func NewLabel(name, value string) Label {
	return Label{Name: name, Value: value}
}

// ModuleMeasureSinceWithLabels emits a time measure metric for a module with a given set of keys and labels.
func ModuleMeasureSinceWithLabels(module string, start time.Time, keys []string, extraLabels []Label) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	labels := append(
		[]Label{
			NewLabel(telemetry.MetricLabelNameModule, module),
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
func ModuleIncrCounterWithLabels(module string, value float32, keys []string, extraLabels []Label) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	labels := append(
		[]Label{
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
func ModuleMeasureWithCounter(module, msgType string, start time.Time, err error, extraLabels []Label) {
	labels := []Label{
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

// ModuleMeasurSinceWithCounter emits latency and counter metrics for a module method with common and extra labels.
func ModuleMeasurSinceWithCounter(moduleName, methodName string, start time.Time, err error, extraLabels []Label) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	labels := []Label{
		{Name: ModuleLabel, Value: moduleName},
		{Name: EndpointLabel, Value: methodName},
	}
	labels = append(labels, commonLabels...)
	labels = append(labels, extraLabels...)

	// Track message handling latency
	gometrics.MeasureSinceWithLabels(SourcehubMsgSeconds, start, labels)

	// Increment message count
	gometrics.IncrCounterWithLabels(SourcehubMsgTotal, 1, labels)

	if err != nil {
		// Increment error count
		gometrics.IncrCounterWithLabels(SourcehubMsgErrorsTotal, 1, labels)
	}
}

// ModuleIncrInternalErrorCounter tracks internal method errors for a module.
func ModuleIncrInternalErrorCounter(moduleName, methodName string, err error) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	labels := []Label{
		{Name: ModuleLabel, Value: moduleName},
		{Name: EndpointLabel, Value: methodName},
	}
	labels = append(labels, commonLabels...)

	if err != nil {
		gometrics.IncrCounterWithLabels(SourcehubInternalErrorsTotal, 1, labels)
	}
}
