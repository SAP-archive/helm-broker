package broker

import (
	"github.com/kyma-project/helm-broker/internal"
)

func NewBindService(ag addonIDGetter, cg chartGetter, ig instanceGetter,
	btplrndr bindTemplateRenderer, btplres bindTemplateResolver, idp func() (internal.OperationID, error)) *bindService {
	return &bindService{
		addonIDGetter:        ag,
		chartGetter:          cg,
		instanceGetter:       ig,
		bindTemplateRenderer: btplrndr,
		bindTemplateResolver: btplres,
		resolvedBindData:     make(map[internal.BindingID]*internal.InstanceBindData),
		bindOperation:        make(map[internal.InstanceID][]*internal.BindOperation),
		operationIDProvider:  idp,
	}
	}

