package broker

import (
	"github.com/kyma-project/helm-broker/internal"
)

func NewBindService(ag addonIDGetter, cg chartGetter,  ig instanceGetter,
	ibdg instanceBindDataGetter, ibdi instanceBindDataInserter,
	btplrndr bindTemplateRenderer, btplres bindTemplateResolver, bsg bindStateGetter,
	boi bindOperationInserter, bog bindOperationGetter, bocg bindOperationCollectionGetter,
	bou bindOperationUpdater, idp func() (internal.OperationID, error)) *bindService {
	return &bindService{
		addonIDGetter:        ag,
		chartGetter:          cg,
		instanceGetter:       ig,
		bindTemplateRenderer: btplrndr,
		bindTemplateResolver: btplres,
		instanceBindDataGetter: ibdg,
		instanceBindDataInserter: ibdi,
		bindStateGetter:       bsg,
		bindOperationGetter:           bog,
		bindOperationCollectionGetter: bocg,
		bindOperationInserter: boi,
		bindOperationUpdater:  bou,
		operationIDProvider:  idp,
	}
	}

func (svc *bindService) WithTestHookOnAsyncCalled(h func(internal.OperationID)) *bindService {
	svc.testHookAsyncCalled = h
	return svc
}

