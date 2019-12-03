package broker

import (
	"github.com/pkg/errors"

	"github.com/kyma-project/helm-broker/internal"
)

type instanceStateService struct {
	operationCollectionGetter operationCollectionGetter
}

func (svc *instanceStateService) IsProvisioned(iID internal.InstanceID) (bool, error) {
	result := false

	ops, err := svc.operationCollectionGetter.GetAll(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return false, nil
	default:
		return false, errors.Wrap(err, "while getting instance operations from storage")
	}

OpsLoop:
	for _, op := range ops {
		if op.Type == internal.OperationTypeCreate && op.State == internal.OperationStateSucceeded {
			result = true
		}
		if op.Type == internal.OperationTypeRemove && op.State == internal.OperationStateSucceeded {
			result = false
			break OpsLoop
		}
	}

	return result, nil
}

func (svc *instanceStateService) IsProvisioningInProgress(iID internal.InstanceID) (internal.OperationID, bool, error) {
	resultInProgress := false
	var resultOpID internal.OperationID

	ops, err := svc.operationCollectionGetter.GetAll(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return resultOpID, false, nil
	default:
		return resultOpID, false, errors.Wrap(err, "while getting instance operations from storage")
	}

OpsLoop:
	for _, op := range ops {
		if op.Type == internal.OperationTypeCreate && op.State == internal.OperationStateInProgress {
			resultInProgress = true
			resultOpID = op.OperationID
			break OpsLoop
		}
	}

	return resultOpID, resultInProgress, nil
}

func (svc *instanceStateService) IsDeprovisioned(iID internal.InstanceID) (bool, error) {
	result := false

	ops, err := svc.operationCollectionGetter.GetAll(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return false, err
	default:
		return false, errors.Wrap(err, "while getting instance operations from storage")
	}

OpsLoop:
	for _, op := range ops {
		if op.Type == internal.OperationTypeRemove && op.State == internal.OperationStateSucceeded {
			result = true
			break OpsLoop
		}
	}

	return result, nil
}

func (svc *instanceStateService) IsDeprovisioningInProgress(iID internal.InstanceID) (internal.OperationID, bool, error) {
	resultInProgress := false
	var resultOpID internal.OperationID

	ops, err := svc.operationCollectionGetter.GetAll(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return resultOpID, false, nil
	default:
		return resultOpID, false, errors.Wrap(err, "while getting instance operations from storage")
	}

OpsLoop:
	for _, op := range ops {
		if op.Type == internal.OperationTypeRemove && op.State == internal.OperationStateInProgress {
			resultInProgress = true
			resultOpID = op.OperationID
			break OpsLoop
		}
	}

	return resultOpID, resultInProgress, nil
}

type bindStateService struct {
	bindOperationCollectionGetter bindOperationCollectionGetter
}

func (svc *bindStateService) IsBound(iID internal.InstanceID, bID internal.BindingID) (internal.BindOperation, bool, error) {
	result := false
	ops, err := svc.bindOperationCollectionGetter.GetAll(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return internal.BindOperation{}, false, nil
	default:
		return internal.BindOperation{}, false, errors.Wrap(err, "while getting bind operations from storage")
	}

	boundOp := &internal.BindOperation{}
	for _, op := range ops {
		if op.Type == internal.OperationTypeCreate && op.State == internal.OperationStateSucceeded && op.BindingID == bID {
			result = true
			boundOp = op
		}
		if op.Type == internal.OperationTypeRemove && op.State == internal.OperationStateSucceeded && op.BindingID == bID {
			result = false
			boundOp = &internal.BindOperation{}
			break
		}
	}

	return *boundOp, result, nil
}

func (svc *bindStateService) IsBindingInProgress(iID internal.InstanceID, bID internal.BindingID) (internal.OperationID, bool, error) {
	result := false
	var resultOpID internal.OperationID

	ops, err := svc.bindOperationCollectionGetter.GetAll(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return resultOpID, false, nil
	default:
		return resultOpID, false, errors.Wrap(err, "while getting bind operations from storage")
	}

	for _, op := range ops {
		if op.Type == internal.OperationTypeCreate && op.State == internal.OperationStateInProgress && op.BindingID == bID {
			result = true
			resultOpID = op.OperationID
			break
		}
	}

	return resultOpID, result, nil
}

// IsNotFoundError check if error is NotFound one.
func IsNotFoundError(err error) bool {
	nfe, ok := err.(interface {
		NotFound() bool
	})
	return ok && nfe.NotFound()
}

// IsAlreadyExistsError checks if errors is BadRequest one.
func IsAlreadyExistsError(err error) bool {
	nfe, ok := err.(interface {
		AlreadyExists() bool
	})
	return ok && nfe.AlreadyExists()
}

// IsActiveOperationInProgressError checks if errors is BadRequest one.
func IsActiveOperationInProgressError(err error) bool {
	nfe, ok := err.(interface {
		ActiveOperationInProgress() bool
	})
	return ok && nfe.ActiveOperationInProgress()
}
