package converting

import "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1/convertingv1connect"

// Service is ConvertingService on this Node, mounted in both halves.
func Service(_ Proofs, _ func() string) convertingv1connect.ConvertingServiceHandler {
	return convertingv1connect.UnimplementedConvertingServiceHandler{}
}
