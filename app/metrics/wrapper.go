package metrics

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	gometrics "github.com/hashicorp/go-metrics"
	grpc "google.golang.org/grpc"
)

// WrapMsgServerServiceDescriptor wraps a message service descriptor and adds metric instrumentation.
func WrapMsgServerServiceDescriptor(moduleName string, desc grpc.ServiceDesc) grpc.ServiceDesc {
	if !telemetry.IsTelemetryEnabled() {
		return desc
	}

	methods := make([]grpc.MethodDesc, 0, len(desc.Methods))
	for _, method := range desc.Methods {
		handler := wrapMsgSeverHandler(moduleName, method.MethodName,
			SourcehubMsgSeconds, SourcehubMsgTotal, SourcehubMsgErrorsTotal,
			method.Handler)
		method.Handler = handler
		methods = append(methods, method)
	}
	desc.Methods = methods
	return desc
}

// WrapQueryServiceDescriptor wraps a query service descriptor and adds metric instrumentation.
func WrapQueryServiceDescriptor(moduleName string, desc grpc.ServiceDesc) grpc.ServiceDesc {
	if !telemetry.IsTelemetryEnabled() {
		return desc
	}

	methods := make([]grpc.MethodDesc, 0, len(desc.Methods))
	for _, method := range desc.Methods {
		handler := wrapMsgSeverHandler(moduleName, method.MethodName,
			SourcehubQuerySeconds, SourcehubQueryTotal, SourcehubQueryErrorsTotal,
			method.Handler)
		method.Handler = handler
		methods = append(methods, method)
	}
	desc.Methods = methods
	return desc
}

// wrapMsgSeverHandler wraps an individual GRPC server method handler with metric collection logic.
// It tracks the number of processed messages, error count, and message handling latency.
func wrapMsgSeverHandler(
	moduleName, methodName string,
	latencyMetricName, countMetricName, errMetricName []string,
	handler grpc.MethodHandler,
) grpc.MethodHandler {
	return func(
		srv interface{},
		ctx context.Context,
		dec func(interface{}) error,
		interceptor grpc.UnaryServerInterceptor,
	) (interface{}, error) {
		labels := []Label{
			{Name: ModuleLabel, Value: moduleName},
			{Name: EndpointLabel, Value: methodName},
		}
		labels = append(labels, commonLabels...)

		// Track message handling latency
		now := time.Now()
		defer gometrics.MeasureSinceWithLabels(latencyMetricName, now, labels)

		// Increment message count
		gometrics.IncrCounterWithLabels(countMetricName, 1, labels)

		resp, err := handler(srv, ctx, dec, interceptor)
		if err != nil {
			// Increment error count
			gometrics.IncrCounterWithLabels(errMetricName, 1, labels)
		}
		return resp, err
	}
}
