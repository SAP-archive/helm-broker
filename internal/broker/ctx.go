package broker

import (
	"context"
	"strings"

	osb "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/pkg/errors"
)

type contextKey int

const (
	osbAPIVersion = "2.14"

	osbContextKey contextKey = 5001
)

// OsbContext contains data sent in X-Broker-API-Version and X-Broker-API-Originating-Identity HTTP headers.
type OsbContext struct {
	APIVersion          string
	OriginatingIdentity string
	BrokerNamespace     internal.Namespace
}

func (ctx *OsbContext) validateAPIVersion() error {
	if ctx.APIVersion != osbAPIVersion {
		return errors.Errorf("while checking 'X-Broker-API-Version' header, should be %s, got %s", osbAPIVersion, ctx.APIVersion)
	}
	return nil
}

func (ctx *OsbContext) validateOriginatingIdentity() error {
	if ctx.OriginatingIdentity != "" && !strings.Contains(ctx.OriginatingIdentity, osb.PlatformKubernetes) {
		return errors.Errorf("while checking 'X-Broker-API-Originating-Identity' header, should be %s, got %s", osb.PlatformKubernetes, ctx.OriginatingIdentity)
	}
	return nil
}

func contextWithOSB(ctx context.Context, osbCtx OsbContext) context.Context {
	return context.WithValue(ctx, osbContextKey, osbCtx)
}

func osbContextFromContext(ctx context.Context) (OsbContext, bool) {
	osbCtx, ok := ctx.Value(osbContextKey).(OsbContext)
	return osbCtx, ok
}
