package broker

import (
	"github.com/kyma-project/helm-broker/internal"
)

func NewBindService(ag addonIDGetter, cg chartGetter, ig instanceGetter, ibds instanceBindDataStorage,
	btplrndr bindTemplateRenderer, btplres bindTemplateResolver, bsg bindStateGetter,
	bos bindOperationStorage, idp func() (internal.OperationID, error)) *bindService {
	return &bindService{
		addonIDGetter:           ag,
		chartGetter:             cg,
		instanceGetter:          ig,
		instanceBindDataStorage: ibds,
		bindTemplateRenderer:    btplrndr,
		bindTemplateResolver:    btplres,
		bindStateGetter:         bsg,
		bindOperationStorage:    bos,
		operationIDProvider:     idp,
	}
}

func (svc *bindService) WithTestHookOnAsyncCalled(h func(internal.OperationID)) *bindService {
	svc.testHookAsyncCalled = h
	return svc
}
