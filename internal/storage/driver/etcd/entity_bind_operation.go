package etcd

import (
	"bytes"
	"context"
	"encoding/gob"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/namespace"

	"github.com/kyma-project/helm-broker/internal/platform/ptr"
	yTime "github.com/kyma-project/helm-broker/internal/platform/time"

	"github.com/kyma-project/helm-broker/internal"
)

// NewBindOperation returns new instance of BindOperation storage.
func NewBindOperation(cli clientv3.KV) (*BindOperation, error) {
	prefixParts := append(entityNamespacePrefixParts(), string(entityNamespaceBindOperation))
	kv := namespace.NewKV(cli, strings.Join(prefixParts, entityNamespaceSeparator))

	d := &BindOperation{
		generic: generic{
			kv: kv,
		},
	}

	return d, nil
}

// BindOperation implements in-memory storage InstanceOperation.
type BindOperation struct {
	generic
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
	opKey := s.key(bo.InstanceID, bo.BindingID, bo.OperationID)

	respGet, err := s.kv.Get(context.TODO(), opKey)
	if err != nil {
		return errors.Wrap(err, "while getting bind operation")
	}
	if respGet.Count > 0 {
		return alreadyExistsError{}
	}

	opInProgress, err := s.isOperationInProgress(bo.InstanceID, bo.BindingID)
	if err != nil {
		return errors.Wrap(err, "while checking if there are operations in progress")
	}
	if *opInProgress {
		return activeOperationInProgressError{}
	}

	bo.CreatedAt = s.nowProvider.Now()

	dso, err := s.encodeDMToDSO(bo)
	if err != nil {
		return err
	}

	if _, err := s.kv.Put(context.TODO(), opKey, dso); err != nil {
		return errors.Wrap(err, "while putting bind operation")
	}

	return nil
}

func (s *BindOperation) isOperationInProgress(iID internal.InstanceID, bID internal.BindingID) (*bool, error) {
	k := s.instanceKeyPrefix(iID) + s.bindKeyPrefix(bID)
	resp, err := s.kv.Get(context.TODO(), k, clientv3.WithPrefix())
	if err != nil {
		return nil, s.handleGetError(err)
	}

	for _, kv := range resp.Kvs {
		io, err := s.decodeDSOToDM(kv.Value)
		if err != nil {
			return nil, errors.Wrap(err, "while decoding returned entities")
		}

		if io.State == internal.OperationStateInProgress {
			return ptr.Bool(true), nil
		}
	}

	return ptr.Bool(false), nil
}

// Get returns object from storage.
func (s *BindOperation) Get(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) (*internal.BindOperation, error) {
	return s.get(iID, bID, opID)
}

func (s *BindOperation) get(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) (*internal.BindOperation, error) {
	resp, err := s.kv.Get(context.TODO(), s.key(iID, bID, opID))
	if err != nil {
		return nil, s.handleGetError(err)
	}

	switch resp.Count {
	case 1:
	case 0:
		return nil, notFoundError{}
	default:
		return nil, errors.New("more than one element matching requested id, should never happen")
	}

	return s.decodeDSOToDM(resp.Kvs[0].Value)
}

// GetAll returns all objects from storage for a given Instance ID
func (s *BindOperation) GetAll(iID internal.InstanceID) ([]*internal.BindOperation, error) {
	if iID.IsZero() {
		return nil, errors.New("instance id cannot be empty")
	}

	var out []*internal.BindOperation

	resp, err := s.kv.Get(context.TODO(), s.instanceKeyPrefix(iID), clientv3.WithPrefix())
	if err != nil {
		return nil, s.handleGetError(err)
	}

	if resp.Count == 0 {
		return nil, notFoundError{}
	}

	for _, kv := range resp.Kvs {
		io, err := s.decodeDSOToDM(kv.Value)
		if err != nil {
			return nil, errors.Wrap(err, "while decoding returned entities")
		}
		out = append(out, io)
	}

	return out, nil
}

// UpdateState modifies state on object in storage.
func (s *BindOperation) UpdateState(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID, state internal.OperationState) error {
	return s.updateStateDesc(iID, bID, opID, state, nil)
}

// UpdateStateDesc updates both state and description for single operation.
// If desc is nil than description will be removed.
func (s *BindOperation) UpdateStateDesc(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID, state internal.OperationState, desc *string) error {
	return s.updateStateDesc(iID, bID, opID, state, desc)
}

func (s *BindOperation) updateStateDesc(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID, state internal.OperationState, desc *string) error {
	bo, err := s.get(iID, bID, opID)
	if err != nil {
		return errors.Wrap(err, "while getting bind operation on updateStateDesc")
	}

	bo.State = state
	bo.StateDescription = desc

	dso, err := s.encodeDMToDSO(bo)
	if err != nil {
		return errors.Wrap(err, "while encoding bind operation on updateStateDesc")
	}

	if _, err := s.kv.Put(context.TODO(), s.key(iID, bID, opID), dso); err != nil {
		return errors.Wrap(err, "while calling database on put")
	}

	return nil
}

// Remove removes object from storage.
func (s *BindOperation) Remove(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) error {
	resp, err := s.kv.Delete(context.TODO(), s.key(iID, bID, opID))
	if err != nil {
		return errors.Wrap(err, "while deleting bind operation")
	}

	switch resp.Deleted {
	case 1:
	case 0:
		return notFoundError{}
	default:
		return errors.New("more than one element matching requested id, should never happen")
	}

	return nil
}

// key returns key for the specific bind operation in a instance space
func (s *BindOperation) key(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) string {
	return s.instanceKeyPrefix(iID) + s.bindKeyPrefix(bID) + string(opID)
}

// instanceKeyPrefix returns prefix for all bind operation keys in single instance namespace
// Trailing separator is appended.
func (*BindOperation) instanceKeyPrefix(id internal.InstanceID) string {
	return string(id) + entityNamespaceSeparator
}

// instanceKeyPrefix returns prefix for all bind operation keys in single instance namespace
// Trailing separator is appended.
func (*BindOperation) bindKeyPrefix(id internal.BindingID) string {
	return string(id) + entityOperationIDSeparator
}

func (*BindOperation) handleGetError(errIn error) error {
	return errors.Wrap(errIn, "while getting bind operation")
}

func (s *BindOperation) encodeDMToDSO(dm *internal.BindOperation) (string, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(dm); err != nil {
		return "", errors.Wrap(err, "while encoding entity")
	}

	return buf.String(), nil
}

func (s *BindOperation) decodeDSOToDM(dsoEnc []byte) (*internal.BindOperation, error) {
	dec := gob.NewDecoder(bytes.NewReader(dsoEnc))
	var bo internal.BindOperation
	if err := dec.Decode(&bo); err != nil {
		return nil, errors.Wrap(err, "while decoding DSO")
	}

	return &bo, nil
}
