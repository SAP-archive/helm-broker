package etcd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/namespace"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/kyma-project/helm-broker/internal"
)

// NewChart creates new storage for Charts
func NewChart(cli clientv3.KV) (*Chart, error) {

	prefixParts := append(entityNamespacePrefixParts(), string(entityNamespaceChart))
	kv := namespace.NewKV(cli, strings.Join(prefixParts, entityNamespaceSeparator))

	d := &Chart{
		generic: generic{
			kv: kv,
		},
	}

	return d, nil
}

// Chart provides storage operations on Chart entity
type Chart struct {
	generic
}

// Upsert persists Chart in memory.
//
// If chart already exists in storage then full replace is performed.
//
// Replace is set to true if chart already existed in storage and was replaced.
func (s *Chart) Upsert(namespace internal.Namespace, c *chart.Chart) (replaced bool, err error) {
	nv, err := s.nameVersionFromChart(c)
	if err != nil {
		return false, err
	}
	encoded, err := s.encodeChart(c)
	if err != nil {
		return false, errors.Wrap(err, "while encoding chart")
	}

	resp, err := s.kv.Put(context.TODO(), s.key(namespace, nv), encoded, clientv3.WithPrevKV())
	if err != nil {
		return false, errors.Wrap(err, "while calling database")
	}

	if resp.PrevKv != nil {
		return true, nil
	}

	return false, nil
}

// Get returns chart with given name and version from storage
func (s *Chart) Get(namespace internal.Namespace, name internal.ChartName, ver semver.Version) (*chart.Chart, error) {
	nv, err := s.nameVersion(name, ver)
	if err != nil {
		return nil, err
	}

	resp, err := s.kv.Get(context.TODO(), s.key(namespace, nv))
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

	c, err := s.decodeChart(resp.Kvs[0].Value)
	if err != nil {
		return nil, errors.Wrap(err, "while decoding single DSO")
	}

	return c, nil
}

// Remove is removing chart with given name and version from storage
func (s *Chart) Remove(namespace internal.Namespace, name internal.ChartName, ver semver.Version) error {
	nv, err := s.nameVersion(name, ver)
	if err != nil {
		return errors.Wrap(err, "while getting nameVersion from deleted entity")
	}

	resp, err := s.kv.Delete(context.TODO(), s.key(namespace, nv))
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

type chartNameVersion string

func (s *Chart) nameVersionFromChart(c *chart.Chart) (k chartNameVersion, err error) {
	if c == nil {
		return k, errors.New("entity may not be nil")
	}

	if c.Metadata == nil {
		return k, errors.New("entity metadata may not be nil")
	}

	if c.Metadata.Name == "" || c.Metadata.Version == "" {
		return k, errors.New("both name and version must be set")
	}

	ver, err := semver.NewVersion(c.Metadata.Version)
	if err != nil {
		return k, errors.Wrap(err, "while parsing version")
	}

	return s.nameVersion(internal.ChartName(c.Metadata.Name), *ver)
}

func (*Chart) nameVersion(name internal.ChartName, ver semver.Version) (k chartNameVersion, err error) {
	if name == "" || ver.Original() == "" {
		return k, errors.New("both name and version must be set")
	}

	return chartNameVersion(fmt.Sprintf("%s|%s", name, ver.Original())), nil
}

func (*Chart) key(namespace internal.Namespace, nv chartNameVersion) string {
	prefix := ""
	if namespace == internal.ClusterWide {
		prefix = "cluster"
	} else {
		prefix = fmt.Sprintf("ns|%s", string(namespace))
	}
	return fmt.Sprintf("%s|%s", prefix, string(nv))
}

type dto struct {
	Main *chart.Chart `json:"main"`
	Deps []*dto       `json:"dependencies"`
}

func (s *Chart) toDto(c *chart.Chart) *dto {
	var deps []*dto
	for _, d := range c.Dependencies() {
		deps = append(deps, s.toDto(d))
	}
	return &dto{
		Main: c,
		Deps: deps,
	}
}

func (s *Chart) fromDto(obj *dto) *chart.Chart {
	chrt := obj.Main

	deps := make([]*chart.Chart, len(obj.Deps))
	for i, d := range obj.Deps {
		deps[i] = s.fromDto(d)
	}
	chrt.SetDependencies(deps...)
	return chrt
}

func (s *Chart) encodeChart(c *chart.Chart) (string, error) {
	obj := s.toDto(c)
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(obj); err != nil {
		return "", errors.Wrap(err, "while encoding entity")
	}
	return buf.String(), nil
}

func (s *Chart) decodeChart(raw []byte) (*chart.Chart, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var obj dto
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	if obj.Main == nil {
		return nil, errors.Errorf("chart cannot be nil: %s", string(raw))
	}

	return s.fromDto(&obj), nil
}
