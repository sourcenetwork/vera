package metrics

import (
	"context"
	"time"

	gometrics "github.com/hashicorp/go-metrics"
	grpc "google.golang.org/grpc"
)

// WrapMsgServerServiceDescriptor wraps a service descriptor,
// assume to be of msg servers, and add metric collection
// to it.
func WrapMsgServerServiceDescriptor(moduleName string, desc grpc.ServiceDesc) grpc.ServiceDesc {
	methods := make([]grpc.MethodDesc, 0, len(desc.Methods))
	for _, method := range desc.Methods {
		handler := wrapMsgSeverHandler(moduleName, method.MethodName, method.Handler)
		method.Handler = handler
		methods = append(methods, method)
	}
	desc.Methods = methods
	return desc
}

// wrapMsgSeverHandler wraps an individual GRPC server method handler
// with metric collection logic.
// The wrapped method tracks the number of processed messages,
// number of errors returned and latecy.
func wrapMsgSeverHandler(moduleName string, methodName string, handler grpc.MethodHandler) grpc.MethodHandler {
	return func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
		labels := []gometrics.Label{
			{
				Name:  "module",
				Value: moduleName,
			},
			{
				Name:  "endpoint",
				Value: methodName,
			},
		}

		// measure msg handling latency
		now := time.Now()
		defer gometrics.MeasureSinceWithLabels(SourcehubMsgSeconds, now, labels)

		// total msg count
		gometrics.IncrCounterWithLabels(SourcehubMsgTotal, 1, labels)

		resp, err := handler(srv, ctx, dec, interceptor)
		if err != nil {
			// count error if returned
			gometrics.IncrCounterWithLabels(SourcehubErrorsTotal, 1, labels)
		}
		return resp, err
	}
}
