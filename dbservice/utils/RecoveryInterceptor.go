package utils

import (
	"context"
	"log"
	"runtime/debug"

	"google.golang.org/grpc"
)

// RecoveryInterceptor - gRPC interceptor для обработки паники
func RecoveryInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic occurred: %v\nStack Trace:\n%s", r, string(debug.Stack()))
			err = grpc.Errorf(grpc.Code(err), "internal server error")
		}
	}()
	return handler(ctx, req)
}
