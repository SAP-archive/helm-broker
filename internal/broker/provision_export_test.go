package broker

import (
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/helm-broker/internal"
)

func NewProvisionService(bg addonIDGetter, cg chartGetter, is instanceStorage, isg instanceStateGetter, oi operationInserter, ou operationUpdater,
	hi helmInstaller, oIDProv func() (internal.OperationID, error), log *logrus.Entry) *provisionService {
	return &provisionService{
		addonIDGetter:            bg,
		chartGetter:              cg,
		instanceGetter:           is,
		instanceInserter:         is,
		instanceStateGetter:      isg,
		operationInserter:        oi,
		operationUpdater:         ou,
		operationIDProvider:      oIDProv,
		helmInstaller:            hi,
		log:                      log,
	}
}

func (svc *provisionService) WithTestHookOnAsyncCalled(h func(internal.OperationID)) *provisionService {
	svc.testHookAsyncCalled = h
	return svc
}
