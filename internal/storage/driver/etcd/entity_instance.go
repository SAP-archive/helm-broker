package etcd

import (
	"bytes"
	"context"
	"encoding/gob"
	"strings"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/namespace"
)

// NewInstance creates new Instances storage
func NewInstance(cli clientv3.KV) (*Instance, error) {
	prefixParts := append(entityNamespacePrefixParts(), entityNamespaceInstance)
	kv := namespace.NewKV(cli, strings.Join(prefixParts, entityNamespaceSeparator))

	// Register interface types which are used by this domain.
	// Not registered globally as helm-broker gives an option to configure storage
	// driver for each domain, so they should be treated separately and cannot
	// assume that other domain registered that type already.
	gob.Register(map[string]interface{}{})

	return &Instance{
		generic: generic{
			kv: kv,
		},
	}, nil
}

// Instance implements etcd based storage for Instance entities.
type Instance struct {
	generic
}

// Upsert persists Instance in memory.
//
// If instance already exists in storage then full replace is performed.
//
// Replace is set to true if instance already existed in storage and was replaced.
func (s *Instance) Upsert(i *internal.Instance) (replaced bool, err error) {
	if i == nil {
		return false, errors.New("entity may not be nil")
	}

	if i.ID.IsZero() {
		return false, errors.New("instance id must be set")
	}

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(i); err != nil {
		return false, errors.Wrap(err, "while encoding entity")
	}

	resp, err := s.kv.Put(context.TODO(), s.key(i.ID), buf.String(), clientv3.WithPrevKV())
	if err != nil {
		return false, errors.Wrap(err, "while calling database on put")
	}

	if resp.PrevKv != nil {
		return true, nil
	}

	return false, nil
}

// Insert inserts object to storage.
func (s *Instance) Insert(i *internal.Instance) error {
	if i == nil {
		return errors.New("entity may not be nil")
	}

	if i.ID.IsZero() {
		return errors.New("instance id must be set")
	}

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(i); err != nil {
		return errors.Wrap(err, "while encoding entity")
	}

	respGet, err := s.kv.Get(context.TODO(), s.key(i.ID))
	if err != nil {
		return errors.Wrap(err, "while calling database on get")
	}
	if respGet.Count > 0 {
		return alreadyExistsError{}
	}

	if _, err := s.kv.Put(context.TODO(), s.key(i.ID), buf.String()); err != nil {
		return errors.Wrap(err, "while calling database on put")
	}

	return nil
}

// Get returns object from storage.
func (s *Instance) Get(id internal.InstanceID) (*internal.Instance, error) {
	resp, err := s.kv.Get(context.TODO(), s.key(id))
	if err != nil {
		return nil, errors.Wrap(err, "while calling database")
	}

	switch resp.Count {
	case 1:
	case 0:
		return nil, notFoundError{}
	default:
		return nil, errors.New("more than one element matching requested id, should never happen")
	}

	i, err := s.decodeInstance(resp.Kvs[0].Value)
	if err != nil {
		return nil, errors.Wrap(err, "while decoding single DSO")
	}

	return i, nil
}

// GetAll returns collection of Instance objects from storage
func (s *Instance) GetAll() ([]*internal.Instance, error) {
	out := []*internal.Instance{}

	// special chart NULL hex (\x00) is used to select all entities with prefix defined during create new instance
	// empty string is not allowed for etcd/clientv3 library
	resp, err := s.kv.Get(context.TODO(), "\x00", clientv3.WithFromKey())
	if err != nil {
		return nil, errors.Wrap(err, "while get collection from storage")
	}

	for _, rawInst := range resp.Kvs {
		i, err := s.decodeInstance(rawInst.Value)
		if err != nil {
			return nil, errors.Wrap(err, "while decoding DSO collection")
		}
		out = append(out, i)
	}

	return out, nil
}

func (s *Instance) decodeInstance(raw []byte) (*internal.Instance, error) {
	dec := gob.NewDecoder(bytes.NewReader(raw))
	var i internal.Instance
	if err := dec.Decode(&i); err != nil {
		return nil, err
	}

	return &i, nil
}

// Remove removing object from storage.
func (s *Instance) Remove(id internal.InstanceID) error {
	resp, err := s.kv.Delete(context.TODO(), s.key(id))
	if err != nil {
		return errors.Wrap(err, "while calling database")
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

func (*Instance) key(id internal.InstanceID) string {
	return string(id)
}
