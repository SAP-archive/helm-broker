package broker

import (
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/helm-broker/internal"
)

func NewWithIDProvider(bs addonStorage, cs chartStorage, os operationStorage, bos bindOperationStorage, is instanceStorage, ibd instanceBindDataStorage,
	bindTmplRenderer bindTemplateRenderer, bindTmplResolver bindTemplateResolver,
	hc helmClient, log *logrus.Entry, idp func() (internal.OperationID, error)) *Server {
	return newWithIDProvider(bs, cs, os, bos, is, ibd, bindTmplRenderer, bindTmplResolver, hc, log, idp)
}
