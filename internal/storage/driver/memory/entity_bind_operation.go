package memory

import (
	"time"

	"github.com/pkg/errors"

	yTime "github.com/kyma-project/helm-broker/internal/platform/time"

	"github.com/kyma-project/helm-broker/internal"
)

// NewBindOperation returns new instance of BindOperation storage.
func NewBindOperation() *BindOperation {
	return &BindOperation{
		storage: make(map[internal.InstanceID]map[internal.OperationID]*internal.BindOperation),
	}
}

// BindOperation implements in-memory storage BindOperation.
type BindOperation struct {
	threadSafeStorage
	storage     map[internal.InstanceID]map[internal.OperationID]*internal.BindOperation
	nowProvider yTime.NowProvider
}

// WithTimeProvider allows for passing custom time provider.
// Used mostly in testing.
func (s *BindOperation) WithTimeProvider(nowProvider func() time.Time) *BindOperation {
	s.nowProvider = nowProvider
	return s
}

// Insert inserts object into storage.
func (s *BindOperation) Insert(bo *internal.BindOperation) error {
	defer unlock(s.lockW())

	if _, found := s.storage[bo.InstanceID]; !found {
		s.storage[bo.InstanceID] = make(map[internal.OperationID]*internal.BindOperation)
	}

	for oID := range s.storage[bo.InstanceID] {
		if s.storage[bo.InstanceID][oID].State == internal.OperationStateInProgress {
			return activeOperationInProgressError{}
		}
	}

	bo.CreatedAt = s.nowProvider.Now()

	s.storage[bo.InstanceID][bo.OperationID] = bo

	return nil
}

// Get returns object from storage.
func (s *BindOperation) Get(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) (*internal.BindOperation, error) {
	defer unlock(s.lockR())

	return s.get(iID, bID, opID)
}

func (s *BindOperation) get(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) (*internal.BindOperation, error) {
	if iID.IsZero() || bID.IsZero() || opID.IsZero() {
		return nil, errors.Errorf("all parameters: instance, binding and operation id must be set. InstanceID: %q | BindingID: %q | OperationID: %q", iID, bID, opID)
	}

	if _, found := s.storage[iID]; !found {
		return nil, notFoundError{}
	}

	io, found := s.storage[iID][opID]
	if !found {
		return nil, notFoundError{}
	}

	return io, nil
}

// GetAll returns all objects from storage.
func (s *BindOperation) GetAll(iID internal.InstanceID) ([]*internal.BindOperation, error) {
	defer unlock(s.lockR())

	if iID.IsZero() {
		return nil, errors.New("instance id cannot be empty")
	}

	out := []*internal.BindOperation{}

	opsForInstance, found := s.storage[iID]
	if !found {
		return nil, notFoundError{}
	}

	for i := range opsForInstance {
		out = append(out, opsForInstance[i])
	}

	return out, nil
}

// UpdateState modifies state on object in storage.
func (s *BindOperation) UpdateState(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID, state internal.OperationState) error {
	defer unlock(s.lockW())

	op, err := s.get(iID, bID, opID)
	if err != nil {
		return errors.Wrap(err, "while getting bind operation on UpdateState")
	}

	op.State = state
	op.StateDescription = nil

	//s.logStateChange(iID, opID, state, nil)
	return nil
}

// UpdateStateDesc updates both state and description for single operation.
// If desc is nil than description will be removed.
func (s *BindOperation) UpdateStateDesc(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID, state internal.OperationState, desc *string) error {
	defer unlock(s.lockW())

	op, err := s.get(iID, bID, opID)
	if err != nil {
		return errors.Wrap(err, "while getting bind operation on UpdateStateDesc")
	}

	op.State = state
	op.StateDescription = desc

	//s.logStateChange(iID, opID, state, desc)
	return nil
}

// Remove removes object from storage.
func (s *BindOperation) Remove(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) error {
	defer unlock(s.lockW())

	if _, err := s.get(iID, bID, opID); err != nil {
		return errors.Wrap(err, "while getting bind operation on Remove")
	}

	delete(s.storage[iID], opID)
	if len(s.storage[iID]) == 0 {
		delete(s.storage, iID)
	}

	return nil
}
