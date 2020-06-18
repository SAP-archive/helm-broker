package automock

import (
	"github.com/kyma-project/helm-broker/internal"
	"github.com/stretchr/testify/mock"
)

// InstanceStateGetter extensions
func (_m *instanceStateGetter) ExpectOnIsDeprovisioned(iID internal.InstanceID, deprovisioned bool) *mock.Call {
	return _m.On("IsDeprovisioned", iID).Return(deprovisioned, nil)
}

func (_m *instanceStateGetter) ExpectOnIsDeprovisioningInProgress(iID internal.InstanceID, optID internal.OperationID, inProgress bool) *mock.Call {
	return _m.On("IsDeprovisioningInProgress", iID).Return(optID, inProgress, nil)
}

func (_m *instanceStateGetter) ExpectErrorOnIsDeprovisioningInProgress(iID internal.InstanceID, err error) *mock.Call {
	return _m.On("IsDeprovisioningInProgress", iID).Return(internal.OperationID(""), false, err)
}

func (_m *instanceStateGetter) ExpectErrorIsDeprovisioned(iID internal.InstanceID, err error) *mock.Call {
	return _m.On("IsDeprovisioned", iID).Return(false, err)
}

// InstanceStorage extensions
func (_m *instanceStorage) ExpectOnGet(iID internal.InstanceID, expInstance internal.Instance) *mock.Call {
	return _m.On("Get", iID).Return(&expInstance, nil)
}

func (_m *instanceStorage) ExpectOnRemove(iID internal.InstanceID) *mock.Call {
	return _m.On("Remove", iID).Return(nil)
}

func (_m *instanceStorage) ExpectErrorRemove(iID internal.InstanceID, err error) *mock.Call {
	return _m.On("Remove", iID).Return(err)
}

func (_m *instanceStorage) ExpectErrorOnGet(iID internal.InstanceID, err error) *mock.Call {
	return _m.On("Get", iID).Return(nil, err)
}

// OperationStorage extensions
func (_m *operationStorage) ExpectOnInsert(op internal.InstanceOperation) *mock.Call {
	return _m.On("Insert", &op).Return(nil)
}

func (_m *operationStorage) ExpectOnUpdateStateDesc(iID internal.InstanceID, opID internal.OperationID, state internal.OperationState, desc string) *mock.Call {
	return _m.On("UpdateStateDesc", iID, opID, state, &desc).Return(nil)
}

// HelmClient extensions
func (_m *helmClient) ExpectOnDelete(rName internal.ReleaseName, ns internal.Namespace) *mock.Call {
	return _m.On("Delete", rName, ns).Return(nil)
}

func (_m *helmClient) ExpectErrorOnDelete(rName internal.ReleaseName, ns internal.Namespace, err error) *mock.Call {
	return _m.On("Delete", rName, ns).Return(err)
}

// InstanceBindDataRemover extensions
func (_m *instanceBindDataRemover) ExpectOnRemove(iID internal.InstanceID) *mock.Call {
	return _m.On("Remove", iID).Return(nil)
}

func (_m *instanceBindDataRemover) ExpectErrorRemove(iID internal.InstanceID, err error) *mock.Call {
	return _m.On("Remove", iID).Return(err)
}
