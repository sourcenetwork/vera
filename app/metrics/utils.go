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

// ModuleMeasureSinceWithCounter emits latency and counter metrics for a module method with common and extra labels.
func ModuleMeasureSinceWithCounter(moduleName, methodName string, start time.Time, err error, extraLabels []Label) {
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
	gometrics.MeasureSinceWithLabels(SourcehubMethodSeconds, start, labels)

	// Increment message count
	gometrics.IncrCounterWithLabels(SourcehubMethodTotal, 1, labels)

	if err != nil {
		// Increment error count
		gometrics.IncrCounterWithLabels(SourcehubMethodErrorsTotal, 1, labels)
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
