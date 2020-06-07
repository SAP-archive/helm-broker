package etcd

import (
	"time"

	"go.etcd.io/etcd/clientv3"
)

// Client wraps etcd client for testing purposes.
type Client interface {
	clientv3.KV
}

// NewClient produces new, configured etcd client.
func NewClient(cfg Config) (Client, error) {
	dialTimeout, err := time.ParseDuration(cfg.DialTimeout)
	if err != nil {
		return nil, err
	}
	dialKeepAliveTime, err := time.ParseDuration(cfg.DialKeepAliveTime)
	if err != nil {
		return nil, err
	}
	dialKeepAliveTimeout, err := time.ParseDuration(cfg.DialKeepAliveTimeout)
	if err != nil {
		return nil, err
	}

	etcdCfg := clientv3.Config{
		Endpoints:            cfg.Endpoints,
		Username:             cfg.Username,
		Password:             cfg.Password,
		DialTimeout:          dialTimeout,
		DialKeepAliveTime:    dialKeepAliveTime,
		DialKeepAliveTimeout: dialKeepAliveTimeout,
	}

	cli, err := clientv3.New(etcdCfg)

	return cli, err
}
