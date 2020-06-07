package etcd

import (
	"go.etcd.io/etcd/clientv3"
)

// TODO list:
// - Use etcd lease for garbage collection of removed elements.
//   Create lease on element delete and attach it to each object which should be deleted.

const (
	entityNamespaceSeparator   = "/"
	entityOperationIDSeparator = "/"

	entityNamespaceAddon          = "addon"
	entityNamespaceAddonMappingID = "id"
	entityNamespaceAddonMappingNV = "nv"

	entityNamespaceChart             = "chart"
	entityNamespaceInstance          = "instance/"
	entityNamespaceInstanceOperation = "instanceOperation/"
	entityNamespaceBindOperation     = "bindOperation/"
)

// Config holds configuration for etcd access in storage.
type Config struct {
	Endpoints            []string `json:"endpoints"`
	Username             string   `json:"username"`
	Password             string   `json:"password"`
	DialTimeout          string   `json:"dialTimeout" default:"5s"`
	DialKeepAliveTime    string   `json:"dialKeepAliveTime" default:"2s"`
	DialKeepAliveTimeout string   `json:"dialKeepAliveTimeout" default:"5s"`

	ForceClient *clientv3.Client
}

func entityNamespacePrefixParts() []string {
	return []string{"helm-broker", "entity"}
}

// generic is a foundation for all drivers using etcd as storage.
type generic struct {
	kv clientv3.KV
}
