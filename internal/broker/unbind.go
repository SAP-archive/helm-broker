package broker

import (
	"context"

	osb "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
)

type unbindService struct{}

func (svc *unbindService) Unbind(ctx context.Context, osbCtx OsbContext, req *osb.UnbindRequest) (*osb.UnbindResponse, error) {
	return nil, nil
}
